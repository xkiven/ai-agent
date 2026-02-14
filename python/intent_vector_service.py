"""
意图向量服务 - 基于Milvus的向量化意图识别
"""

import yaml
from typing import List, Dict, Optional
from pathlib import Path

from pymilvus import Collection, CollectionSchema, FieldSchema, DataType, connections, utility
from pymilvus.exceptions import MilvusException

from embedding_service import EmbeddingService


COLLECTION_NAME = "intent_vectors"


class IntentVectorService:
    """意图向量服务"""

    def __init__(
        self,
        embedding_service: EmbeddingService,
        milvus_host: str = "localhost",
        milvus_port: int = 19530,
        config_path: str = "../config/intents.yaml"
    ):
        self.embedding_service = embedding_service
        self.milvus_host = milvus_host
        self.milvus_port = milvus_port
        self.config_path = config_path
        self.collection = None
        self.intents = []

    def initialize(self):
        """初始化服务"""
        self._connect_milvus()
        self.intents = self._load_intents()
        self._ensure_collection()

    def _connect_milvus(self):
        """连接Milvus"""
        try:
            connections.connect(
                alias="default",
                host=self.milvus_host,
                port=self.milvus_port
            )
            print(f"Connected to Milvus at {self.milvus_host}:{self.milvus_port}")
        except MilvusException as e:
            print(f"Failed to connect to Milvus: {e}")
            raise

    def _load_intents(self) -> List[Dict]:
        """从YAML加载意图定义"""
        config_file = Path(self.config_path)
        if not config_file.exists():
            config_file = Path(__file__).parent / self.config_path

        with open(config_file, 'r', encoding='utf-8') as f:
            config = yaml.safe_load(f)

        intents = config.get('intents', [])
        enabled_intents = [i for i in intents if i.get('enabled', True)]
        print(f"Loaded {len(enabled_intents)} enabled intents from {config_file}")
        return enabled_intents

    def _ensure_collection(self):
        """确保Milvus集合存在"""
        if utility.has_collection(COLLECTION_NAME):
            self.collection = Collection(COLLECTION_NAME)
            self.collection.load()
            print(f"Loaded existing collection: {COLLECTION_NAME}")
            
            if self.collection.num_entities >= len(self.intents):
                print("Intent vectors already indexed")
                return

        self._create_and_index()

    def _create_and_index(self):
        """创建集合并索引意图向量"""
        if utility.has_collection(COLLECTION_NAME):
            utility.drop_collection(COLLECTION_NAME)
            print(f"Dropped existing collection: {COLLECTION_NAME}")

        fields = [
            FieldSchema(name="intent_id", dtype=DataType.VARCHAR, max_length=64),
            FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=1024)
        ]
        schema = CollectionSchema(fields=fields, description="Intent vectors")
        self.collection = Collection(name=COLLECTION_NAME, schema=schema)

        intent_ids = []
        vectors = []

        for intent in self.intents:
            keywords = intent.get('keywords', [])
            examples = intent.get('examples', [])
            text = ' '.join(keywords + examples)
            
            if not text.strip():
                continue

            vector = self.embedding_service.encode_one(text)
            if vector:
                intent_ids.append(intent['id'])
                vectors.append(vector)

        if intent_ids:
            self.collection.insert([intent_ids, vectors])
            self.collection.flush()
            print(f"Indexed {len(intent_ids)} intent vectors")

        index_params = {
            "metric_type": "IP",
            "index_type": "IVF_FLAT",
            "params": {"nlist": 128}
        }
        self.collection.create_index(field_name="vector", index_params=index_params)
        self.collection.load()
        print(f"Created index for collection: {COLLECTION_NAME}")

    def find_similar(self, query: str, top_k: int = 3, threshold: float = 0.5) -> Optional[Dict]:
        """查找最相似的意图"""
        if not self.collection:
            return None

        query_vector = self.embedding_service.encode_one(query)
        if not query_vector:
            return None

        search_params = {"metric_type": "IP", "params": {"nprobe": 10}}
        
        try:
            results = self.collection.search(
                data=[query_vector],
                anns_field="vector",
                param=search_params,
                limit=top_k,
                output_fields=["intent_id"]
            )
        except MilvusException as e:
            print(f"Search error: {e}")
            return None

        if not results or not results[0]:
            return None

        best_hit = results[0][0]
        intent_id = best_hit.entity.get("intent_id")
        score = best_hit.distance

        if score < threshold:
            return None

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
