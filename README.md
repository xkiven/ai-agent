# AI 智能客服系统

基于 LLM 的智能客服对话机器人，支持多轮对话、意图识别、RAG 知识库检索、Function Calling 工具调用，以及 SSE 流式输出。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | React 18 + TypeScript + Vite + TailwindCSS |
| 后端 | Go 1.24 + Gin |
| AI 服务 | Python + FastAPI |
| 存储 | Redis (会话) + FAISS (向量检索) |
| LLM | 通义千问 / OpenAI 兼容 API |

## 系统架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   React     │────▶│   Go API    │────▶│  Python AI  │
│   Frontend  │◀────│   (Gin)     │◀────│  (FastAPI)  │
└─────────────┘     └──────┬──────┘     └──────┬──────┘
                           │                    │
                    ┌──────▼──────┐      ┌──────▼──────┐
                    │    Redis    │      │  LLM API    │
                    │  (Session)  │      │ (通义千问)   │
                    └─────────────┘      └─────────────┘
```

## 功能特性

- **SSE 流式输出** - AI 回复实时推送，打字机效果提升用户体验
- **Function Calling** - 自动调用业务系统（订单查询、物流查询、退货处理）
- **RAG 知识库** - 基于 FAISS 向量检索，增强回答准确性
- **意图智能分类** - Decision Layer 决策层，自动识别用户意图
- **多轮对话状态机** - Flow 流程管理，支持复杂业务场景
- **Markdown 渲染** - 支持代码块、链接等富文本显示

## 快速开始

### 前置要求

- Go 1.24+
- Python 3.10+
- Redis
- LLM API (通义千问或其他 OpenAI 兼容接口)

### 1. 启动 AI 服务

```bash
cd python
pip install -r requirements.txt
python main.py
```

AI 服务默认监听 `http://127.0.0.1:8000`

### 2. 启动 Go 后端

```bash
go run main.go
```

服务默认监听 `:8080`

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认访问 `http://localhost:5173`

## 项目结构

```
ai-agent/
├── api/                    # HTTP 接口处理
│   ├── chat.go            # 聊天相关接口
│   └── knowledge.go       # 知识库接口
├── dao/                   # 数据访问层
│   └── redis.go          # Redis 会话存储
├── model/                 # 数据模型
│   └── model.go          # 核心数据结构
├── route/                 # 路由注册
│   └── router.go         # Gin 路由配置
├── service/               # 业务逻辑
│   ├── chat.go           # 聊天服务
│   ├── decision_layer.go # 意图决策层
│   ├── flow_registry.go  # Flow 注册
│   ├── flows/            # 业务流程实现
│   │   ├── order_query.go
│   │   ├── logistics.go
│   │   ├── return_goods.go
│   │   └── customer_service.go
│   └── type_classify.go  # 类型分类
├── internal/aiclient/    # AI 客户端
│   └── client.go         # HTTP 调用封装
├── config/               # 配置文件
│   └── intents.yaml     # 意图配置
├── python/               # Python AI 服务
│   ├── main.py          # FastAPI 入口
│   ├── services.py      # 核心服务
│   ├── knowledge_store.py # 知识库存储
│   └── vector_store.py  # 向量存储
└── frontend/             # React 前端
    ├── src/
    │   ├── App.tsx      # 主应用
    │   ├── api/         # API 调用
    │   └── components/  # UI 组件
    └── package.json
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/chat` | 普通聊天（非流式） |
| POST | `/chat/stream` | SSE 流式聊天 |
| DELETE | `/session/:session_id` | 清除会话 |
| POST | `/knowledge/add` | 添加知识条目 |
| POST | `/knowledge/query` | 知识库检索 |

## 支持的意图

### 流程类

- **退货申请** - 处理退货请求
- **订单查询** - 查询订单状态
- **物流查询** - 查询快递信息
- **联系客服** - 转人工服务

### FAQ 类

- 发货问题、支付问题、退款时效
- 账户问题、活动优惠、产品咨询
- 退货政策、联系客服

## 技术亮点

本项目展示了以下技术能力：

1. **SSE 流式输出** - 使用 Server-Sent Events 实现实时响应
2. **Function Calling** - LLM 工具调用模式，连接业务系统
3. **RAG 知识库** - 向量检索增强回答质量
4. **多轮对话状态机** - Flow 流程管理复杂交互
5. **意图分类** - Decision Layer 智能路由

## 配置说明

### Go 后端

默认配置在 `main.go` 中：

- AI 客户端地址: `http://127.0.0.1:8000`
- Redis 地址: `localhost:6379`
- 服务端口: `:8080`

### Python AI 服务

在 `python/config.py` 中配置：

- LLM API 地址和密钥
- 向量模型配置
- FAISS 索引路径

### 意图配置

编辑 `config/intents.yaml` 自定义意图、关键词和流程。

## 注意事项

- 确保 Redis 服务正常运行
- 确保 LLM API 可访问（通义千问或其他）
- 前端需与后端保持同源或配置 CORS
