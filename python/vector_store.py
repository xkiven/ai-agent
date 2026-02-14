"""
向量存储服务 (Milvus + Embedding)
"""

import os
from typing import List, Optional, Dict, Any
from config import config


class EmbeddingService:
    """Embedding 服务封装"""

    def __init__(self):
        self.api_key = config.embedding.api_key if config.embedding else ""
        self.base_url = config.embedding.base_url if config.embedding else ""
        self.model = config.embedding.model if config.embedding else "text-embedding-v3"
        self.dimension = config.embedding.dimension if config.embedding else 1024

    def embed_text(self, text: str) -> List[float]:
        """将文本转换为向量"""
        if not self.api_key:
            raise ValueError("Embedding API key not configured")

        try:
            import dashscope
            dashscope.api_key = self.api_key

            from dashscope import TextEmbedding
            response = TextEmbedding.call(
                model=self.model,
                input=text
            )

            if response.status_code == 200:
                return response.output['embeddings'][0]['embedding']
            else:
                raise Exception(f"Embedding API error: {response.code} - {response.message}")

        except ImportError:
            raise ImportError("dashscope not installed. Run: pip install dashscope")

    def embed_texts_batch(self, texts: List[str]) -> List[List[float]]:
        """批量将文本转换为向量"""
        if not self.api_key:
            raise ValueError("Embedding API key not configured")

        try:
            import dashscope
            dashscope.api_key = self.api_key

            from dashscope import TextEmbedding
            response = TextEmbedding.call(
                model=self.model,
                input=texts
            )

            if response.status_code == 200:
                return [item['embedding'] for item in response.output['embeddings']]
            else:
                raise Exception(f"Embedding API error: {response.code} - {response.message}")

        except ImportError:
            raise ImportError("dashscope not installed. Run: pip install dashscope")


class MilvusStore:
    """Milvus 向量库封装"""

    def __init__(self):
        if not config.milvus:
            raise ValueError("Milvus not configured")

        self.host = config.milvus.host
        self.port = config.milvus.port
        self.collection_name = config.milvus.collection_name
        self.dimension = config.embedding.dimension if config.embedding else 1024
        self.client = None
        self.collection = None

    def connect(self):
        """连接 Milvus"""
        try:
            from pymilvus import connections
            connections.connect(
                host=self.host,
                port=self.port,
                user=config.milvus.user,
                password=config.milvus.password
            )
            print(f"Connected to Milvus at {self.host}:{self.port}")
        except ImportError:
            raise ImportError("pymilvus not installed. Run: pip install pymilvus")

    def create_collection(self, if_not_exists: bool = True):
        """创建 Collection"""
        from pymilvus import Collection, CollectionSchema, FieldSchema, DataType

        fields = [
            FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=True),
            FieldSchema(name="text", dtype=DataType.VARCHAR, max_length=65535),
            FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=self.dimension),
            FieldSchema(name="metadata", dtype=DataType.JSON, nullable=True),
        ]

        schema = CollectionSchema(fields=fields, description="Knowledge base collection")
        self.collection = Collection(name=self.collection_name, schema=schema, using="default")

        # 创建索引
        index_params = {
            "index_type": "IVF_FLAT",
            "metric_type": "COSINE",
            "params": {"nlist": 128}
        }
        self.collection.create_index(field_name="vector", index_params=index_params)
        print(f"Collection '{self.collection_name}' created/loaded")

    def load_collection(self):
        """加载 Collection"""
        from pymilvus import Collection
        self.collection = Collection(name=self.collection_name, using="default")
        self.collection.load()
        print(f"Collection '{self.collection_name}' loaded")

    def search(
        self,
        query_vector: List[float],
        top_k: int = 5,
        filter_expr: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """向量检索"""
        if not self.collection:
            self.load_collection()

        search_params = {"metric_type": "COSINE", "params": {"nprobe": 10}}

        results = self.collection.search(
            data=[query_vector],
            anns_field="vector",
            param=search_params,
            limit=top_k,
            expr=filter_expr,
            output_fields=["text", "metadata"]
        )

        return [
            {
                "text": hit.entity.get("text"),
                "metadata": hit.entity.get("metadata"),
                "distance": hit.distance
            }
            for hit in results[0]
        ]

    def add_texts(self, texts: List[str], metadata: Optional[List[Dict]] = None):
        """添加文本到向量库"""
        if not self.collection:
            self.create_collection()

        embedding_service = EmbeddingService()
        vectors = embedding_service.embed_texts(texts)

        data = []
        for i, (text, vector) in enumerate(zip(texts, vectors)):
            item = {
                "text": text,
                "vector": vector,
                "metadata": metadata[i] if metadata else None
            }
            data.append(item)

        self.collection.insert(data)
        self.collection.flush()
        print(f"Added {len(texts)} texts to collection")

    def delete_all(self):
        """删除所有数据"""
        if self.collection:
            self.collection.delete(expr="id >= 0")
            self.collection.flush()
            print("All data deleted")

    def close(self):
        """关闭连接"""
        from pymilvus import connections
        connections.disconnect("default")


# 全局实例
_embedding_service: Optional[EmbeddingService] = None
_milvus_store: Optional[MilvusStore] = None


def get_embedding_service() -> EmbeddingService:
    """获取 Embedding 服务实例"""
    global _embedding_service
    if _embedding_service is None:
        _embedding_service = EmbeddingService()
    return _embedding_service


def get_milvus_store() -> MilvusStore:
    """获取 Milvus 向量库实例"""
    global _milvus_store
    if _milvus_store is None:
        _milvus_store = MilvusStore()
        _milvus_store.connect()
    return _milvus_store
