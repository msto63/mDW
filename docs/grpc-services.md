# meinDENKWERK gRPC Services Documentation

This document describes the gRPC services available in meinDENKWERK.

## Service Overview

| Service | Package | Port | Description |
|---------|---------|------|-------------|
| **Turing** | `mdw.turing` | 9200 | LLM Management (Chat, Embeddings, Models) |
| **Hypatia** | `mdw.hypatia` | 9220 | RAG (Search, Documents, Collections) |
| **Leibniz** | `mdw.leibniz` | 9140 | Agentic AI (Agents, Execution, Tools) |
| **Babbage** | `mdw.babbage` | 9150 | NLP Processing (Analyze, Sentiment, Entities) |
| **Russell** | `mdw.russell` | 9100 | Service Discovery (Register, Discover, Health) |
| **Bayes** | `mdw.bayes` | 9120 | Logging & Metrics |

---

## Turing Service (LLM Management)

**Package:** `mdw.turing`
**Port:** 9200

### Methods

#### Chat
```protobuf
rpc Chat(ChatRequest) returns (ChatResponse)
```
Synchronous chat completion with an LLM.

#### StreamChat
```protobuf
rpc StreamChat(ChatRequest) returns (stream ChatChunk)
```
Streaming chat completion - tokens are streamed as they are generated.

#### Embed
```protobuf
rpc Embed(EmbedRequest) returns (EmbedResponse)
```
Generate embedding vector for a single text input.

#### BatchEmbed
```protobuf
rpc BatchEmbed(BatchEmbedRequest) returns (BatchEmbedResponse)
```
Generate embeddings for multiple texts efficiently.

#### ListModels
```protobuf
rpc ListModels(Empty) returns (ModelListResponse)
```
List all available LLM models.

#### GetModel
```protobuf
rpc GetModel(GetModelRequest) returns (ModelInfo)
```
Get detailed information about a specific model.

#### PullModel
```protobuf
rpc PullModel(PullModelRequest) returns (stream PullProgress)
```
Download a model from the registry with progress streaming.

### Messages

#### ChatRequest
| Field | Type | Description |
|-------|------|-------------|
| model | string | Model name (optional, uses default) |
| messages | Message[] | Chat history |
| temperature | float | Sampling temperature (0.0-2.0, default: 0.7) |
| max_tokens | int32 | Maximum tokens in response (default: 2048) |
| system_prompt | string | System prompt override |
| conversation_id | string | For conversation tracking |
| stop | string[] | Stop sequences |
| top_p | float | Nucleus sampling (default: 1.0) |

#### Message
| Field | Type | Description |
|-------|------|-------------|
| role | string | `system`, `user`, or `assistant` |
| content | string | Message content |

#### ChatResponse
| Field | Type | Description |
|-------|------|-------------|
| content | string | Assistant's response |
| model | string | Model used |
| prompt_tokens | int32 | Tokens in prompt |
| completion_tokens | int32 | Tokens in response |
| total_tokens | int32 | Total tokens |
| finish_reason | string | `stop`, `length`, or `error` |

---

## Hypatia Service (RAG)

**Package:** `mdw.hypatia`
**Port:** 9220

### Methods

#### Search
```protobuf
rpc Search(SearchRequest) returns (SearchResponse)
```
Semantic vector search across indexed documents.

#### HybridSearch
```protobuf
rpc HybridSearch(HybridSearchRequest) returns (SearchResponse)
```
Combined vector + keyword search with configurable weights.

#### IngestDocument
```protobuf
rpc IngestDocument(IngestDocumentRequest) returns (IngestResponse)
```
Index a document for semantic search.

#### IngestFile
```protobuf
rpc IngestFile(stream FileChunk) returns (IngestResponse)
```
Stream-upload and index a file.

#### DeleteDocument / GetDocument / ListDocuments
Document CRUD operations.

#### CreateCollection / DeleteCollection / ListCollections
Collection management.

#### GetCollectionStats
```protobuf
rpc GetCollectionStats(GetCollectionStatsRequest) returns (CollectionStats)
```
Get statistics about a collection (document count, storage, etc.).

#### AugmentPrompt
```protobuf
rpc AugmentPrompt(AugmentPromptRequest) returns (AugmentPromptResponse)
```
Augment a prompt with relevant context from the knowledge base (RAG).

### Messages

#### SearchRequest
| Field | Type | Description |
|-------|------|-------------|
| query | string | Search query |
| collection | string | Collection name (optional) |
| top_k | int32 | Number of results (default: 5) |
| min_score | float | Minimum similarity (default: 0.7) |
| filters | map<string,string> | Metadata filters |

#### SearchResult
| Field | Type | Description |
|-------|------|-------------|
| chunk_id | string | Chunk identifier |
| document_id | string | Parent document ID |
| content | string | Chunk content |
| score | float | Similarity score (0-1) |
| metadata | DocumentMetadata | Document metadata |

#### IngestDocumentRequest
| Field | Type | Description |
|-------|------|-------------|
| title | string | Document title |
| content | string | Document content |
| collection | string | Target collection |
| source | string | Source URL/path |
| options | IngestOptions | Chunking options |
| metadata | map<string,string> | Custom metadata |

#### ChunkingStrategy (Enum)
- `CHUNKING_STRATEGY_FIXED` - Fixed character count
- `CHUNKING_STRATEGY_SENTENCE` - Sentence boundaries
- `CHUNKING_STRATEGY_PARAGRAPH` - Paragraph boundaries
- `CHUNKING_STRATEGY_SEMANTIC` - Semantic similarity

---

## Leibniz Service (Agentic AI)

**Package:** `mdw.leibniz`
**Port:** 9140

### Methods

#### CreateAgent / UpdateAgent / DeleteAgent / GetAgent / ListAgents
Agent lifecycle management.

#### Execute
```protobuf
rpc Execute(ExecuteRequest) returns (ExecuteResponse)
```
Execute a task with the agent (synchronous).

#### StreamExecute
```protobuf
rpc StreamExecute(ExecuteRequest) returns (stream AgentChunk)
```
Execute with streaming output showing thinking, tool calls, and results.

#### ContinueExecution
```protobuf
rpc ContinueExecution(ContinueRequest) returns (ExecuteResponse)
```
Continue an execution that's awaiting confirmation.

#### CancelExecution
```protobuf
rpc CancelExecution(CancelRequest) returns (Empty)
```
Cancel a running execution.

#### ListTools / RegisterTool / UnregisterTool
Tool management.

### Messages

#### ExecuteRequest
| Field | Type | Description |
|-------|------|-------------|
| agent_id | string | Agent to use (optional) |
| message | string | Task description |
| conversation_id | string | Conversation context |
| variables | map<string,string> | Variables for the agent |
| auto_approve_tools | bool | Skip tool confirmations |

#### AgentChunk (Stream)
| Field | Type | Description |
|-------|------|-------------|
| type | ChunkType | Type of chunk |
| content | string | Content/output |
| action | AgentAction | Tool action details |
| iteration | int32 | Current iteration |

#### ChunkType (Enum)
- `CHUNK_TYPE_THINKING` - Agent reasoning
- `CHUNK_TYPE_TOOL_CALL` - Tool invocation
- `CHUNK_TYPE_TOOL_RESULT` - Tool result
- `CHUNK_TYPE_RESPONSE` - Intermediate response
- `CHUNK_TYPE_FINAL` - Final response

#### ExecutionStatus (Enum)
- `EXECUTION_STATUS_RUNNING`
- `EXECUTION_STATUS_COMPLETED`
- `EXECUTION_STATUS_AWAITING_CONFIRMATION`
- `EXECUTION_STATUS_ERROR`
- `EXECUTION_STATUS_CANCELLED`

#### ToolSource (Enum)
- `TOOL_SOURCE_BUILTIN` - Built-in tools
- `TOOL_SOURCE_MCP` - MCP (Model Context Protocol) tools
- `TOOL_SOURCE_CUSTOM` - Custom registered tools

---

## Babbage Service (NLP)

**Package:** `mdw.babbage`
**Port:** 9150

### Methods

#### Analyze
```protobuf
rpc Analyze(AnalyzeRequest) returns (AnalyzeResponse)
```
Full NLP analysis (sentiment, entities, keywords, language).

#### AnalyzeSentiment
```protobuf
rpc AnalyzeSentiment(SentimentRequest) returns (SentimentResponse)
```

#### ExtractEntities
```protobuf
rpc ExtractEntities(EntityRequest) returns (EntityResponse)
```

#### ExtractKeywords
```protobuf
rpc ExtractKeywords(KeywordRequest) returns (KeywordResponse)
```

#### DetectLanguage
```protobuf
rpc DetectLanguage(LanguageRequest) returns (LanguageResponse)
```

#### Summarize
```protobuf
rpc Summarize(SummarizeRequest) returns (SummarizeResponse)
```

#### Translate
```protobuf
rpc Translate(TranslateRequest) returns (TranslateResponse)
```

#### Classify
```protobuf
rpc Classify(ClassifyRequest) returns (ClassifyResponse)
```

### Entity Types
- `ENTITY_TYPE_PERSON`
- `ENTITY_TYPE_ORGANIZATION`
- `ENTITY_TYPE_LOCATION`
- `ENTITY_TYPE_DATE`
- `ENTITY_TYPE_NUMBER`
- `ENTITY_TYPE_EMAIL`
- `ENTITY_TYPE_URL`
- `ENTITY_TYPE_PHONE`

### Sentiment
- `SENTIMENT_POSITIVE`
- `SENTIMENT_NEGATIVE`
- `SENTIMENT_NEUTRAL`

### Summarization Styles
- `SUMMARIZATION_STYLE_BRIEF`
- `SUMMARIZATION_STYLE_DETAILED`
- `SUMMARIZATION_STYLE_BULLET_POINTS`
- `SUMMARIZATION_STYLE_HEADLINE`

---

## Russell Service (Service Discovery)

**Package:** `mdw.russell`
**Port:** 9100

### Methods

#### RegisterService
```protobuf
rpc RegisterService(RegisterRequest) returns (RegisterResponse)
```
Register a service with the discovery system.

#### DeregisterService
```protobuf
rpc DeregisterService(DeregisterRequest) returns (Empty)
```

#### Heartbeat
```protobuf
rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse)
```
Send a heartbeat to indicate service is alive.

#### DiscoverServices
```protobuf
rpc DiscoverServices(DiscoverRequest) returns (DiscoverResponse)
```
Find services by name or type.

#### GetService
```protobuf
rpc GetService(GetServiceRequest) returns (ServiceInfo)
```

#### ListServices
```protobuf
rpc ListServices(ListServicesRequest) returns (ListServicesResponse)
```
List all registered services.

### ServiceStatus (Enum)
- `SERVICE_STATUS_UNKNOWN`
- `SERVICE_STATUS_HEALTHY`
- `SERVICE_STATUS_UNHEALTHY`
- `SERVICE_STATUS_STARTING`
- `SERVICE_STATUS_STOPPING`

---

## Bayes Service (Logging & Metrics)

**Package:** `mdw.bayes`
**Port:** 9120

### Methods

#### Log / LogBatch
Send log entries.

#### QueryLogs / StreamLogs
Query and stream log entries.

#### RecordMetric / RecordMetricBatch
Record metrics.

#### QueryMetrics
Query recorded metrics with aggregation.

#### GetStats
Get logging and metrics statistics.

### Log Levels
- `LOG_LEVEL_DEBUG`
- `LOG_LEVEL_INFO`
- `LOG_LEVEL_WARN`
- `LOG_LEVEL_ERROR`

### Metric Types
- `METRIC_TYPE_COUNTER`
- `METRIC_TYPE_GAUGE`
- `METRIC_TYPE_HISTOGRAM`
- `METRIC_TYPE_SUMMARY`

---

## Common Types

All services share common types from `mdw.common`:

### Empty
Empty message for RPC calls with no input/output.

### HealthCheckRequest / HealthCheckResponse
Standard health check interface.

### Pagination
| Field | Type | Description |
|-------|------|-------------|
| page | int32 | Current page (1-indexed) |
| page_size | int32 | Items per page |
| total | int64 | Total items |
| total_pages | int32 | Total pages |

---

## Usage Example (Go)

```go
import (
    "context"
    turingpb "github.com/msto63/mDW/api/gen/turing"
    "google.golang.org/grpc"
)

func main() {
    // Connect to Turing service
    conn, err := grpc.Dial("localhost:9200", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := turingpb.NewTuringServiceClient(conn)

    // Send chat request
    resp, err := client.Chat(context.Background(), &turingpb.ChatRequest{
        Messages: []*turingpb.Message{
            {Role: "user", Content: "Hello, how are you?"},
        },
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

## Streaming Example

```go
// Streaming chat
stream, err := client.StreamChat(ctx, &turingpb.ChatRequest{
    Messages: []*turingpb.Message{
        {Role: "user", Content: "Write a poem about AI"},
    },
})
if err != nil {
    log.Fatal(err)
}

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(chunk.Delta)
}
```
