"""
业务逻辑服务
"""

import json
import requests
from typing import List, Optional
from models import Message, IntentRecognitionResponse, InterruptCheckRequest, InterruptCheckResponse
from config import config
from vector_store import get_embedding_service, get_milvus_store
from knowledge_store import get_knowledge_store, init_knowledge_store

try:
    from intent_vector_service import IntentVectorService
    from embedding_service import EmbeddingService
    INTENT_VECTOR_AVAILABLE = True
except ImportError:
    INTENT_VECTOR_AVAILABLE = False
    IntentVectorService = None
    EmbeddingService = None


# 全局意图向量服务
_intent_vector_service: Optional[object] = None


def get_intent_vector_service():
    """获取意图向量服务"""
    global _intent_vector_service
    return _intent_vector_service


def init_intent_vector_service():
    """初始化意图向量服务"""
    global _intent_vector_service
    if not INTENT_VECTOR_AVAILABLE:
        print("意图向量模块不可用，跳过初始化")
        return
    
    try:
        # Chroma需要embedding配置
        if config.embedding and EmbeddingService and IntentVectorService:
            embedding_service = EmbeddingService(
                api_key=config.embedding.api_key,
                model=config.embedding.model
            )
            _intent_vector_service = IntentVectorService(
                embedding_service=embedding_service,
            )
            _intent_vector_service.initialize()
            print("意图向量服务初始化成功")
        else:
            print("Embedding未配置，跳过意图向量服务")
    except Exception as e:
        print(f"初始化意图向量服务失败: {e}")


class ChatService:
    
    def __init__(self):
        """初始化聊天服务"""
        self.openai_api_key = config.llm.api_key
        self.openai_base_url = config.llm.base_url
        self.api_model = config.llm.model
    
    def recognize_intent_with_ai(self, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
        """使用OpenAI API进行意图识别"""
        return _recognize_intent_with_ai(self, message, history)
    
    def recognize_intent_fallback(self, message: str) -> IntentRecognitionResponse:
        """基于规则的备用意图识别（当AI不可用时）"""
        return _recognize_intent_fallback(message)
    
    def recognize_intent(self, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
        """意图识别主函数：向量匹配 -> LLM兜底 -> 规则兜底"""
        return _recognize_intent(self, message, history)
    
    def generate_reply(self, message: str, intent: str, flow_id: Optional[str] = None, history: Optional[List[Message]] = None) -> str:
        """根据意图和上下文生成回复"""
        return _generate_reply(self, message, intent, flow_id, history)
    
    def check_flow_interrupt(self, request: InterruptCheckRequest) -> InterruptCheckResponse:
        """检查是否应该打断当前Flow"""
        return _check_flow_interrupt(self, request)

    def retrieve_context(self, query: str, top_k: int = 3) -> str:
        """从向量库检索相关上下文"""
        return _retrieve_context(self, query, top_k)

    def add_knowledge(self, texts: List[str], metadata: Optional[List[dict]] = None):
        """添加知识到向量库"""
        return _add_knowledge(self, texts, metadata)




def _recognize_intent_with_ai(chat_service: ChatService, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """使用OpenAI API进行意图识别 - 具体实现"""
    
    # 构建系统提示
    system_prompt = """
你是一个专业的意图识别助手。请根据用户消息识别意图。

意图类型：
- faq
- flow
- unknown

如果 intent == flow，则 flow_id 只能从下面列表中选择：
- return_goods   （退货流程）
- order_query    （订单查询）
- customer_service （客服流程）

严禁输出其他 flow_id（如 return_process, refund_flow 等）。

请严格以 JSON 格式返回：
{
    "intent": "faq/flow/unknown",
    "confidence": 0-1,
    "reply": "简短说明",
    "flow_id": "return_goods / order_query / customer_service 或 null"
}

注意：
- 退货、退款、注册、登录等操作流程应识别为flow
- 如何、什么、为什么等一般性问题应识别为faq
- 如果无法确定，请返回unknown
"""
    
    # 构建消息列表
    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": f"用户消息: {message}"}
    ]

    # 如果有历史记录，添加到消息中
    if history:
        print(f"添加历史记录: {len(history)} 条消息")
        for msg in history:
            messages.insert(-1, {"role": msg.role, "content": msg.content})
    else:
        print("没有历史记录")

    try:
        # 调用OpenAI API
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {chat_service.openai_api_key}"
        }
        
        data = {
            "model": chat_service.api_model,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 500
        }
        
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
            headers=headers,
            json=data,
            timeout=config.llm.timeout
        )
        
        if response.status_code != 200:
            raise Exception(f"API请求失败: {response.status_code}, {response.text}")
        
        result = response.json()
        content = result["choices"][0]["message"]["content"]
        
        # 尝试解析JSON响应
        try:
            intent_data = json.loads(content)
            return IntentRecognitionResponse(**intent_data)
        except json.JSONDecodeError:
            # 如果JSON解析失败，返回默认响应
            return IntentRecognitionResponse(
                intent="unknown",
                confidence=0.5,
                reply="意图识别失败，请稍后再试。"
            )
            
    except Exception as e:
        print(f"意图识别错误: {str(e)}")
        # 返回默认响应
        return IntentRecognitionResponse(
            intent="unknown",
            confidence=0.5,
            reply="意图识别服务暂时不可用，请稍后再试。"
        )


def _recognize_intent_fallback(message: str) -> IntentRecognitionResponse:
    """基于规则的备用意图识别（当AI不可用时） - 具体实现"""
    message_lower = message.lower()
    
    # 退货/退款流程意图识别 - 优先级最高
    if any(word in message_lower for word in ["退货", "退款", "怎么退", "如何退"]):
        print(f"识别到退货/退款流程: {message}")
        return IntentRecognitionResponse(
            intent="flow",
            confidence=0.9,
            flow_id="return_goods",
            reply="退货流程已识别，后续将接入AI处理具体步骤"
        )
    
    # 注册流程意图识别
    elif any(word in message_lower for word in ["注册", "怎么注册", "如何注册"]):
        return IntentRecognitionResponse(
            intent="flow",
            confidence=0.9,
            flow_id="return_goods",
            reply="注册流程已识别，后续将接入AI处理具体步骤"
        )
    
    # FAQ意图识别 - 识别常见问题
    elif any(word in message_lower for word in ["如何", "怎么", "什么", "为什么"]):
        return IntentRecognitionResponse(
            intent="faq",
            confidence=0.85,
            reply="FAQ问题已识别，后续将接入AI处理具体回答"
        )
    
    # 未知意图
    else:
        return IntentRecognitionResponse(
            intent="unknown",
            confidence=0.5,
            reply="未知意图，后续将接入AI处理"
        )


def _recognize_intent(chat_service: ChatService, message: str, history: Optional[List[Message]] = None) -> IntentRecognitionResponse:
    """意图识别主函数：向量匹配 -> LLM兜底 -> 规则兜底"""
    print(f"开始意图识别: {message}")
    
    # 1. 尝试向量匹配
    vector_result = _recognize_intent_by_vector(message)
    if vector_result:
        print(f"向量匹配识别结果: {vector_result}")
        return vector_result
    
    # 2. 向量匹配失败，使用LLM识别
    try:
        result = chat_service.recognize_intent_with_ai(message, history)
        print(f"LLM意图识别结果: {result}")
        return result
    except Exception as e:
        print(f"LLM意图识别失败: {str(e)}")
    
    # 3. LLM失败，使用规则兜底
    result = chat_service.recognize_intent_fallback(message)
    print(f"规则兜底识别结果: {result}")
    return result


def _recognize_intent_by_vector(message: str) -> Optional[IntentRecognitionResponse]:
    """使用向量匹配进行意图识别"""
    try:
        intent_service = get_intent_vector_service()
        if not intent_service:
            return None
            
        result = intent_service.find_similar(message, top_k=1, threshold=0.6)
        if not result:
            return None

        return IntentRecognitionResponse(
            intent=result["intent_id"],
            confidence=result["confidence"],
            flow_id=result.get("next_flow"),
            reply=f"已识别为{result['intent_name']}"
        )
    except Exception as e:
        print(f"向量匹配失败: {e}")
        return None


def _generate_reply(chat_service: ChatService, message: str, intent: str, flow_id: Optional[str] = None, history: Optional[List[Message]] = None) -> str:
    """根据意图和上下文生成回复 - 具体实现"""
    
    # RAG: 如果是 faq 意图，先检索相关知识
    context = ""
    if intent == "faq" or intent == "unknown":
        context = _retrieve_context(chat_service, message, top_k=3)

    system_prompt = f""" 
    你是一个智能客服助手，不说废话，直接回答用户问题。 
    当前意图类型: {intent} 
    当前流程ID: {flow_id} 
    """

    # 添加知识库上下文
    if context:
        system_prompt += f"""
    下面是知识库中与用户问题相关的参考信息：
    {context}

    规则：
    - 请优先基于以上知识库信息回答用户问题
    - 如果知识库中没有相关信息，请如实说明
    - 如果是 flow，请引导用户继续完成该流程 
    - 如果是 unknown，请礼貌说明并建议转人工 
    """
    else:
        system_prompt += """
    规则： 
    - 如果是 flow，请引导用户继续完成该流程 
    - 如果是 faq，请直接回答用户问题 
    - 如果是 unknown，请礼貌说明并建议转人工 
    """

    messages = [
        {"role": "system", "content": system_prompt},
    ]

    # 加入历史对话
    if history:
        for msg in history:
            messages.append({"role": msg.role, "content": msg.content})

    # 加入当前用户消息
    messages.append({"role": "user", "content": message})

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {chat_service.openai_api_key}"
    }

    data = {
        "model": chat_service.api_model,
        "messages": messages,
        "temperature": 0.7,
        "max_tokens": 500
    }

    try:
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
            headers=headers,
            json=data,
            timeout=config.llm.timeout
        )

        if response.status_code != 200:
            raise Exception(f"API请求失败: {response.status_code}, {response.text}")

        result = response.json()
        return result["choices"][0]["message"]["content"]

    except Exception as e:
        print(f"生成回复失败: {e}")
        return "抱歉，当前系统繁忙，请稍后再试。"


def _check_flow_interrupt(chat_service: ChatService, request: InterruptCheckRequest) -> InterruptCheckResponse:
    """检查是否应该打断当前Flow - 具体实现"""
    print(f"Flow中断检查: session={request.session_id}, flow={request.flow_id}, step={request.current_step}")
    
    # 构建系统提示
    system_prompt = f"""
    你是一个Flow中断判断助手。请根据用户在当前流程中的输入，判断是否应该打断当前流程。

    当前流程信息：
    - 流程ID: {request.flow_id}
    - 当前步骤: {request.current_step}
    - 流程状态: {request.flow_state}

    用户输入: {request.user_message}

    判断规则：
    1. 如果用户明确表示要退出、取消、停止当前流程，应该打断
    2. 如果用户询问与当前流程无关的问题，应该打断
    3. 如果用户输入包含明显的错误或误解，应该打断
    4. 如果用户只是继续当前流程的正常操作，不应该打断

    请以JSON格式返回判断结果，格式如下：
    {{
        "should_interrupt": true/false,
        "confidence": 0-1之间的置信度,
        "new_intent": "如果打断，建议的新意图类型(可选)",
        "reason": "判断理由(可选)"
    }}
    """

    messages = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": f"用户输入: {request.user_message}"}
    ]

    try:
        # 调用OpenAI API进行中断判断
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {chat_service.openai_api_key}"
        }
        
        data = {
            "model": chat_service.api_model,
            "messages": messages,
            "temperature": 0.1,
            "max_tokens": 300
        }
        
        response = requests.post(
            f"{chat_service.openai_base_url}/chat/completions",
            headers=headers,
            json=data,
            timeout=config.llm.timeout
        )
        
        if response.status_code != 200:
            raise Exception(f"API请求失败: {response.status_code}, {response.text}")
        
        result = response.json()
        content = result["choices"][0]["message"]["content"]
        
        # 尝试解析JSON响应
        try:
            interrupt_data = json.loads(content)
            return InterruptCheckResponse(**interrupt_data)
        except json.JSONDecodeError:
            # 如果JSON解析失败，返回默认响应（不打断）
            print("Flow中断检查JSON解析失败，默认不打断")
            return InterruptCheckResponse(
                should_interrupt=False,
                confidence=0.5,
                reason="解析失败，默认继续流程"
            )
            
    except Exception as e:
        print(f"Flow中断检查错误: {str(e)}")
        # 出错时默认不打断
        return InterruptCheckResponse(
            should_interrupt=False,
            confidence=0.3,
            reason=f"检查失败: {str(e)}"
        )


def _retrieve_context(chat_service: ChatService, query: str, top_k: int = 3) -> str:
    """从向量库检索相关上下文"""
    try:
        # 使用 FAISS 知识库检索
        knowledge_store = get_knowledge_store()
        if not knowledge_store:
            print("知识库未初始化，跳过 RAG 检索")
            return ""

        # 检索相似内容
        print(f"RAG检索相似内容 top_k={top_k}")
        results = knowledge_store.search(query, top_k=top_k)

        if not results:
            print("未找到相关内容")
            return ""

        # 构建上下文
        context_parts = []
        for i, result in enumerate(results, 1):
            text = result["text"]
            distance = result.get("distance", 0)
            context_parts.append(f"[{i}] {text} (相似度: {distance:.3f})")

        context = "\n\n".join(context_parts)
        print(f"检索到 {len(results)} 条相关知识")
        return context

    except Exception as e:
        print(f"RAG 检索失败: {str(e)}")
        return ""


def _add_knowledge(chat_service: ChatService, texts: List[str], metadata: Optional[List[dict]] = None):
    """添加知识到向量库"""
    try:
        knowledge_store = get_knowledge_store()
        if not knowledge_store:
            raise ValueError("知识库未初始化")

        knowledge_store.add_texts(texts, metadata)
        print(f"成功添加 {len(texts)} 条知识")

    except Exception as e:
        print(f"添加知识失败: {str(e)}")
        raise