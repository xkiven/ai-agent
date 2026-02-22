"""
API路由处理
"""

import os
import uuid
from datetime import datetime
from typing import List, Optional, Dict, Any

os.environ.setdefault("LLM_API_KEY", "sk-c3c62b663de04038b76d2f444efbc979")

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from models import ChatRequest, ChatResponse, IntentRecognitionRequest, IntentRecognitionResponse, Ticket, InterruptCheckRequest, InterruptCheckResponse
from services import ChatService, init_intent_vector_service
from knowledge_store import init_knowledge_store, get_knowledge_store


router = APIRouter()

# 初始化意图向量服务
init_intent_vector_service()

# 初始化知识库服务
init_knowledge_store()

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


class ListKnowledgeResponse(BaseModel):
    """知识列表响应"""
    success: bool
    data: List[Dict[str, Any]]
    total: int
    message: str


@router.get("/knowledge/list", response_model=ListKnowledgeResponse)
def list_knowledge_endpoint(limit: int = 100, offset: int = 0):
    """获取知识列表"""
    try:
        store = get_knowledge_store()
        if not store:
            raise HTTPException(status_code=500, detail="知识库未初始化")
        data = store.list_knowledge(limit, offset)
        total = store.count()
        return ListKnowledgeResponse(
            success=True,
            data=data,
            total=total,
            message=f"获取成功，共 {total} 条知识"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


class DeleteKnowledgeResponse(BaseModel):
    """删除知识响应"""
    success: bool
    message: str


@router.delete("/knowledge/delete", response_model=DeleteKnowledgeResponse)
def delete_knowledge_endpoint(index: int):
    """删除指定索引的知识"""
    try:
        store = get_knowledge_store()
        if not store:
            raise HTTPException(status_code=500, detail="知识库未初始化")
        success = store.delete(index)
        if success:
            return DeleteKnowledgeResponse(success=True, message=f"成功删除索引 {index} 的知识")
        else:
            raise HTTPException(status_code=404, detail=f"索引 {index} 不存在")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/knowledge/clear", response_model=DeleteKnowledgeResponse)
def clear_knowledge_endpoint():
    """清空知识库"""
    try:
        store = get_knowledge_store()
        if not store:
            raise HTTPException(status_code=500, detail="知识库未初始化")
        store.delete_all()
        return DeleteKnowledgeResponse(success=True, message="知识库已清空")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


class CountKnowledgeResponse(BaseModel):
    """知识数量响应"""
    success: bool
    count: int
    message: str


@router.get("/knowledge/count", response_model=CountKnowledgeResponse)
def count_knowledge_endpoint():
    """获取知识数量"""
    try:
        store = get_knowledge_store()
        if not store:
            raise HTTPException(status_code=500, detail="知识库未初始化")
        count = store.count()
        return CountKnowledgeResponse(
            success=True,
            count=count,
            message=f"知识库共有 {count} 条知识"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))