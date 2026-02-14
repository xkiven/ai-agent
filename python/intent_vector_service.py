"""
意图向量服务 - 基于FAISS的向量化意图识别
"""

import yaml
import numpy as np
from typing import List, Dict, Optional
from pathlib import Path
import pickle
import os

import faiss

from embedding_service import EmbeddingService


INDEX_FILE = "./faiss_index.bin"


class IntentVectorService:
    """意图向量服务"""

    def __init__(
        self,
        embedding_service: EmbeddingService,
        config_path: str = "../config/intents.yaml"
    ):
        self.embedding_service = embedding_service
        self.config_path = config_path
        self.index = None
        self.intents = []
        self.intent_ids = []

    def initialize(self):
        """初始化服务"""
        self.intents = self._load_intents()
        self._ensure_index()

    def _load_intents(self) -> List[Dict]:
        """从YAML加载意图定义"""
        config_file = Path(self.config_path)
        if not config_file.exists():
            config_file = Path(__file__).parent.parent / "config" / "intents.yaml"

        with open(config_file, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)

        intents = config.get('intents', [])
        enabled_intents = [i for i in intents if i.get('enabled', True)]
        print(f"Loaded {len(enabled_intents)} enabled intents from {config_file}")
        return enabled_intents

    def _ensure_index(self):
        """确保FAISS索引存在"""
        # 尝试加载已有索引
        if os.path.exists(INDEX_FILE):
            try:
                self.index = faiss.read_index(INDEX_FILE)
                print(f"Loaded existing FAISS index with {self.index.ntotal} vectors")
                return
            except Exception as e:
                print(f"Failed to load index: {e}")

        # 创建新索引
        dim = 1024  # text-embedding-v3 输出维度
        self.index = faiss.IndexFlatIP(dim)  # 内积相似度
        self._build_index()

    def _build_index(self):
        """构建意图向量索引"""
        if not self.intents:
            return

        vectors = []
        self.intent_ids = []

        for intent in self.intents:
            keywords = intent.get('keywords', [])
            examples = intent.get('examples', [])
            text = ' '.join(keywords + examples)

            if not text.strip():
                continue

            vector = self.embedding_service.encode_one(text)
            if vector:
                vectors.append(vector)
                self.intent_ids.append(intent['id'])

        if vectors:
            # FAISS需要float32
            vectors_array = np.array(vectors, dtype=np.float32)
            # 归一化（用于余弦相似度）
            faiss.normalize_L2(vectors_array)
            self.index.add(vectors_array)
            print(f"Indexed {len(vectors)} intent vectors")

            # 保存索引
            try:
                faiss.write_index(self.index, INDEX_FILE)
                print(f"Saved index to {INDEX_FILE}")
            except Exception as e:
                print(f"Failed to save index: {e}")

    def find_similar(self, query: str, top_k: int = 3, threshold: float = 0.5) -> Optional[Dict]:
        """查找最相似的意图"""
        if not self.index or self.index.ntotal == 0:
            return None

        query_vector = self.embedding_service.encode_one(query)
        if not query_vector:
            return None

        # 转为numpy数组并归一化
        query_array = np.array([query_vector], dtype=np.float32)
        faiss.normalize_L2(query_array)

        try:
            distances, indices = self.index.search(query_array, min(top_k, self.index.ntotal))
        except Exception as e:
            print(f"Search error: {e}")
            return None

        if len(indices) == 0 or len(indices[0]) == 0:
            return None

        best_idx = indices[0][0]
        best_distance = distances[0][0]

        # FAISS内积就是余弦相似度
        score = float(best_distance)

        if score < threshold:
            return None

        intent_id = self.intent_ids[best_idx]
        intent = next((i for i in self.intents if i['id'] == intent_id), None)

        if not intent:
            return None

        return {
            "intent_id": intent_id,
            "intent_name": intent.get('name', intent_id),
            "type": intent.get('type', 'unknown'),
            "confidence": min(score, 1.0),
            "next_flow": intent.get('next_flow'),
            "score": score
        }

    def get_intent_by_id(self, intent_id: str) -> Optional[Dict]:
        """根据ID获取意图定义"""
        return next((i for i in self.intents if i['id'] == intent_id), None)
