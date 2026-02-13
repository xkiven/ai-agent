"""
数据模型定义
"""

from pydantic import BaseModel
from typing import List, Optional


class Message(BaseModel):
    """消息模型"""
    role: str
    content: str


class ChatRequest(BaseModel):
    """聊天请求模型"""
    session_id: str
    message: str
    user_id: Optional[str] = None
    history: Optional[List[Message]] = None
    intent: Optional[str] = None  # 由Go传入的意图类型
    flow_id: Optional[str] = None  # 由Go传入的流程ID


class ChatResponse(BaseModel):
    """聊天响应模型"""
    reply: str
    type: str
    session_state: Optional[str] = None


class IntentRecognitionRequest(BaseModel):
    """意图识别请求模型"""
    message: str
    session_id: str
    history: Optional[List[Message]] = None


class IntentRecognitionResponse(BaseModel):
    """意图识别响应模型"""
    intent: str
    confidence: float
    reply: Optional[str] = None
    flow_id: Optional[str] = None
    suggestions: Optional[List[str]] = None


class Ticket(BaseModel):
    """工单模型"""
    id: str
    session_id: str
    user_id: str
    intent: str
    subject: Optional[str] = None
    description: str
    status: str
    created_at: str
    updated_at: str


class InterruptCheckRequest(BaseModel):
    """Flow中断检查请求模型"""
    session_id: str
    flow_id: str
    current_step: Optional[str] = None
    user_message: str
    flow_state: Optional[Dict[str, Any]] = None


class InterruptCheckResponse(BaseModel):
    """Flow中断检查响应模型"""
    should_interrupt: bool
    confidence: float
    new_intent: Optional[str] = None
    reason: Optional[str] = None