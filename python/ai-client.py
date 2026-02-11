from fastapi import FastAPI
from pydantic import BaseModel
from typing import List, Optional
import uuid
from datetime import datetime

app = FastAPI()

class Message(BaseModel):
    role: str
    content: str

class ChatRequest(BaseModel):
    session_id: str
    message: str
    user_id: Optional[str] = None

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

# 简单的意图识别逻辑
def recognize_intent(message: str) -> IntentRecognitionResponse:
    message_lower = message.lower()
    
    # FAQ意图识别 - 识别常见问题
    if any(word in message_lower for word in ["如何", "怎么", "什么", "为什么"]):
        return IntentRecognitionResponse(
            intent="faq",
            confidence=0.85,
            reply=f"这是一个常见问题。关于'{message}'，我们的标准答案是：这是一个常见问题的回答。"
        )
    
    # Flow意图识别 - 识别流程相关
    elif any(word in message_lower for word in ["流程", "步骤", "退货", "退款", "注册"]):
        return IntentRecognitionResponse(
            intent="flow",
            confidence=0.8,
            flow_id="general_flow",
            reply=f"我将引导您完成相关流程，请按照以下步骤操作：",
            suggestions=["步骤1: 准备相关信息", "步骤2: 提交申请", "步骤3: 等待处理", "步骤4: 完成操作"]
        )
    
    # 未知意图
    else:
        return IntentRecognitionResponse(
            intent="unknown",
            confidence=0.5,
            reply="抱歉，我无法理解您的问题。"
        )

@app.post("/chat", response_model=ChatResponse)
def chat(req: ChatRequest):
    # 识别意图
    intent_resp = recognize_intent(req.message)
    
    # 根据意图类型确定会话状态
    if intent_resp.intent == "flow":
        session_state = "on_flow"
    elif intent_resp.intent == "faq":
        session_state = "active"
    else:
        session_state = "new"
    
    # 构建回复
    reply = intent_resp.reply
    if intent_resp.intent == "flow" and intent_resp.suggestions:
        reply += "\n\n" + "\n".join([f"{i+1}. {step}" for i, step in enumerate(intent_resp.suggestions)])
    
    return ChatResponse(
        reply=reply,
        type=intent_resp.intent,
        session_state=session_state
    )

@app.post("/intent/recognize", response_model=IntentRecognitionResponse)
def recognize_intent_endpoint(req: IntentRecognitionRequest):
    return recognize_intent(req.message)

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