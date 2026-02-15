"""
配置管理
"""

import os
from typing import Optional
from pydantic import BaseModel


class LLMConfig(BaseModel):
    """LLM配置"""
    api_key: str
    base_url: str = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    model: str = "qwen-plus"
    temperature: float = 0.7
    max_tokens: int = 500
    timeout: int = 30


class EmbeddingConfig(BaseModel):
    """Embedding配置"""
    api_key: str
    base_url: str = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    model: str = "text-embedding-v3"
    dimension: int = 1024


class MilvusConfig(BaseModel):
    """Milvus向量库配置"""
    host: str = "localhost"
    port: int = 19530
    user: str = ""
    password: str = ""
    database: str = "default"
    collection_name: str = "knowledge_base"


class Config(BaseModel):
    """全局配置"""
    llm: LLMConfig
    embedding: Optional[EmbeddingConfig] = None
    milvus: Optional[MilvusConfig] = None
    log_level: str = "INFO"


def load_config() -> Config:
    """从环境变量加载配置"""
    llm_key = os.getenv("LLM_API_KEY", "")
    return Config(
        llm=LLMConfig(
            api_key=llm_key,
            base_url=os.getenv("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
            model=os.getenv("LLM_MODEL", "qwen-plus"),
            temperature=float(os.getenv("LLM_TEMPERATURE", "0.7")),
            max_tokens=int(os.getenv("LLM_MAX_TOKENS", "500")),
            timeout=int(os.getenv("LLM_TIMEOUT", "30")),
        ),
        embedding=EmbeddingConfig(
            api_key=os.getenv("EMBEDDING_API_KEY", llm_key),
            base_url=os.getenv("EMBEDDING_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
            model=os.getenv("EMBEDDING_MODEL", "text-embedding-v3"),
            dimension=int(os.getenv("EMBEDDING_DIMENSION", "1024")),
        ) if llm_key else None,
        milvus=MilvusConfig(
            host=os.getenv("MILVUS_HOST", "localhost"),
            port=int(os.getenv("MILVUS_PORT", "19530")),
            user=os.getenv("MILVUS_USER", ""),
            password=os.getenv("MILVUS_PASSWORD", ""),
            database=os.getenv("MILVUS_DATABASE", "default"),
            collection_name=os.getenv("MILVUS_COLLECTION", "knowledge_base"),
        ) if os.getenv("MILVUS_HOST") else None,
        log_level=os.getenv("LOG_LEVEL", "INFO"),
    )


config = load_config()
