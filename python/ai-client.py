from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import uuid
from datetime import datetime
import json
import os
import requests

app = FastAPI()

# 配置API
OPENAI_API_KEY =  "sk-c3c62b663de04038b76d2f444efbc979"
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
API_MODEL = os.getenv("API_MODEL", "qwen-plus")

class Message(BaseModel):
    role: str
    content: str

class ChatRequest(BaseModel):
    session_id: str
    message: str
    user_id: Optional[str] = None
    history: Optional[List[Message]] = None
    intent: Optional[str] = None  # 由Go传入的意图类型
    flow_id: Optional[str] = None  # 由Go传入的流程ID


class ChatResponse(BaseModel):
    reply: str
    type: str
    session_state: Optional[str] = None

class IntentRecognitionRequest(BaseModel):
    message: str
    session_id: str
    history: Optional[List[Message]] = None

class IntentRecognitionResponse(BaseModel):
    intent: str
    confidence: float
    reply: Optional[str] = None
    flow_id: Optional[str] = None
    suggestions: Optional[List[str]] = None

class Ticket(BaseModel):
    id: str
    session_id: str
    user_id: str
    intent: str
    subject: Optional[str] = None
    description: str
    status: str
    created_at: str
    updated_at: str

# 使用OpenAI API进行意图识别
def recognize_intent_with_ai(message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """使用OpenAI API进行意图识别"""
    
    # 构建系统提示
    system_prompt = """
你是一个专业的意图识别助手。请根据用户的消息，识别其意图类型。

意图类型包括：
1. faq - 常见问题，用户询问关于产品、服务的一般性问题
2. flow - 流程相关，用户需要执行某个具体流程或操作步骤
3. unknown - 未知意图，无法明确分类的问题

请以JSON格式返回识别结果，格式如下：
{
    "intent": "意图类型(faq/flow/unknown)",
    "confidence": 置信度(0-1之间的浮点数),
    "reply": "针对该意图的简单回复",
    "flow_id": "如果是flow类型，提供流程ID(可选)",
    "suggestions": ["如果是flow类型，提供步骤建议(可选)"]
}

注意：
- 退货、退款、注册、登录等操作流程应识别为flow
- 如何、什么、为什么等一般性问题应识别为faq
- 如果无法确定，请返回unknown
"""
    
    # 构建消息列表
    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": f"用户消息: {message}"}
    ]

    # 如果有历史记录，添加到消息中
    if history:
        print(f"添加历史记录: {len(history)} 条消息")
    # 在用户消息前添加历史记录
        for msg in history:
            messages.insert(-1, {"role": msg.role, "content": msg.content})
    else:
        print("没有历史记录")

    try:
        # 调用OpenAI API
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {OPENAI_API_KEY}"
        }
        
        data = {
            "model": API_MODEL,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 500
        }
        
        response = requests.post(
            f"{OPENAI_BASE_URL}/chat/completions",
            headers=headers,
            json=data,
            timeout=30
        )
        
        if response.status_code != 200:
            raise Exception(f"API请求失败: {response.status_code}, {response.text}")
        
        result = response.json()
        content = result["choices"][0]["message"]["content"]
        
        # 尝试解析JSON响应
        try:
            intent_data = json.loads(content)
            return IntentRecognitionResponse(**intent_data)
        except json.JSONDecodeError:
            # 如果JSON解析失败，返回默认响应
            return IntentRecognitionResponse(
                intent="unknown",
                confidence=0.5,
                reply="意图识别失败，请稍后再试。"
            )
            
    except Exception as e:
        print(f"意图识别错误: {str(e)}")
        # 返回默认响应
        return IntentRecognitionResponse(
            intent="unknown",
            confidence=0.5,
            reply="意图识别服务暂时不可用，请稍后再试。"
        )

# 基于规则的备用意图识别（当AI不可用时）
def recognize_intent_fallback(message: str) -> IntentRecognitionResponse:
    """基于规则的备用意图识别"""
    message_lower = message.lower()
    
    # 退货/退款流程意图识别 - 优先级最高
    if any(word in message_lower for word in ["退货", "退款", "怎么退", "如何退"]):
        print(f"识别到退货/退款流程: {message}")
        return IntentRecognitionResponse(
            intent="flow",
            confidence=0.9,
            flow_id="refund_flow",
            reply="退货流程已识别，后续将接入AI处理具体步骤"
        )
    
    # 注册流程意图识别
    elif any(word in message_lower for word in ["注册", "怎么注册", "如何注册"]):
        return IntentRecognitionResponse(
            intent="flow",
            confidence=0.9,
            flow_id="register_flow",
            reply="注册流程已识别，后续将接入AI处理具体步骤"
        )
    
    # FAQ意图识别 - 识别常见问题
    elif any(word in message_lower for word in ["如何", "怎么", "什么", "为什么"]):
        return IntentRecognitionResponse(
            intent="faq",
            confidence=0.85,
            reply="FAQ问题已识别，后续将接入AI处理具体回答"
        )
    
    # 未知意图
    else:
        return IntentRecognitionResponse(
            intent="unknown",
            confidence=0.5,
            reply="未知意图，后续将接入AI处理"
        )

# 意图识别主函数
def recognize_intent(message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """意图识别主函数，优先使用AI，失败时使用规则"""
    print(f"开始意图识别: {message}")
    try:
        # 尝试使用AI进行意图识别
        result = recognize_intent_with_ai(message, history)
        print(f"AI意图识别结果: {result}")
        return result
    except Exception as e:
        print(f"AI意图识别失败，使用备用方案: {str(e)}")
        # AI失败时使用基于规则的备用方案
        result = recognize_intent_fallback(message)
        print(f"备用规则识别结果: {result}")
        return result

def generate_reply(message: str, intent: str, flow_id: Optional[str] = None, history: Optional[List[Message]] = None) -> str:
    if intent == "flow":
        return f"【流程处理】你刚才说的是：{message}"
    elif intent == "faq":
        return f"【FAQ回答】你刚才问的是：{message}"
    else:
        return f"【默认回复】我收到你的消息：{message}"


@app.post("/chat", response_model=ChatResponse)
def chat(req: ChatRequest):
    """根据传入的意图和历史记录生成回复，不再进行意图识别"""
    print(f"收到聊天请求: message={req.message}, intent={req.intent}, flow_id={req.flow_id}")

    # 生成回复
    reply = generate_reply(req.message, req.intent or "unknown", req.flow_id, req.history)

    # 根据意图类型确定会话状态
    if req.intent == "flow":
        session_state = "on_flow"
    elif req.intent == "faq":
        session_state = "active"
    else:
        session_state = "new"

    return ChatResponse(
        reply=reply,
        type=req.intent or "unknown",
        session_state=session_state
    )

@app.post("/intent/recognize", response_model=IntentRecognitionResponse)
def recognize_intent_endpoint(req: IntentRecognitionRequest):
    """只做意图识别，不生成回复"""
    return recognize_intent(req.message, req.history)

@app.post("/ticket/create", response_model=Ticket)
def create_ticket(ticket: Ticket):
    # 简单返回工单信息
    now = datetime.now().isoformat()
    ticket.id = str(uuid.uuid4())
    ticket.created_at = now
    ticket.updated_at = now
    ticket.status = "open"
    ticket.intent = "unknown"
    
    return ticket

@app.get("/health")
def health_check():
    return {"status": "ok", "message": "AI意图识别服务运行正常"}