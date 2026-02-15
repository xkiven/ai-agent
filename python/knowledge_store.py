"""
知识库向量服务 - 基于FAISS的RAG知识检索
"""

import numpy as np
from typing import List, Dict, Optional, Any
import pickle
import os

import faiss

from embedding_service import EmbeddingService


INDEX_FILE = "./knowledge_faiss_index.bin"
METADATA_FILE = "./knowledge_metadata.pkl"


class KnowledgeStore:
    """知识库向量存储服务"""

    def __init__(self, embedding_service: EmbeddingService):
        self.embedding_service = embedding_service
        self.index = None
        self.metadata: List[Dict[str, Any]] = []

    def initialize(self):
        """初始化知识库"""
        self._ensure_index()

    def _ensure_index(self):
        """确保FAISS索引存在"""
        dim = 1024  # text-embedding-v3 输出维度

        # 尝试加载已有索引
        if os.path.exists(INDEX_FILE):
            try:
                self.index = faiss.read_index(INDEX_FILE)
                print(f"Loaded existing knowledge index with {self.index.ntotal} vectors")
            except Exception as e:
                print(f"Failed to load knowledge index: {e}")
                self.index = faiss.IndexFlatIP(dim)
        else:
            self.index = faiss.IndexFlatIP(dim)

        # 尝试加载 metadata
        if os.path.exists(METADATA_FILE):
            try:
                with open(METADATA_FILE, 'rb') as f:
                    self.metadata = pickle.load(f)
                print(f"Loaded {len(self.metadata)} metadata entries")
            except Exception as e:
                print(f"Failed to load metadata: {e}")
                self.metadata = []

    def add_texts(self, texts: List[str], metadata: Optional[List[Dict]] = None):
        """添加知识文本到向量库"""
        if not texts:
            return

        # 批量向量化
        vectors = self.embedding_service.encode(texts)
        if not vectors:
            raise Exception("Failed to embed texts")

        # 转为numpy数组并归一化
        vectors_array = np.array(vectors, dtype=np.float32)
        faiss.normalize_L2(vectors_array)

        # 添加到索引
        self.index.add(vectors_array)

        # 保存 metadata
        if metadata:
            self.metadata.extend(metadata)
        else:
            for text in texts:
                self.metadata.append({"text": text[:100]})

        # 持久化
        self._save()

        print(f"Added {len(texts)} texts to knowledge base, total: {self.index.ntotal}")

    def search(self, query: str, top_k: int = 3) -> List[Dict[str, Any]]:
        """检索与查询最相似的知识"""
        if not self.index or self.index.ntotal == 0:
            return []

        # 向量化查询
        query_vector = self.embedding_service.encode_one(query)
        if not query_vector:
            return []

        # 转为numpy数组并归一化
        query_array = np.array([query_vector], dtype=np.float32)
        faiss.normalize_L2(query_array)

        try:
            distances, indices = self.index.search(query_array, min(top_k, self.index.ntotal))
        except Exception as e:
            print(f"Knowledge search error: {e}")
            return []

        results = []
        for i, idx in enumerate(indices[0]):
            if idx < 0:
                continue

            result = {
                "text": self.metadata[idx]["text"] if idx < len(self.metadata) else "",
                "metadata": self.metadata[idx] if idx < len(self.metadata) else {},
                "distance": float(distances[0][i])
            }
            results.append(result)

        return results

    def delete_all(self):
        """删除所有知识"""
        if self.index:
            self.index.reset()
        self.metadata = []
        self._save()
        print("All knowledge deleted")

    def count(self) -> int:
        """返回知识库中的条目数"""
        return self.index.ntotal if self.index else 0

    def _save(self):
        """保存索引和metadata"""
        try:
            faiss.write_index(self.index, INDEX_FILE)
            with open(METADATA_FILE, 'wb') as f:
                pickle.dump(self.metadata, f)
        except Exception as e:
            print(f"Failed to save knowledge index: {e}")


# 全局实例
_knowledge_store: Optional[KnowledgeStore] = None


def get_knowledge_store() -> Optional[KnowledgeStore]:
    """获取知识库实例"""
    return _knowledge_store


def init_knowledge_store(embedding_service: Optional[EmbeddingService] = None):
    """初始化知识库服务"""
    global _knowledge_store

    if not embedding_service:
        from config import config
        if not config.embedding:
            print("Embedding未配置，跳过知识库初始化")
            return

        embedding_service = EmbeddingService(
            api_key=config.embedding.api_key,
            model=config.embedding.model
        )

    _knowledge_store = KnowledgeStore(embedding_service)
    _knowledge_store.initialize()
    print(f"Knowledge store initialized with {_knowledge_store.count()} entries")
