from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class ChatRequest(BaseModel):
    session_id: str
    message: str

class ChatResponse(BaseModel):
    reply: str
    type: str

@app.post("/chat", response_model=ChatResponse)
def chat(req: ChatRequest):
    # 这里就是“伪 AI 逻辑”
    if "怎么" in req.message or "如何" in req.message:
        return ChatResponse(
            reply=f"你是在问具体怎么操作吗？可以再详细 说一下吗？（你刚说：{req.message}）",
            type="ask",
        )
    else:
        return ChatResponse(
            reply=f"【Python AI 回复】我收到了你的消息：{req.message}",
            type="mock",
        )