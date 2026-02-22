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


# ==================== Function Calling Mock 数据 ====================
MOCK_ORDERS = {
    "123456": {
        "order_id": "123456",
        "status": "已发货",
        "product": "iPhone 15 Pro 256GB",
        "amount": "7999元",
        "create_time": "2026-02-10 10:30:00",
        "logistics_company": "顺丰速运",
        "logistics_no": "SF1234567890"
    },
    "789012": {
        "order_id": "789012",
        "status": "配送中",
        "product": "MacBook Air M3",
        "amount": "9499元",
        "create_time": "2026-02-12 15:20:00",
        "logistics_company": "圆通速递",
        "logistics_no": "YT7890123456"
    },
    "345678": {
        "order_id": "345678",
        "status": "已签收",
        "product": "AirPods Pro 2",
        "amount": "1899元",
        "create_time": "2026-02-08 09:00:00",
        "logistics_company": "京东物流",
        "logistics_no": "JD3456789012"
    }
}

MOCK_LOGISTICS = {
    "SF1234567890": {
        "logistics_no": "SF1234567890",
        "company": "顺丰速运",
        "status": "派送中",
        "current": "北京市朝阳区XX街道营业部",
        "track": [
            {"time": "2026-02-15 08:00", "status": "正在派送中"},
            {"time": "2026-02-14 20:00", "status": "到达北京分拨中心"},
            {"time": "2026-02-13 15:00", "status": "已发货"}
        ]
    },
    "YT7890123456": {
        "logistics_no": "YT7890123456",
        "company": "圆通速递",
        "status": "运输中",
        "current": "上海市青浦区XX中转站",
        "track": [
            {"time": "2026-02-15 10:00", "status": "运输中"},
            {"time": "2026-02-14 18:00", "status": "已到达上海分拨中心"},
            {"time": "2026-02-13 12:00", "status": "已发货"}
        ]
    },
    "JD3456789012": {
        "logistics_no": "JD3456789012",
        "company": "京东物流",
        "status": "已签收",
        "current": "已签收",
        "track": [
            {"time": "2026-02-14 14:00", "status": "已签收"},
            {"time": "2026-02-14 08:00", "status": "配送员正在为您派送"},
            {"time": "2026-02-13 20:00", "status": "已到达配送站"}
        ]
    }
}


# ==================== Function Calling 工具定义 ====================
TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "query_order",
            "description": "查询订单状态和详细信息。当用户问订单相关问题（如订单在哪、订单状态、订单详情）时使用。",
            "parameters": {
                "type": "object",
                "properties": {
                    "order_id": {
                        "type": "string",
                        "description": "订单号"
                    }
                },
                "required": ["order_id"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "query_logistics",
            "description": "查询快递物流信息。当用户问快递、物流、运单相关问题时使用。",
            "parameters": {
                "type": "object",
                "properties": {
                    "logistics_no": {
                        "type": "string",
                        "description": "快递单号"
                    },
                    "order_id": {
                        "type": "string",
                        "description": "订单号（如果有的话）"
                    }
                },
                "required": []
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "create_ticket",
            "description": "创建客服工单。当用户需要投诉、反馈问题、无法解决的问题需要转人工时使用。",
            "parameters": {
                "type": "object",
                "properties": {
                    "title": {
                        "type": "string",
                        "description": "工单标题"
                    },
                    "content": {
                        "type": "string",
                        "description": "工单内容"
                    },
                    "ticket_type": {
                        "type": "string",
                        "description": "工单类型：complaint(投诉)、feedback(反馈)、consult(咨询)",
                        "enum": ["complaint", "feedback", "consult"]
                    }
                },
                "required": ["title", "content"]
            }
        }
    }
]


def execute_tool(tool_name: str, arguments: dict) -> str:
    """执行工具函数并返回结果"""
    try:
        if tool_name == "query_order":
            order_id = arguments.get("order_id", "")
            order = MOCK_ORDERS.get(order_id)
            if order:
                return json.dumps(order, ensure_ascii=False)
            else:
                return json.dumps({"error": f"未找到订单 {order_id}"}, ensure_ascii=False)
        
        elif tool_name == "query_logistics":
            logistics_no = arguments.get("logistics_no", "")
            order_id = arguments.get("order_id", "")
            
            # 优先用快递单号查，其次用订单号查
            logistics = None
            if logistics_no:
                logistics = MOCK_LOGISTICS.get(logistics_no)
            elif order_id:
                # 从订单中获取快递信息
                order = MOCK_ORDERS.get(order_id)
                if order and order.get("logistics_no"):
                    logistics = MOCK_LOGISTICS.get(order["logistics_no"])
            
            if logistics:
                return json.dumps(logistics, ensure_ascii=False)
            else:
                return json.dumps({"error": "未找到物流信息"}, ensure_ascii=False)
        
        elif tool_name == "create_ticket":
            import uuid
            ticket_id = f"TKT-{uuid.uuid4().hex[:8].upper()}"
            ticket = {
                "ticket_id": ticket_id,
                "title": arguments.get("title", ""),
                "content": arguments.get("content", ""),
                "ticket_type": arguments.get("ticket_type", "consult"),
                "status": "created",
                "message": "工单创建成功，客服将尽快处理"
            }
            return json.dumps(ticket, ensure_ascii=False)
        
        else:
            return json.dumps({"error": f"未知工具: {tool_name}"}, ensure_ascii=False)
    
    except Exception as e:
        return json.dumps({"error": f"执行工具失败: {str(e)}"}, ensure_ascii=False)


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
你是一个专业的意图识别助手。请根据用户消息识别具体的意图ID。

具体意图ID列表：
- return_goods   （退货、退款、怎么退、不想要了）
- order_query    （订单查询、查订单、看看订单）
- logistics      （物流查询、快递到哪里了）
- exchange       （换货、换一个、换颜色）
- faq_shipping   （发货问题、什么时候发）
- faq_payment    （支付问题、怎么付款）
- faq_refund_time （退款时效、什么时候退）
- faq_account    （账户问题、登录注册）
- faq_promotion  （活动优惠、折扣）
- faq_product    （产品咨询）
- faq_return_policy （退货政策）
- faq_contact    （联系客服）

重要规则：
- intent字段必须返回上面列出的具体intent_id，不能是"flow"或"faq"！
- flow_id字段：如果是flow类型，填写对应的flow_id（如return_goods、order_query、logistics、exchange），不要加任何后缀！
- 如果是退货、退款相关 → intent=return_goods, flow_id=return_goods
- 如果是查询订单相关 → intent=order_query, flow_id=order_query
- 如果是物流快递相关 → intent=logistics, flow_id=logistics
- 如果是一般问题 → intent=faq开头的ID，不需要flow_id
- 如果无法确定 → intent=unknown

请严格以 JSON 格式返回：
{
    "intent": "具体的intent_id",
    "confidence": 0-1,
    "flow_id": "对应的flow_id（如flow类型则填写，无则不填）"
}
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
            # 不需要 reply 字段
            intent_data.pop("reply", None)
            return IntentRecognitionResponse(**intent_data)
        except json.JSONDecodeError:
            # 如果JSON解析失败，返回默认响应
            return IntentRecognitionResponse(
                intent="unknown",
                confidence=0.5
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
    - 如果用户询问订单、物流相关信息，可以使用工具查询
    - 如果用户需要投诉或转人工，可以使用工具创建工单
    """
    else:
        system_prompt += """
    规则： 
    - 如果是 flow，请引导用户继续完成该流程 
    - 如果是 faq，请直接回答用户问题 
    - 如果是 unknown，请礼貌说明并建议转人工 
    - 如果用户询问订单、物流相关信息，可以使用工具查询
    - 如果用户需要投诉或转人工，可以使用工具创建工单
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
        "tools": TOOLS,
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
        assistant_message = result["choices"][0]["message"]

        # 检查是否需要调用工具
        if "tool_calls" in assistant_message:
            tool_calls = assistant_message["tool_calls"]
            print(f"Function Calling: 检测到 {len(tool_calls)} 个工具调用")

            # 执行工具调用
            for tool_call in tool_calls:
                func_name = tool_call["function"]["name"]
                func_args = json.loads(tool_call["function"]["arguments"])
                print(f"执行工具: {func_name}, 参数: {func_args}")

                # 执行工具并获取结果
                tool_result = execute_tool(func_name, func_args)
                print(f"工具返回: {tool_result}")

                # 将工具结果添加到消息中
                messages.append({
                    "role": "assistant",
                    "tool_calls": tool_calls
                })
                messages.append({
                    "role": "tool",
                    "tool_call_id": tool_call["id"],
                    "content": tool_result
                })

            # 再次调用 LLM 生成最终回复
            data2 = {
                "model": chat_service.api_model,
                "messages": messages,
                "temperature": 0.7,
                "max_tokens": 500
            }

            response2 = requests.post(
                f"{chat_service.openai_base_url}/chat/completions",
                headers=headers,
                json=data2,
                timeout=config.llm.timeout
            )

            if response2.status_code != 200:
                raise Exception(f"二次调用失败: {response2.status_code}")

            result2 = response2.json()
            return result2["choices"][0]["message"]["content"]

        return assistant_message["content"]

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