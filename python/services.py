"""
业务逻辑服务
"""

import json
import os
import requests
from typing import List, Optional
from models import Message, IntentRecognitionResponse, InterruptCheckRequest, InterruptCheckResponse


# 配置
OPENAI_API_KEY = "sk-c3c62b663de04038b76d2f444efbc979"
OPENAI_BASE_URL = os.getenv("OPENAI_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
API_MODEL = os.getenv("API_MODEL", "qwen-plus")


class ChatService:
    
    def __init__(self):
        """初始化聊天服务"""
        self.openai_api_key = "sk-c3c62b663de04038b76d2f444efbc979"
        self.openai_base_url = os.getenv("OPENAI_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1")
        self.api_model = os.getenv("API_MODEL", "qwen-plus")
    
    def recognize_intent_with_ai(self, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
        """使用OpenAI API进行意图识别"""
        return _recognize_intent_with_ai(self, message, history)
    
    def recognize_intent_fallback(self, message: str) -> IntentRecognitionResponse:
        """基于规则的备用意图识别（当AI不可用时）"""
        return _recognize_intent_fallback(message)
    
    def recognize_intent(self, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
        """意图识别主函数，优先使用AI，失败时使用规则"""
        return _recognize_intent(self, message, history)
    
    def generate_reply(self, message: str, intent: str, flow_id: Optional[str] = None, history: Optional[List[Message]] = None) -> str:
        """根据意图和上下文生成回复"""
        return _generate_reply(self, message, intent, flow_id, history)
    
    def check_flow_interrupt(self, request: InterruptCheckRequest) -> InterruptCheckResponse:
        """检查是否应该打断当前Flow"""
        return _check_flow_interrupt(self, request)




def _recognize_intent_with_ai(chat_service: ChatService, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """使用OpenAI API进行意图识别 - 具体实现"""
    
    # 构建系统提示
    system_prompt = """
你是一个专业的意图识别助手。请根据用户的消息，识别其意图类型，不说废话，置信度在0-1之间。

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
        for msg in history:
            messages.insert(-1, {"role": msg.role, "content": msg.content})
    else:
        print("没有历史记录")

    try:
        # 调用OpenAI API
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {chat_service.openai_api_key}"
        }
        
        data = {
            "model": chat_service.api_model,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 500
        }
        
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
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


def _recognize_intent_fallback(message: str) -> IntentRecognitionResponse:
    """基于规则的备用意图识别（当AI不可用时） - 具体实现"""
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


def _recognize_intent(chat_service: ChatService, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """意图识别主函数，优先使用AI，失败时使用规则 - 具体实现"""
    print(f"开始意图识别: {message}")
    try:
        # 尝试使用AI进行意图识别
        result = chat_service.recognize_intent_with_ai(message, history)
        print(f"AI意图识别结果: {result}")
        return result
    except Exception as e:
        print(f"AI意图识别失败，使用备用方案: {str(e)}")
        # AI失败时使用基于规则的备用方案
        result = chat_service.recognize_intent_fallback(message)
        print(f"备用规则识别结果: {result}")
        return result


def _generate_reply(chat_service: ChatService, message: str, intent: str, flow_id: Optional[str] = None, history: Optional[List[Message]] = None) -> str:
    """根据意图和上下文生成回复 - 具体实现"""
    system_prompt = f""" 
你是一个智能客服助手，不说废话，直接回答用户问题。 
当前意图类型: {intent} 
当前流程ID: {flow_id} 

规则： 
- 如果是 flow，请引导用户继续完成该流程 
- 如果是 faq，请直接回答用户问题 
- 如果是 unknown，请礼貌说明并建议转人工 
"""

    messages = [
        {"role": "system", "content": system_prompt},
    ]

    # 加入历史对话
    if history:
        for msg in history:
            messages.append({"role": msg.role, "content": msg.content})

    # 加入当前用户消息
    messages.append({"role": "user", "content": message})

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {chat_service.openai_api_key}"
    }

    data = {
        "model": chat_service.api_model,
        "messages": messages,
        "temperature": 0.7,
        "max_tokens": 500
    }

    try:
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
            headers=headers,
            json=data,
            timeout=30
        )

        if response.status_code != 200:
            raise Exception(f"API请求失败: {response.status_code}, {response.text}")

        result = response.json()
        return result["choices"][0]["message"]["content"]

    except Exception as e:
        print(f"生成回复失败: {e}")
        return "抱歉，当前系统繁忙，请稍后再试。"


def _check_flow_interrupt(chat_service: ChatService, request: InterruptCheckRequest) -> InterruptCheckResponse:
    """检查是否应该打断当前Flow - 具体实现"""
    print(f"Flow中断检查: session={request.session_id}, flow={request.flow_id}, step={request.current_step}")
    
    # 构建系统提示
    system_prompt = f"""
你是一个Flow中断判断助手。请根据用户在当前流程中的输入，判断是否应该打断当前流程。

当前流程信息：
- 流程ID: {request.flow_id}
- 当前步骤: {request.current_step}
- 流程状态: {request.flow_state}

用户输入: {request.user_message}

判断规则：
1. 如果用户明确表示要退出、取消、停止当前流程，应该打断
2. 如果用户询问与当前流程无关的问题，应该打断
3. 如果用户输入包含明显的错误或误解，应该打断
4. 如果用户只是继续当前流程的正常操作，不应该打断

请以JSON格式返回判断结果，格式如下：
{{
    "should_interrupt": true/false,
    "confidence": 0-1之间的置信度,
    "new_intent": "如果打断，建议的新意图类型(可选)",
    "reason": "判断理由(可选)"
}}
"""

    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": f"用户输入: {request.user_message}"}
    ]

    try:
        # 调用OpenAI API进行中断判断
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {chat_service.openai_api_key}"
        }
        
        data = {
            "model": chat_service.api_model,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 300
        }
        
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
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
            interrupt_data = json.loads(content)
            return InterruptCheckResponse(**interrupt_data)
        except json.JSONDecodeError:
            # 如果JSON解析失败，返回默认响应（不打断）
            print("Flow中断检查JSON解析失败，默认不打断")
            return InterruptCheckResponse(
                should_interrupt=False,
                confidence=0.5,
                reason="解析失败，默认继续流程"
            )
            
    except Exception as e:
        print(f"Flow中断检查错误: {str(e)}")
        # 出错时默认不打断
        return InterruptCheckResponse(
            should_interrupt=False,
            confidence=0.3,
            reason=f"检查失败: {str(e)}"
        )