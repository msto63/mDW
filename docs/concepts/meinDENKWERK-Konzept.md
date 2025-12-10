# meinDENKWERK - Konzept

**Version:** 1.0
**Datum:** 2024-12-05
**Autor:** Claude Code
**Basiert auf:** RDS DENKWERK (RDW)

---

## 1. Executive Summary

**meinDENKWERK** ist eine leichtgewichtige, lokal installierbare AI-Plattform für den Einzelarbeitsplatz. Das System ist von der Enterprise-Plattform RDS DENKWERK inspiriert, wurde jedoch radikal vereinfacht für den Single-User-Betrieb ohne Authentifizierung, Multi-Tenancy oder Enterprise-Features.

### Kernziele

- **Einfachheit:** Lokale Installation ohne komplexe Infrastruktur
- **Souveränität:** Alle Daten bleiben auf dem lokalen Rechner
- **Erweiterbarkeit:** Modulare Microservice-Architektur
- **Performance:** Optimiert für Single-User-Betrieb

---

## 2. Architekturübersicht

### 2.1 Service-Landschaft

```
┌─────────────────────────────────────────────────────────────────┐
│                        meinDENKWERK                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│            ┌─────────────┐    ┌─────────────┐                  │
│            │    CLI      │    │    TUI      │                  │
│            │  (Cobra)    │    │ (Bubble Tea)│                  │
│            └──────┬──────┘    └──────┬──────┘                  │
│                   │                  │                          │
│                   └────────┬─────────┘                          │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    KANT (Gateway)                        │   │
│  │              HTTP REST + WebSocket + SSE                 │   │
│  │                      Port: 8080                          │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                    │
│         ┌──────────────────┼──────────────────┐                │
│         │                  │                  │                │
│         ▼                  ▼                  ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │   RUSSELL   │    │   TURING    │    │   HYPATIA   │        │
│  │ Orchestrat. │◄──►│  LLM Mgmt   │◄──►│     RAG     │        │
│  │  Port:9100  │    │  Port:9200  │    │  Port:9220  │        │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘        │
│         │                  │                  │                │
│         │           ┌──────┴──────┐           │                │
│         │           │             │           │                │
│         ▼           ▼             ▼           ▼                │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │   LEIBNIZ   │ │   BABBAGE   │ │    BAYES    │              │
│  │  Agentic AI │ │     NLP     │ │   Logging   │              │
│  │  Port:9140  │ │  Port:9150  │ │  Port:9120  │              │
│  └─────────────┘ └─────────────┘ └─────────────┘              │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                         CORE                             │   │
│  │    gRPC Utilities, Config, Health, Service Discovery     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                      FOUNDATION                          │   │
│  │   Logging, Error, Config, i18n, Validation, Utilities    │   │
│  │              (aus tbp/foundation)                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Service-Matrix

| Service | Port | Protokoll | Beschreibung |
|---------|------|-----------|--------------|
| **Kant** | 8080 | HTTP/WS/SSE | API Gateway, Request Routing |
| **Russell** | 9100 | gRPC | Service Discovery, Health Monitoring |
| **Turing** | 9200 | gRPC | LLM Provider Management, Inference |
| **Hypatia** | 9220 | gRPC | RAG, Vector Search, Document Ingestion |
| **Leibniz** | 9140 | gRPC | Agentic AI, Tool Orchestration |
| **Babbage** | 9150 | gRPC | NLP Processing, Text Analysis |
| **Bayes** | 9120 | gRPC | Centralized Logging, Metrics |

### 2.3 Entfernte Features (vs. RDW)

| Feature | RDW | meinDENKWERK | Begründung |
|---------|-----|--------------|------------|
| Authentifizierung | JWT + OAuth2 | Keine | Single-User, lokal |
| Multi-Tenancy | Hierarchisch | Keine | Ein Benutzer |
| Benutzerverwaltung | ASP.NET Identity | Keine | Nicht benötigt |
| Billing | Token Tracking | Keine | Kein Abrechnungsbedarf |
| Audit-Log | Vollständig | Vereinfacht | Nur lokales Logging |
| Policy Engine | OPA + Custom | Vereinfacht | Basis-Content-Filter |
| Rate Limiting | Komplex | Einfach | Lokaler Betrieb |
| Admin Dashboard | Vollständig | CLI/TUI | Vereinfacht |

---

## 3. Service-Spezifikationen

### 3.1 KANT - API Gateway

**Verantwortlichkeiten:**
- HTTP REST API für Clients
- WebSocket für Real-time Chat
- Server-Sent Events (SSE) für Streaming
- Request Routing zu Backend-Services
- Einfaches Rate Limiting (optional)

**API Endpunkte:**

```
POST   /api/v1/chat                 # Chat Completion
GET    /api/v1/chat/stream          # Streaming Chat (SSE)
POST   /api/v1/chat/conversation    # Neue Konversation
GET    /api/v1/chat/conversations   # Liste Konversationen
DELETE /api/v1/chat/conversation/:id

POST   /api/v1/rag/search           # Semantische Suche
POST   /api/v1/rag/ingest           # Dokument einlesen
GET    /api/v1/rag/collections      # Collections auflisten

POST   /api/v1/agent/execute        # Agent ausführen
GET    /api/v1/agent/stream         # Agent Streaming (SSE)
GET    /api/v1/agents               # Verfügbare Agents

POST   /api/v1/nlp/analyze          # Text analysieren
POST   /api/v1/nlp/summarize        # Text zusammenfassen
POST   /api/v1/nlp/extract          # Entitäten extrahieren

GET    /api/v1/models               # Verfügbare LLM Models
GET    /api/v1/health               # Health Check
GET    /api/v1/status               # System Status
```

**Konfiguration:**
```toml
[kant]
port = 8080
read_timeout = "30s"
write_timeout = "120s"
max_request_size = "10MB"

[kant.cors]
allowed_origins = ["http://localhost:3000"]
allowed_methods = ["GET", "POST", "DELETE"]

[kant.rate_limit]
enabled = false
requests_per_minute = 60
```

---

### 3.2 RUSSELL - Service Orchestration

**Verantwortlichkeiten:**
- Service Registry (Register/Discover)
- Health Monitoring
- Service Status Dashboard
- Graceful Shutdown Coordination

**gRPC Service:**

```protobuf
service RussellService {
  // Service Registration
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Deregister(DeregisterRequest) returns (Empty);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // Service Discovery
  rpc Discover(DiscoverRequest) returns (DiscoverResponse);
  rpc GetService(GetServiceRequest) returns (ServiceInfo);
  rpc ListServices(Empty) returns (ServiceListResponse);

  // Health
  rpc GetSystemHealth(Empty) returns (SystemHealthResponse);
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message ServiceInfo {
  string id = 1;
  string name = 2;
  string address = 3;
  int32 port = 4;
  string status = 5;  // healthy, unhealthy, unknown
  int64 last_heartbeat = 6;
  map<string, string> metadata = 7;
}
```

**Konfiguration:**
```toml
[russell]
port = 9100
health_check_interval = "10s"
heartbeat_timeout = "30s"
cleanup_interval = "60s"
```

---

### 3.3 TURING - LLM Management

**Verantwortlichkeiten:**
- Multi-Provider Abstraktion (Ollama, OpenAI, Mistral, Anthropic)
- Chat Completion (Unary + Streaming)
- Embedding Generation
- Model Discovery & Management
- Token Counting (lokal, ohne Billing)

**gRPC Service:**

```protobuf
service TuringService {
  // Chat
  rpc Chat(ChatRequest) returns (ChatResponse);
  rpc StreamChat(ChatRequest) returns (stream ChatChunk);

  // Embeddings
  rpc Embed(EmbedRequest) returns (EmbedResponse);
  rpc BatchEmbed(BatchEmbedRequest) returns (BatchEmbedResponse);

  // Model Management
  rpc ListModels(Empty) returns (ModelListResponse);
  rpc GetModel(GetModelRequest) returns (ModelInfo);
  rpc PullModel(PullModelRequest) returns (stream PullProgress);

  // Health
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message ChatRequest {
  string model = 1;
  repeated Message messages = 2;
  float temperature = 3;      // 0.0-2.0, default: 0.7
  int32 max_tokens = 4;       // default: 2048
  string system_prompt = 5;
  string conversation_id = 6;
}

message Message {
  string role = 1;    // system, user, assistant
  string content = 2;
}

message ChatResponse {
  string content = 1;
  string model = 2;
  int32 prompt_tokens = 3;
  int32 completion_tokens = 4;
  string finish_reason = 5;
  string conversation_id = 6;
}

message ChatChunk {
  string delta = 1;
  bool done = 2;
  string finish_reason = 3;
}

message EmbedRequest {
  string model = 1;
  string input = 2;
}

message EmbedResponse {
  repeated float embedding = 1;
  int32 dimensions = 2;
  int32 tokens = 3;
}
```

**Provider-Konfiguration:**
```toml
[turing]
port = 9200
default_provider = "ollama"
default_model = "llama3.2"
default_temperature = 0.7
default_max_tokens = 2048
timeout = "120s"

[turing.providers.ollama]
enabled = true
base_url = "http://localhost:11434"
models = ["llama3.2", "mistral", "codellama"]

[turing.providers.openai]
enabled = false
api_key = "${OPENAI_API_KEY}"
models = ["gpt-4o", "gpt-4o-mini"]

[turing.providers.mistral]
enabled = false
api_key = "${MISTRAL_API_KEY}"
models = ["mistral-large-latest"]

[turing.providers.anthropic]
enabled = false
api_key = "${ANTHROPIC_API_KEY}"
models = ["claude-3-5-sonnet-20241022"]
```

---

### 3.4 HYPATIA - RAG Service

**Verantwortlichkeiten:**
- Document Ingestion & Chunking
- Embedding Generation (via Turing)
- Vector Storage (SQLite + vec Extension oder Qdrant)
- Semantic Search
- Prompt Augmentation

**gRPC Service:**

```protobuf
service HypatiaService {
  // Search
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc HybridSearch(HybridSearchRequest) returns (SearchResponse);

  // Document Management
  rpc IngestDocument(IngestDocumentRequest) returns (IngestResponse);
  rpc IngestFile(stream FileChunk) returns (IngestResponse);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc GetDocument(GetDocumentRequest) returns (DocumentInfo);

  // Collection Management
  rpc CreateCollection(CreateCollectionRequest) returns (CollectionInfo);
  rpc DeleteCollection(DeleteCollectionRequest) returns (Empty);
  rpc ListCollections(Empty) returns (CollectionListResponse);

  // RAG
  rpc AugmentPrompt(AugmentPromptRequest) returns (AugmentPromptResponse);

  // Health
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message SearchRequest {
  string query = 1;
  string collection = 2;
  int32 top_k = 3;            // default: 5
  float min_score = 4;        // default: 0.7
}

message SearchResult {
  string chunk_id = 1;
  string content = 2;
  float score = 3;
  DocumentMetadata metadata = 4;
}

message IngestDocumentRequest {
  string title = 1;
  string content = 2;
  string collection = 3;
  string source = 4;
  IngestOptions options = 5;
}

message IngestOptions {
  int32 chunk_size = 1;       // default: 512
  int32 chunk_overlap = 2;    // default: 128
  string chunking_strategy = 3; // sentence, paragraph, fixed
}

message AugmentPromptRequest {
  string prompt = 1;
  string collection = 2;
  int32 top_k = 3;
  int32 max_context_tokens = 4;
}

message AugmentPromptResponse {
  string augmented_prompt = 1;
  repeated SearchResult sources = 2;
  int32 context_tokens = 3;
}
```

**Konfiguration:**
```toml
[hypatia]
port = 9220
default_collection = "default"
default_top_k = 5
min_relevance_score = 0.7

[hypatia.chunking]
default_size = 512
default_overlap = 128
strategy = "sentence"  # sentence, paragraph, fixed

[hypatia.embedding]
model = "nomic-embed-text"
dimensions = 768
cache_enabled = true
cache_ttl = "1h"

[hypatia.vectorstore]
type = "sqlite"  # sqlite, qdrant
path = "./data/vectors.db"

# Alternative: Qdrant
# [hypatia.vectorstore]
# type = "qdrant"
# url = "http://localhost:6333"
```

---

### 3.5 LEIBNIZ - Agentic AI

**Verantwortlichkeiten:**
- Agent Definition & Management
- ReAct Pattern Execution
- Tool Registry & Orchestration
- Streaming Agent Responses
- Conversation Memory

**gRPC Service:**

```protobuf
service LeibnizService {
  // Agent Management
  rpc CreateAgent(CreateAgentRequest) returns (AgentInfo);
  rpc UpdateAgent(UpdateAgentRequest) returns (AgentInfo);
  rpc DeleteAgent(DeleteAgentRequest) returns (Empty);
  rpc GetAgent(GetAgentRequest) returns (AgentInfo);
  rpc ListAgents(Empty) returns (AgentListResponse);

  // Execution
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  rpc StreamExecute(ExecuteRequest) returns (stream AgentChunk);
  rpc ContinueExecution(ContinueRequest) returns (ExecuteResponse);
  rpc CancelExecution(CancelRequest) returns (Empty);

  // Tools
  rpc ListTools(Empty) returns (ToolListResponse);
  rpc RegisterTool(RegisterToolRequest) returns (ToolInfo);

  // Health
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message AgentInfo {
  string id = 1;
  string name = 2;
  string description = 3;
  string system_prompt = 4;
  repeated string tools = 5;
  AgentConfig config = 6;
}

message AgentConfig {
  string model = 1;
  float temperature = 2;
  int32 max_iterations = 3;   // default: 10
  int32 timeout_seconds = 4;  // default: 60
  bool use_knowledge_base = 5;
  string knowledge_collection = 6;
}

message ExecuteRequest {
  string agent_id = 1;
  string message = 2;
  string conversation_id = 3;
  map<string, string> variables = 4;
}

message ExecuteResponse {
  string execution_id = 1;
  string response = 2;
  repeated AgentAction actions = 3;
  ExecutionStatus status = 4;
  int32 iterations = 5;
  int64 duration_ms = 6;
}

message AgentAction {
  string tool = 1;
  string input = 2;
  string output = 3;
  bool success = 4;
}

message AgentChunk {
  ChunkType type = 1;  // THINKING, TOOL_CALL, TOOL_RESULT, RESPONSE
  string content = 2;
  AgentAction action = 3;
}

message ToolInfo {
  string name = 1;
  string description = 2;
  string parameter_schema = 3;  // JSON Schema
  bool enabled = 4;
}
```

**Built-in Tools:**
- `web_search` - Websuche (via SearXNG oder DuckDuckGo)
- `calculator` - Mathematische Berechnungen
- `code_interpreter` - Code ausführen (sandboxed)
- `file_reader` - Lokale Dateien lesen
- `shell_command` - Shell-Befehle (eingeschränkt)

**Konfiguration:**
```toml
[leibniz]
port = 9140
max_iterations = 10
default_timeout = "60s"
enable_streaming = true

[leibniz.tools]
web_search = true
calculator = true
code_interpreter = true
file_reader = true
shell_command = false  # Sicherheitsrisiko

[leibniz.sandbox]
enabled = true
max_memory = "256MB"
max_cpu_time = "10s"
```

---

### 3.6 BABBAGE - NLP Service (NEU)

**Verantwortlichkeiten:**
- Text-Analyse (Sentiment, Entities, Keywords)
- Zusammenfassung
- Übersetzung
- Textklassifikation
- Named Entity Recognition (NER)
- Language Detection

**gRPC Service:**

```protobuf
service BabbageService {
  // Analysis
  rpc Analyze(AnalyzeRequest) returns (AnalyzeResponse);
  rpc ExtractEntities(ExtractRequest) returns (EntityResponse);
  rpc ExtractKeywords(ExtractRequest) returns (KeywordResponse);
  rpc DetectLanguage(DetectLanguageRequest) returns (LanguageResponse);

  // Transformation
  rpc Summarize(SummarizeRequest) returns (SummarizeResponse);
  rpc Translate(TranslateRequest) returns (TranslateResponse);
  rpc Classify(ClassifyRequest) returns (ClassifyResponse);

  // Sentiment
  rpc AnalyzeSentiment(SentimentRequest) returns (SentimentResponse);

  // Health
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message AnalyzeRequest {
  string text = 1;
  repeated string analyses = 2;  // sentiment, entities, keywords, language
}

message AnalyzeResponse {
  SentimentResult sentiment = 1;
  repeated Entity entities = 2;
  repeated Keyword keywords = 3;
  string language = 4;
  float language_confidence = 5;
}

message Entity {
  string text = 1;
  string type = 2;    // PERSON, ORG, LOC, DATE, MONEY, etc.
  int32 start = 3;
  int32 end = 4;
  float confidence = 5;
}

message Keyword {
  string word = 1;
  float score = 2;
}

message SummarizeRequest {
  string text = 1;
  int32 max_length = 2;       // in tokens/words
  string style = 3;           // brief, detailed, bullet_points
}

message TranslateRequest {
  string text = 1;
  string source_language = 2; // auto for detection
  string target_language = 3;
}

message ClassifyRequest {
  string text = 1;
  repeated string categories = 2;  // Vorgegebene Kategorien
}

message ClassifyResponse {
  string category = 1;
  float confidence = 2;
  repeated CategoryScore scores = 3;
}
```

**Konfiguration:**
```toml
[babbage]
port = 9150
default_language = "de"

[babbage.models]
summarization = "llama3.2"  # via Turing
translation = "llama3.2"
classification = "llama3.2"

[babbage.ner]
# Lokales NER-Modell oder via LLM
use_local_model = false
```

---

### 3.7 BAYES - Logging Service

**Verantwortlichkeiten:**
- Centralized Log Collection
- Log Querying & Filtering
- Simple Metrics (Request Count, Latency)
- Log Rotation & Retention

**gRPC Service:**

```protobuf
service BayesService {
  // Logging
  rpc Log(LogRequest) returns (Empty);
  rpc LogBatch(LogBatchRequest) returns (Empty);
  rpc QueryLogs(QueryLogsRequest) returns (QueryLogsResponse);

  // Metrics
  rpc RecordMetric(MetricRequest) returns (Empty);
  rpc QueryMetrics(QueryMetricsRequest) returns (QueryMetricsResponse);

  // Health
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
}

message LogEntry {
  string service = 1;
  string level = 2;           // DEBUG, INFO, WARN, ERROR
  string message = 3;
  int64 timestamp = 4;
  map<string, string> fields = 5;
  string request_id = 6;
}

message QueryLogsRequest {
  string service = 1;
  string level = 2;
  string search = 3;
  int64 from_timestamp = 4;
  int64 to_timestamp = 5;
  int32 limit = 6;
  int32 offset = 7;
}

message MetricEntry {
  string service = 1;
  string name = 2;
  double value = 3;
  string type = 4;            // counter, gauge, histogram
  int64 timestamp = 5;
  map<string, string> labels = 6;
}
```

**Konfiguration:**
```toml
[bayes]
port = 9120
storage_path = "./data/logs"
retention_days = 30
max_log_size = "100MB"

[bayes.rotation]
enabled = true
max_files = 10
compress = true
```

---

## 4. Shared Components

### 4.1 CORE Package

Das Core-Package enthält geteilte Funktionalität für alle Services:

```
core/
├── grpc/
│   ├── server.go           # gRPC Server Setup mit Interceptors
│   ├── client.go           # gRPC Client Factory
│   └── interceptors.go     # Logging, Recovery, Tracing
├── health/
│   ├── checker.go          # Health Check Interface
│   └── reporter.go         # Health Status Reporting
├── discovery/
│   ├── client.go           # Russell Client
│   └── registration.go     # Auto-Registration
├── config/
│   └── loader.go           # Service-spezifische Config
└── middleware/
    ├── logging.go          # Request Logging
    ├── recovery.go         # Panic Recovery
    └── tracing.go          # Request ID Propagation
```

### 4.2 FOUNDATION Package

Übernommen aus `tbp/foundation`:

```
foundation/
├── core/
│   ├── config/             # TOML/YAML Config Loading
│   ├── error/              # Strukturierte Fehlerbehandlung
│   ├── log/                # Strukturiertes Logging
│   ├── i18n/               # Internationalisierung
│   └── validation/         # Validierungs-Framework
├── utils/
│   ├── stringx/            # String Utilities
│   ├── slicex/             # Slice Utilities (Generics)
│   ├── mapx/               # Map Utilities
│   ├── mathx/              # Präzisions-Arithmetik
│   ├── timex/              # Zeit-Utilities
│   ├── filex/              # Datei-Operationen
│   └── validationx/        # Konkrete Validatoren
└── tcol/                   # Terminal Command Object Language
```

---

## 5. Datenmodell

### 5.1 Persistenz

Da meinDENKWERK lokal läuft, verwenden wir **SQLite** als primäre Datenbank:

```
data/
├── meindenkwerk.db         # Hauptdatenbank
├── vectors.db              # Vector Store (sqlite-vec)
├── logs/                   # Log Files
│   ├── bayes.log
│   └── bayes.log.1.gz
└── cache/                  # Embedding Cache
    └── embeddings.db
```

### 5.2 Schema

```sql
-- Conversations
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    title TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Messages
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT REFERENCES conversations(id),
    role TEXT NOT NULL,  -- user, assistant, system
    content TEXT NOT NULL,
    model TEXT,
    tokens INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Documents (RAG)
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    collection TEXT NOT NULL,
    title TEXT,
    source TEXT,
    content TEXT,
    chunk_count INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Chunks (RAG)
CREATE TABLE chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT REFERENCES documents(id),
    content TEXT NOT NULL,
    chunk_index INTEGER,
    embedding BLOB  -- oder via sqlite-vec
);

-- Agents
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    system_prompt TEXT,
    config JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent Executions
CREATE TABLE agent_executions (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES agents(id),
    conversation_id TEXT,
    status TEXT,
    iterations INTEGER,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. Deployment

### 6.1 Deployment-Strategie

**Primär: Podman Container (Production)**
```bash
# Alle Services starten
podman-compose up -d

# Einzelnen Service starten
podman-compose up -d kant

# Status prüfen
podman-compose ps
```

**Sekundär: Lokale Binaries (Development/Testing)**
```bash
# Alle Services lokal starten (Development)
make run-all

# Einzelnen Service starten
make run SERVICE=kant

# Mit Hot-Reload (Development)
make dev SERVICE=turing
```

### 6.2 Container-Struktur

```yaml
# podman-compose.yml
services:
  kant:
    build: ./containers/kant
    ports: ["8080:8080"]
    depends_on: [russell, turing]

  russell:
    build: ./containers/russell
    ports: ["9100:9100"]

  turing:
    build: ./containers/turing
    ports: ["9200:9200"]
    depends_on: [bayes]
    environment:
      - OLLAMA_HOST=host.containers.internal:11434

  hypatia:
    build: ./containers/hypatia
    ports: ["9220:9220"]
    volumes:
      - ./data/vectors:/data/vectors
    depends_on: [turing, bayes]

  leibniz:
    build: ./containers/leibniz
    ports: ["9140:9140"]
    depends_on: [turing, hypatia]

  babbage:
    build: ./containers/babbage
    ports: ["9150:9150"]
    depends_on: [turing]

  bayes:
    build: ./containers/bayes
    ports: ["9120:9120"]
    volumes:
      - ./data/logs:/data/logs

  # Optional: Qdrant für große Vector Stores
  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333"]
    profiles: ["qdrant"]
    volumes:
      - ./data/qdrant:/qdrant/storage
```

### 6.3 Systemanforderungen

| Ressource | Minimum | Empfohlen |
|-----------|---------|-----------|
| CPU | 4 Cores | 8 Cores |
| RAM | 8 GB | 16 GB |
| Disk | 20 GB | 100 GB |
| GPU | - | NVIDIA (für lokale LLMs) |

### 6.4 Abhängigkeiten

**Erforderlich:**
- Podman 4.0+ (oder Docker als Alternative)
- Ollama (für lokale LLMs)

**Optional:**
- Qdrant (für große Vector Stores, via `--profile qdrant`)
- SearXNG (für Web-Suche in Agents)

---

## 7. Vorschläge und Erweiterungen

### 7.1 Geplante Features

| Feature | Beschreibung | Status | Priorität |
|---------|--------------|--------|-----------|
| **CLI** | Command-Line Interface | Geplant | Hoch |
| **TUI** | Terminal UI (Bubble Tea) | Geplant | Hoch |
| **MCP Support** | Model Context Protocol | Geplant | Hoch |
| **Vector Store** | sqlite-vec + Qdrant Option | Geplant | Hoch |
| **Plugins** | Plugin-System für Tools | Später | Mittel |
| **Voice** | Whisper STT + TTS | Später | Niedrig |
| **OCR** | Dokument-Scan via Tesseract | Später | Niedrig |

### 7.2 MCP (Model Context Protocol) Integration

MCP ermöglicht standardisierte Tool-Integrationen für Agents:

```toml
[leibniz.mcp]
enabled = true

[[leibniz.mcp.servers]]
name = "filesystem"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/documents"]

[[leibniz.mcp.servers]]
name = "github"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-github"]
env = { GITHUB_TOKEN = "${GITHUB_TOKEN}" }

[[leibniz.mcp.servers]]
name = "sqlite"
command = "npx"
args = ["-y", "@modelcontextprotocol/server-sqlite", "./data/meindenkwerk.db"]
```

**MCP-Architektur in Leibniz:**
```
┌─────────────────────────────────────────────┐
│               LEIBNIZ                        │
├─────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────────────┐ │
│  │ Agent Loop  │───►│  Tool Router        │ │
│  └─────────────┘    └──────────┬──────────┘ │
│                                │            │
│         ┌──────────────────────┼───────┐    │
│         │                      │       │    │
│         ▼                      ▼       ▼    │
│  ┌─────────────┐    ┌─────────────┐  ┌────┐│
│  │ Built-in    │    │ MCP Client  │  │... ││
│  │ Tools       │    │ (stdio)     │  │    ││
│  └─────────────┘    └──────┬──────┘  └────┘│
│                            │               │
└────────────────────────────┼───────────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
            ▼                ▼                ▼
     ┌──────────┐     ┌──────────┐     ┌──────────┐
     │ MCP      │     │ MCP      │     │ MCP      │
     │ Server   │     │ Server   │     │ Server   │
     │ (FS)     │     │ (GitHub) │     │ (SQLite) │
     └──────────┘     └──────────┘     └──────────┘
```

### 7.3 Vor- und Nachteile verschiedener Ansätze

#### A) Monolithische Binary vs. Microservices

**Monolith (Single Binary):**
| Vorteile | Nachteile |
|----------|-----------|
| Einfache Installation | Größere Binary |
| Keine Netzwerk-Overhead | Weniger flexibel |
| Einfaches Deployment | Alles oder nichts |
| Weniger Ressourcen | Schwerer zu debuggen |

**Microservices (Separate Prozesse):**
| Vorteile | Nachteile |
|----------|-----------|
| Unabhängige Skalierung | Komplexeres Setup |
| Einzelne Services restarten | Mehr Ressourcen |
| Einfacher zu entwickeln | Netzwerk-Overhead |
| Bessere Isolation | Service Discovery nötig |

**Empfehlung:** Hybrid-Ansatz - Eine Binary, die alle Services startet, aber auch einzelne Services starten kann.

#### B) SQLite vs. PostgreSQL

**SQLite:**
| Vorteile | Nachteile |
|----------|-----------|
| Keine Installation | Begrenzte Concurrency |
| Portabel (eine Datei) | Keine JSON-Operationen |
| Schnell für Reads | Kein pgvector |
| Embedded | Größenbeschränkung |

**PostgreSQL:**
| Vorteile | Nachteile |
|----------|-----------|
| Volle SQL-Unterstützung | Installation nötig |
| pgvector für RAG | Mehr Ressourcen |
| Bessere Concurrency | Komplexer |
| JSON-Support | Overkill für Single-User |

**Empfehlung:** SQLite mit sqlite-vec Extension. Bei Bedarf Migration zu PostgreSQL.

#### C) gRPC vs. REST intern

**gRPC:**
| Vorteile | Nachteile |
|----------|-----------|
| Schneller (Protobuf) | Komplexer |
| Streaming | Schwerer zu debuggen |
| Type-Safety | Tooling erforderlich |
| Code-Generation | Nicht Browser-kompatibel |

**REST:**
| Vorteile | Nachteile |
|----------|-----------|
| Einfacher | Langsamer (JSON) |
| Browser-kompatibel | Kein natives Streaming |
| Leicht zu debuggen | Keine Code-Gen |
| Bekannt | Mehr Boilerplate |

**Empfehlung:** gRPC für Service-zu-Service, REST nur für Kant (Gateway).

#### D) Vector Store Optionen

**sqlite-vec:**
| Vorteile | Nachteile |
|----------|-----------|
| Keine Installation | Experimentell |
| Portabel | Begrenzte Features |
| Schnell für kleine DBs | Kein HNSW |

**Qdrant:**
| Vorteile | Nachteile |
|----------|-----------|
| Vollwertig | Separate Installation |
| HNSW-Index | Mehr RAM |
| Skalierbar | Komplexer |

**Empfehlung:** Start mit sqlite-vec, Option für Qdrant bei größeren Datenmengen.

---

## 8. Realisierungsplan

### Phase 1: Foundation & Core (Woche 1-2)

1. **Projekt-Setup**
   - Go-Modul initialisieren
   - Verzeichnisstruktur anlegen
   - Foundation aus tbp/foundation integrieren

2. **Core Package**
   - gRPC Server/Client Utilities
   - Health Check Framework
   - Service Discovery Client
   - Config Loading

3. **Proto Definitions**
   - Alle .proto Dateien definieren
   - Code-Generierung einrichten

### Phase 2: Basis-Services (Woche 3-4)

4. **Bayes (Logging)**
   - Log Ingestion
   - SQLite Storage
   - Query API

5. **Russell (Orchestration)**
   - Service Registry
   - Health Monitoring
   - Service Discovery

6. **Kant (Gateway)**
   - HTTP Router
   - gRPC Client Integration
   - Basic Endpoints

### Phase 3: AI-Services (Woche 5-7)

7. **Turing (LLM)**
   - Ollama Integration
   - Chat Completion
   - Streaming
   - Embedding Generation

8. **Hypatia (RAG)**
   - Document Ingestion
   - Chunking
   - Vector Search (sqlite-vec)
   - Prompt Augmentation

9. **Babbage (NLP)**
   - Text Analysis
   - Summarization
   - Entity Extraction

### Phase 4: Agentic AI (Woche 8-9)

10. **Leibniz (Agents)**
    - Agent Definition
    - ReAct Loop
    - Tool Integration
    - Streaming

### Phase 5: Integration & Polish (Woche 10-12)

11. **CLI**
    - `meindenkwerk serve`
    - `meindenkwerk chat`
    - `meindenkwerk agent`

12. **TUI (Optional)**
    - Bubble Tea Interface
    - Chat View
    - Agent View

13. **Testing & Dokumentation**
    - Unit Tests
    - Integration Tests
    - API Dokumentation

---

## 9. Projektstruktur

```
mDW/
├── cmd/
│   └── meindenkwerk/
│       └── main.go                 # CLI Entry Point
├── internal/
│   ├── kant/                       # API Gateway
│   │   ├── server.go
│   │   ├── handlers/
│   │   └── middleware/
│   ├── russell/                    # Service Orchestration
│   │   ├── server.go
│   │   ├── registry/
│   │   └── health/
│   ├── turing/                     # LLM Management
│   │   ├── server.go
│   │   ├── providers/
│   │   │   ├── ollama/
│   │   │   ├── openai/
│   │   │   └── anthropic/
│   │   └── streaming/
│   ├── hypatia/                    # RAG Service
│   │   ├── server.go
│   │   ├── chunking/
│   │   ├── vectorstore/
│   │   └── search/
│   ├── leibniz/                    # Agentic AI
│   │   ├── server.go
│   │   ├── agents/
│   │   ├── tools/
│   │   └── react/
│   ├── babbage/                    # NLP Service
│   │   ├── server.go
│   │   ├── analysis/
│   │   └── transform/
│   └── bayes/                      # Logging Service
│       ├── server.go
│       ├── storage/
│       └── query/
├── pkg/
│   └── core/                       # Shared Core
│       ├── grpc/
│       ├── health/
│       ├── discovery/
│       └── config/
├── api/
│   └── proto/                      # Protobuf Definitions
│       ├── kant.proto
│       ├── russell.proto
│       ├── turing.proto
│       ├── hypatia.proto
│       ├── leibniz.proto
│       ├── babbage.proto
│       └── bayes.proto
├── foundation/                     # TBP Foundation (Submodule/Copy)
├── configs/
│   ├── config.toml                 # Hauptkonfiguration
│   └── config.example.toml
├── data/                           # Runtime Data (gitignored)
│   ├── meindenkwerk.db
│   ├── vectors.db
│   └── logs/
├── docs/
│   ├── concepts/
│   │   └── meinDENKWERK-Konzept.md
│   └── api/
├── scripts/
│   ├── generate-proto.sh
│   └── build.sh
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── docker-compose.yml
├── CLAUDE.md
└── README.md
```

---

## 10. Getroffene Entscheidungen

| Frage | Entscheidung | Begründung |
|-------|--------------|------------|
| **Deployment** | Podman + lokaler Start | Container für Production, lokal für Dev/Test |
| **Frontend** | TUI + CLI | Kein Web-Frontend, Terminal-fokussiert |
| **Vector Store** | sqlite-vec + Qdrant | sqlite-vec als Default, Qdrant optional |
| **MCP Support** | Ja | Standardisierte Tool-Integration für Agents |
| **Sprache** | Deutsch + Englisch | i18n via Foundation |

---

## 11. Fazit

**meinDENKWERK** bietet eine pragmatische Lösung für Einzelanwender, die eine lokale AI-Plattform mit Souveränität über ihre Daten wünschen. Durch die Anlehnung an die bewährte RDW-Architektur profitiert das System von durchdachten Patterns, bleibt aber durch konsequente Vereinfachung leichtgewichtig und einfach zu installieren.

Die Verwendung der existierenden TBP Foundation ermöglicht einen schnellen Start mit bewährten Utilities für Logging, Konfiguration und Error Handling. Die Microservice-Architektur bleibt erhalten, wird aber durch eine Single-Binary-Option für einfaches Deployment ergänzt.
