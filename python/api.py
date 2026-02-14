"""
API路由处理
"""

import uuid
from datetime import datetime
from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from models import ChatRequest, ChatResponse, IntentRecognitionRequest, IntentRecognitionResponse, Ticket, InterruptCheckRequest, InterruptCheckResponse
from services import ChatService, init_intent_vector_service


router = APIRouter()

# 初始化意图向量服务
init_intent_vector_service()

chat_service = ChatService()


@router.post("/chat", response_model=ChatResponse)
def chat_endpoint(req: ChatRequest):
    """根据传入的意图和历史记录生成回复，不再进行意图识别"""
    print(f"收到聊天请求: message={req.message}, intent={req.intent}, flow_id={req.flow_id}")

    # 生成回复
    reply = chat_service.generate_reply(req.message, req.intent or "unknown", req.flow_id, req.history)

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


@router.post("/intent/recognize", response_model=IntentRecognitionResponse)
def recognize_intent_endpoint(req: IntentRecognitionRequest):
    """只做意图识别，不生成回复"""
    return chat_service.recognize_intent(req.message, req.history)


@router.post("/ticket/create", response_model=Ticket)
def create_ticket_endpoint(ticket: Ticket):
    """创建工单"""
    # 简单返回工单信息
    now = datetime.now().isoformat()
    ticket.id = str(uuid.uuid4())
    ticket.created_at = now
    ticket.updated_at = now
    ticket.status = "open"
    ticket.intent = "unknown"
    
    return ticket


@router.post("/flow/interrupt-check", response_model=InterruptCheckResponse)
def check_flow_interrupt_endpoint(request: InterruptCheckRequest):
    """检查是否应该打断当前Flow"""
    return chat_service.check_flow_interrupt(request)


class AddKnowledgeRequest(BaseModel):
    """知识入库请求"""
    texts: List[str]
    metadata: Optional[List[Dict[str, Any]]] = None


class AddKnowledgeResponse(BaseModel):
    """知识入库响应"""
    success: bool
    count: int
    message: str


@router.post("/knowledge/add", response_model=AddKnowledgeResponse)
def add_knowledge_endpoint(request: AddKnowledgeRequest):
    """添加知识到向量库"""
    try:
        chat_service.add_knowledge(request.texts, request.metadata)
        return AddKnowledgeResponse(
            success=True,
            count=len(request.texts),
            message=f"成功添加 {len(request.texts)} 条知识"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))