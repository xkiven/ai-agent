#!/usr/bin/env python3
"""
AI Client 主应用入口
"""

from fastapi import FastAPI
from api import router


def create_app() -> FastAPI:
    """创建FastAPI应用"""
    app = FastAPI(
        title="AI Client",
        description="AI对话客户端",
        version="1.0.0"
    )
    
    # 注册路由
    app.include_router(router)
    
    return app


app = create_app()


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)