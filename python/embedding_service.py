"""
DashScope Embedding 服务
"""

from typing import List, Optional
import dashscope


class EmbeddingService:
    """DashScope Embedding 服务"""

    def __init__(self, api_key: str, model: str = "text-embedding-v3"):
        self.api_key = api_key
        self.model = model
        dashscope.api_key = api_key

    def encode(self, texts: List[str]) -> List[List[float]]:
        """批量向量化"""
        if not texts:
            return []

        response = dashscope.embeddings.TextEmbedding.call(
            model=self.model,
            input=texts
        )

        if response.status_code != 200:
            raise Exception(f"Embedding API error: {response.code} - {response.message}")

        embeddings = response.output['embeddings']
        return [item['embedding'] for item in embeddings]

    def encode_one(self, text: str) -> Optional[List[float]]:
        """单条文本向量化"""
        results = self.encode([text])
        return results[0] if results else None
