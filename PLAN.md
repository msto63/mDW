# meinDENKWERK - Entwicklungsplan

> Letzte Aktualisierung: 2025-12-09

---

## Projektübersicht

**meinDENKWERK** ist eine vereinfachte, lokale Go-basierte KI-Plattform, abgeleitet von RDS DENKWERK (.NET).

### Kernmerkmale
- 8 Microservices mit klarer Aufgabentrennung
- Keine Authentifizierung (Single-User, lokal)
- Podman/Docker Deployment + lokale Binaries
- CLI (Cobra) + TUI (Bubble Tea)
- sqlite-vec für Vektorspeicherung
- MCP (Model Context Protocol) Unterstützung

### Service-Architektur

| Service | Port | Funktion |
|---------|------|----------|
| **Kant** | 8080 | HTTP/REST API Gateway |
| **Russell** | 9100 | Service Discovery & Health |
| **Turing** | 9200 | LLM Management (Ollama) |
| **Hypatia** | 9220 | RAG Service (Vektor-Suche) |
| **Leibniz** | 9140 | Agentic AI mit MCP |
| **Babbage** | 9150 | NLP Processing |
| **Bayes** | 9120 | Logging & Metrics |
| **Platon** | 9130 | Pipeline Processing (Pre-/Post-Processing) |

---

## Aktueller Stand

### ✅ Vollständig Implementiert

#### 1. Proto-Definitionen + Generierung (100%) ✅
```
api/proto/                    # Proto-Definitionen
├── common.proto              # Gemeinsame Typen (Empty, HealthCheck, Pagination)
├── leibniz.proto             # Agentic AI Service
├── turing.proto              # LLM Management
├── hypatia.proto             # RAG Service
├── russell.proto             # Service Discovery
├── bayes.proto               # Logging Service
├── babbage.proto             # NLP Service
└── platon.proto              # Pipeline Processing Service

api/gen/                      # Generierter Go-Code (2025-12-06)
├── common/common.pb.go
├── babbage/babbage.pb.go + babbage_grpc.pb.go
├── platon/platon.pb.go + platon_grpc.pb.go
├── bayes/bayes.pb.go + bayes_grpc.pb.go
├── hypatia/hypatia.pb.go + hypatia_grpc.pb.go
├── leibniz/leibniz.pb.go + leibniz_grpc.pb.go
├── russell/russell.pb.go + russell_grpc.pb.go
└── turing/turing.pb.go + turing_grpc.pb.go
```

#### 2. Container-Infrastruktur (100%)
- `Containerfile` - Multi-Stage Build (Alpine 3.19)
- Individuelle Containerfiles für alle 7 Services
- `podman-compose.yml` - Vollständige Orchestrierung
- Netzwerk-Isolation via `mdw-network`
- Non-Root User Security

#### 3. Makefile (100%)
```makefile
# Verfügbare Targets:
build, build-linux      # Kompilierung
run, run-all, dev       # Ausführung
test, test-coverage     # Tests
lint, fmt, vet          # Code-Qualität
proto, proto-install    # gRPC-Generierung
podman-build/up/down    # Container
clean, deps, help       # Utilities
```

#### 4. CLI (Cobra) (100%)
```
cmd/mdw/cmd/
├── root.go      - Hauptbefehl
├── chat.go      - Interaktiver Chat mit Ollama
├── search.go    - RAG-Suche und Indizierung
├── analyze.go   - NLP-Textanalyse
├── agent.go     - Agent-Aufgabenausführung
├── models.go    - LLM-Modellverwaltung
├── serve.go     - Service-Start
├── status.go    - Service-Status
└── tui.go       - TUI-Start
```

#### 5. TUI (Bubble Tea) (100%)
```
internal/tui/
├── model.go     - Hauptmodell mit 4 Views
└── styles.go    - Lipgloss-Styling
```

#### 6. Core-Pakete (100%)
```
pkg/core/
├── config/      - Konfigurationsmanagement
├── grpc/        - gRPC Server/Client Utilities
├── health/      - Health Check Registry
├── discovery/   - Service Discovery Client
└── logging/     - Strukturiertes Logging
```

#### 7. SQLite Vector Store (100%)
```
internal/hypatia/vectorstore/
├── store.go     - Interface & MemoryStore
└── sqlite.go    - SQLite-basierter Vektorspeicher
```

#### 8. Chunking (100%)
```
internal/hypatia/chunking/
└── chunker.go   - Fixed, Sentence, Paragraph, Recursive
```

#### 9. Ollama Client (100%)
```
internal/turing/ollama/
└── client.go    - Generate, Chat, Embed, Stream, ListModels
```

#### 10. MCP Client (100%)
```
internal/leibniz/mcp/
└── client.go    - JSON-RPC Client für MCP-Server
```

#### 11. Agent Framework (100%)
```
internal/leibniz/agent/
└── agent.go     - Tool Registry, ReAct Loop, Execution
```

#### 12. Platon Pipeline Service (100%) ✅
```
internal/platon/
├── server/server.go      # gRPC Server (Process, ProcessPre, ProcessPost, Handler/Pipeline/Policy Management)
├── service/service.go    # Business Logic (Pipeline, Policy, Handler Management)
├── chain/
│   ├── chain.go          # Handler-Chain (Chain-of-Responsibility Pattern)
│   ├── context.go        # Processing Context
│   └── types.go          # Type Definitions (Handler, Pipeline, AuditEntry)
└── handlers/
    ├── base.go           # BaseHandler + DynamicHandler
    ├── policy.go         # PolicyHandler (PII, Safety, Content, Custom)
    └── audit.go          # Audit Handler

Features:
- Pre-/Post-Processing Pipeline für LLM-Anfragen
- Handler-Chain mit Prioritäten und Abbruch-Logik
- Policy-basierte Validierung (Regex + LLM)
- PII-Erkennung (Email, Telefon, IBAN, Kreditkarte)
- Audit-Logging für alle Verarbeitungsschritte
- REST-API via Kant Gateway (/api/v1/platon/*)
```

#### 13. Tests
```
Getestete Pakete:
✅ pkg/core/config
✅ pkg/core/health
✅ pkg/core/discovery
✅ pkg/core/logging
✅ internal/hypatia/chunking
✅ internal/hypatia/vectorstore (Memory + SQLite)
✅ internal/babbage/service
✅ internal/turing/ollama
✅ internal/platon/chain
✅ internal/platon/handlers
✅ internal/platon/service
```

---

### 🟡 Teilweise Implementiert

#### 1. gRPC Server-Implementierungen ✅ (Phase 1+2 abgeschlossen 2025-12-06)
- ✅ Service-Strukturen existieren
- ✅ Proto-Code generiert (`api/gen/` mit allen Services)
- ✅ Russell gRPC-Handler implementiert (Register, Deregister, Heartbeat, Discover, ListServices, HealthCheck)
- ✅ Turing gRPC-Handler implementiert (Chat, StreamChat, Embed, BatchEmbed, ListModels, GetModel, PullModel, HealthCheck)
- ✅ Bayes gRPC-Handler implementiert (Log, LogBatch, QueryLogs, StreamLogs, RecordMetric, RecordMetricBatch, QueryMetrics, GetStats, HealthCheck) - inkl. Metrics Storage
- ✅ Hypatia gRPC-Handler implementiert (Search, HybridSearch, IngestDocument, IngestFile, DeleteDocument, GetDocument, ListDocuments, CreateCollection, DeleteCollection, ListCollections, GetCollectionStats, AugmentPrompt, HealthCheck)
- ✅ Leibniz gRPC-Handler implementiert (CreateAgent, UpdateAgent, DeleteAgent, GetAgent, ListAgents, Execute, StreamExecute, ContinueExecution, CancelExecution, GetExecution, ListTools, RegisterTool, UnregisterTool, HealthCheck)
- ✅ Babbage gRPC-Handler implementiert (Analyze, ExtractEntities, ExtractKeywords, DetectLanguage, Summarize, Translate, Classify, AnalyzeSentiment, HealthCheck)
- ✅ Platon gRPC-Handler implementiert (Process, ProcessPre, ProcessPost, RegisterHandler, UnregisterHandler, ListHandlers, CreatePipeline, UpdatePipeline, DeletePipeline, ListPipelines, CreatePolicy, UpdatePolicy, DeletePolicy, ListPolicies, TestPolicy, HealthCheck)

#### 2. Kant API Gateway ✅ (100% - Phase 3 abgeschlossen 2025-12-06)
- ✅ HTTP-Server-Setup
- ✅ CORS-Handling
- ✅ gRPC-Client-Manager (`internal/kant/client/clients.go`)
- ✅ REST→gRPC Handler für alle Services (inkl. Platon)
- ✅ SSE für Chat- und Agent-Streaming
- ✅ WebSocket für Echtzeit-Chat (`/api/v1/chat/ws`)
- ✅ `/api/v1/services` Endpoint für Russell.ListServices
- ✅ `/api/v1/platon/*` Endpoints für Pipeline-Processing

#### 3. CLI gRPC-Integration ✅ (100% - Phase 4.1 abgeschlossen 2025-12-06)
- ✅ gRPC-Client-Modul (`cmd/mdw/cmd/grpcclient.go`)
- ✅ `chat.go` → Turing.StreamChat (gRPC + --direct Fallback)
- ✅ `search.go` → Hypatia.Search/IngestDocument (gRPC + --direct Fallback)
- ✅ `analyze.go` → Babbage.Analyze/Summarize (gRPC + --direct Fallback)
- ✅ `agent.go` → Leibniz.Execute/StreamExecute (gRPC + --direct Fallback)
- ✅ `models.go` → Turing.ListModels (gRPC + --direct Fallback)
- ✅ `status.go` → Korrigierte Ports + gRPC Health Checks

#### 4. TUI gRPC-Integration ✅ (100% - Phase 4.2 abgeschlossen 2025-12-06)
- ✅ Chat-View → Turing.StreamChat (gRPC + Ollama Fallback)
- ✅ Search-View → Hypatia.Search (gRPC)
- ✅ Agent-View → Leibniz.Execute (gRPC)
- ✅ Status-View → Russell.ListServices (gRPC + Direktprüfung Fallback)
- ✅ Interaktive Steuerung (Tab-Wechsel, Ctrl+L, Ctrl+R)

#### 5. Integration Tests ✅ (100% - Phase 5.1 abgeschlossen 2025-12-06)
```
test/integration/
├── helpers_test.go      # Test-Utilities, Service-Check, gRPC-Dial
├── turing_test.go       # HealthCheck, ListModels, Chat, StreamChat, Embed, BatchEmbed
├── hypatia_test.go      # HealthCheck, Collections, Documents, Search, AugmentPrompt
├── leibniz_test.go      # HealthCheck, ListTools, AgentLifecycle, Execute, StreamExecute
├── babbage_test.go      # HealthCheck, Analyze, Sentiment, Keywords, Entities, Language, Summarize
├── kant_test.go         # HTTP API Tests (Health, Services, Models, Chat, Search, Agent)
└── e2e_test.go          # RAG-Workflow, Conversation, ServiceDiscovery, FullPipeline
```

#### 6. Service-zu-Service Kommunikation ✅ (Abgeschlossen 2025-12-06)
- ✅ Leibniz → Turing (LLM-Aufrufe via gRPC)
- ✅ Leibniz → Hypatia (RAG-Suche, Prompt-Augmentation)
- ✅ Leibniz → Babbage (NLP: Summarize, Sentiment, Keywords, Entities, Language)
- ✅ Hypatia → Turing (Embeddings via BatchEmbed)
- ✅ Services → Russell (Registrierung mit Heartbeat)
- ✅ Services → Bayes (Zentrales Logging)

#### 7. Konfigurationsladung ✅ (Korrigiert 2025-12-06)
- Config-Strukturen definiert
- TOML-Datei vorhanden (`configs/config.toml`)
- ✅ Zentrale Config-Ladung in `cmd/mdw/cmd/serve.go`
- ✅ Foundation Error-Handling (`mdwerror`) in allen Services integriert

---

### ❌ Nicht Implementiert

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| ~~Proto-Generierung~~ | ~~`protoc` ausführen für Go-Code~~ | ~~KRITISCH~~ ✅ Erledigt |
| ~~Russell gRPC-Handler~~ | ~~Service-Registrierung/Discovery~~ | ~~KRITISCH~~ ✅ Erledigt |
| ~~Turing gRPC-Handler~~ | ~~Chat, Embed, ListModels RPCs~~ | ~~KRITISCH~~ ✅ Erledigt |
| ~~Bayes gRPC-Handler~~ | ~~Log, Query, Metrics~~ | ~~KRITISCH~~ ✅ Erledigt |
| ~~Hypatia gRPC-Handler~~ | ~~Search, Ingest, Collections~~ | ~~HOCH~~ ✅ Erledigt |
| ~~Leibniz gRPC-Handler~~ | ~~Execute, Tool Management~~ | ~~HOCH~~ ✅ Erledigt |
| ~~Babbage gRPC-Handler~~ | ~~Analyze, Summarize, Sentiment~~ | ~~HOCH~~ ✅ Erledigt |
| ~~Kant REST-Handler~~ | ~~HTTP→gRPC Gateway vollständig~~ | ~~HOCH~~ ✅ Erledigt |
| ~~WebSocket-Streaming~~ | ~~Echtzeit-Chat~~ | ~~MITTEL~~ ✅ Erledigt |
| ~~SSE für Agent~~ | ~~Agent-Streaming-Ausgabe~~ | ~~MITTEL~~ ✅ Erledigt |
| ~~Integrationstests~~ | ~~End-to-End Tests~~ | ~~MITTEL~~ ✅ Erledigt |
| ~~API-Dokumentation~~ | ~~OpenAPI/Swagger~~ | ~~NIEDRIG~~ ✅ Erledigt |
| ~~Service-zu-Service~~ | ~~gRPC-Aufrufe zwischen Services~~ | ~~HOCH~~ ✅ Erledigt |

---

## Entwicklungsphasen

### Phase 1: System Lauffähig Machen (Woche 1)

**Ziel:** Grundlegende gRPC-Kommunikation zwischen Services

#### 1.1 Proto-Code Generierung
```bash
# Protoc installieren
brew install protobuf  # macOS
# oder
apt-get install protobuf-compiler  # Linux

# Go-Plugins installieren
make proto-install

# Code generieren
make proto
```

**Erwartetes Ergebnis:**
```
api/gen/
├── common/
├── russell/
├── turing/
├── hypatia/
├── leibniz/
├── babbage/
└── bayes/
```

#### 1.2 Russell Service - Service Discovery
```go
// internal/russell/server/server.go
// Zu implementieren:
- RegisterService(ctx, *RegisterRequest) (*RegisterResponse, error)
- DeregisterService(ctx, *DeregisterRequest) (*Empty, error)
- Heartbeat(ctx, *HeartbeatRequest) (*HeartbeatResponse, error)
- DiscoverServices(ctx, *DiscoverRequest) (*DiscoverResponse, error)
- GetService(ctx, *GetServiceRequest) (*ServiceInfo, error)
- ListServices(ctx, *ListServicesRequest) (*ListServicesResponse, error)
```

#### 1.3 Turing Service - LLM Management
```go
// internal/turing/server/server.go
// Zu implementieren:
- Chat(ctx, *ChatRequest) (*ChatResponse, error)
- ChatStream(*ChatRequest, stream) error
- Generate(ctx, *GenerateRequest) (*GenerateResponse, error)
- GenerateStream(*GenerateRequest, stream) error
- Embed(ctx, *EmbedRequest) (*EmbedResponse, error)
- ListModels(ctx, *Empty) (*ListModelsResponse, error)
```

#### 1.4 Bayes Service - Logging
```go
// internal/bayes/server/server.go
// Zu implementieren:
- Log(ctx, *LogRequest) (*Empty, error)
- Query(ctx, *QueryRequest) (*QueryResponse, error)
- GetStats(ctx, *Empty) (*StatsResponse, error)
```

**Geschätzte LOC:** ~1500-2000

---

### Phase 2: Kern-Services (Woche 2-3)

**Ziel:** Alle Services mit vollständigen gRPC-Handlern

#### 2.1 Hypatia Service - RAG
```go
// Zu implementieren:
- Search(ctx, *SearchRequest) (*SearchResponse, error)
- Ingest(ctx, *IngestRequest) (*IngestResponse, error)
- IngestStream(stream) (*IngestResponse, error)
- GetDocument(ctx, *GetDocumentRequest) (*Document, error)
- DeleteDocument(ctx, *DeleteDocumentRequest) (*Empty, error)
- ListCollections(ctx, *Empty) (*ListCollectionsResponse, error)
- CreateCollection(ctx, *CreateCollectionRequest) (*Empty, error)
- DeleteCollection(ctx, *DeleteCollectionRequest) (*Empty, error)
```

#### 2.2 Leibniz Service - Agentic AI
```go
// Zu implementieren:
- Execute(ctx, *ExecuteRequest) (*ExecuteResponse, error)
- ExecuteStream(*ExecuteRequest, stream) error
- ListTools(ctx, *Empty) (*ListToolsResponse, error)
- RegisterTool(ctx, *RegisterToolRequest) (*Empty, error)
- ConnectMCP(ctx, *ConnectMCPRequest) (*Empty, error)
- DisconnectMCP(ctx, *DisconnectMCPRequest) (*Empty, error)
```

#### 2.3 Babbage Service - NLP
```go
// Zu implementieren:
- Analyze(ctx, *AnalyzeRequest) (*AnalyzeResponse, error)
- Sentiment(ctx, *SentimentRequest) (*SentimentResponse, error)
- ExtractEntities(ctx, *EntitiesRequest) (*EntitiesResponse, error)
- ExtractKeywords(ctx, *KeywordsRequest) (*KeywordsResponse, error)
- Summarize(ctx, *SummarizeRequest) (*SummarizeResponse, error)
- DetectLanguage(ctx, *DetectLanguageRequest) (*DetectLanguageResponse, error)
```

#### 2.4 Inter-Service Kommunikation
```go
// Beispiel: Leibniz ruft Turing für LLM auf
func (s *Service) callTuring(ctx context.Context, prompt string) (string, error) {
    conn, err := grpc.Dial(s.turingAddr, grpc.WithInsecure())
    if err != nil {
        return "", err
    }
    defer conn.Close()

    client := turingpb.NewTuringServiceClient(conn)
    resp, err := client.Generate(ctx, &turingpb.GenerateRequest{
        Prompt: prompt,
        Model:  s.config.DefaultModel,
    })
    return resp.Response, err
}
```

**Geschätzte LOC:** ~2000-2500

---

### Phase 3: API Gateway (Woche 3-4)

**Ziel:** Vollständiges HTTP-Interface über Kant

#### 3.1 REST-Endpunkte
```
POST /api/v1/chat              → Turing.Chat
POST /api/v1/chat/stream       → Turing.ChatStream (SSE)
POST /api/v1/generate          → Turing.Generate
GET  /api/v1/models            → Turing.ListModels

POST /api/v1/search            → Hypatia.Search
POST /api/v1/ingest            → Hypatia.Ingest
GET  /api/v1/collections       → Hypatia.ListCollections
POST /api/v1/collections       → Hypatia.CreateCollection

POST /api/v1/agent/execute     → Leibniz.Execute
GET  /api/v1/agent/execute/:id → Leibniz.ExecuteStream (SSE)
GET  /api/v1/agent/tools       → Leibniz.ListTools

POST /api/v1/analyze           → Babbage.Analyze
POST /api/v1/summarize         → Babbage.Summarize
POST /api/v1/sentiment         → Babbage.Sentiment

GET  /api/v1/health            → Russell.HealthCheck
GET  /api/v1/services          → Russell.ListServices
```

#### 3.2 WebSocket für Chat
```go
// internal/kant/handler/websocket.go
func (h *Handler) handleChatWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)
    defer conn.Close()

    for {
        _, msg, _ := conn.ReadMessage()
        // Stream zu Turing
        stream, _ := h.turingClient.ChatStream(ctx, &ChatRequest{...})
        for {
            resp, err := stream.Recv()
            if err == io.EOF { break }
            conn.WriteJSON(resp)
        }
    }
}
```

#### 3.3 SSE für Agent-Streaming
```go
// internal/kant/handler/sse.go
func (h *Handler) handleAgentSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    flusher := w.(http.Flusher)

    stream, _ := h.leibnizClient.ExecuteStream(ctx, &ExecuteRequest{...})
    for {
        step, err := stream.Recv()
        if err == io.EOF { break }
        fmt.Fprintf(w, "data: %s\n\n", json.Marshal(step))
        flusher.Flush()
    }
}
```

**Geschätzte LOC:** ~1000-1500

---

### Phase 4: Vervollständigung (Woche 4-5)

**Ziel:** CLI und TUI vollständig funktional

#### 4.1 TUI Views Vervollständigen
```go
// internal/tui/views/
├── chat.go      - Chat mit Turing (Streaming)
├── search.go    - RAG-Suche mit Hypatia
├── agent.go     - Agent-Ausführung mit Leibniz
└── status.go    - Service-Status von Russell
```

#### 4.2 CLI-Integration
```go
// cmd/mdw/cmd/chat.go
// Verbindung zu Turing Service herstellen
func runChat(cmd *cobra.Command, args []string) error {
    conn, _ := grpc.Dial(cfg.TuringAddr, ...)
    client := turingpb.NewTuringServiceClient(conn)

    // Chat-Loop mit Streaming
    stream, _ := client.ChatStream(ctx, &ChatRequest{...})
    for {
        resp, err := stream.Recv()
        if err == io.EOF { break }
        fmt.Print(resp.Content)
    }
}
```

#### 4.3 Konfiguration Vollständig Laden
```go
// pkg/core/config/loader.go
func LoadAndValidate() (*Config, error) {
    cfg, err := Load("configs/config.toml")
    if err != nil {
        return nil, err
    }

    // Umgebungsvariablen expandieren
    cfg.expandEnvVars()

    // Validieren
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}
```

**Geschätzte LOC:** ~1500-2000

---

### Phase 5: Testing & Polish (Woche 5-6)

**Ziel:** Produktionsreife

#### 5.1 Integrationstests
```go
// test/integration/
├── turing_test.go      - LLM-Service Tests
├── hypatia_test.go     - RAG-Service Tests
├── leibniz_test.go     - Agent-Service Tests
├── kant_test.go        - API Gateway Tests
└── e2e_test.go         - End-to-End Workflow
```

#### 5.2 Performance-Optimierung
- Connection Pooling für gRPC
- Caching für häufige Anfragen
- Batch-Processing für Embeddings
- Indexierung für Vektorsuche

#### 5.3 Dokumentation
- OpenAPI/Swagger für REST-API
- gRPC-Service-Dokumentation
- Deployment-Guide
- Troubleshooting-Guide

---

## Code-Metriken

| Kategorie | Dateien | LOC |
|-----------|---------|-----|
| Internal Go | 35 | ~11,500 |
| Proto-Definitionen | 8 | ~750 |
| Tests | 60 | ~3,500 |
| **Gesamt** | **103** | **~15,750** |

### Geschätzte Arbeit

| Phase | LOC | Aufwand |
|-------|-----|---------|
| Phase 1 | ~2,000 | 1 Woche |
| Phase 2 | ~2,500 | 2 Wochen |
| Phase 3 | ~1,500 | 1 Woche |
| Phase 4 | ~2,000 | 1 Woche |
| Phase 5 | ~1,500 | 1 Woche |
| **Gesamt** | **~9,500** | **6 Wochen** |

---

## Risiken & Mitigationen

| Risiko | Impact | Mitigation |
|--------|--------|------------|
| Proto nicht generiert | KRITISCH | Zuerst `make proto` ausführen |
| Ollama nicht verfügbar | HOCH | Mock-Client für Tests |
| Service-Timeouts | MITTEL | Circuit Breaker implementieren |
| Memory bei großen Vektoren | MITTEL | SQLite-Paginierung |

---

## Aktueller Stand: Projekt Abgeschlossen

**Alle Phasen sind erfolgreich abgeschlossen!**

### Phase 5 - Zusammenfassung:

#### 5.1 Integrationstests ✅
- Turing, Hypatia, Leibniz, Babbage Service-Tests
- Kant API Gateway Tests
- End-to-End Workflow Tests

#### 5.2 Performance-Optimierung ✅
- **gRPC Connection Pooling** - Thread-sichere Connection-Verwaltung mit Global Singleton
- **Caching-Layer** - TTL-basierter Cache für Models und Embeddings
- **Batch-Processing** - Embedding-Sharding für große Requests (256 Texte pro Batch)
- **Vektorsuche-Indexierung** - Pre-computed Norms + Min-Heap für effizientes Top-K

#### 5.3 Dokumentation ✅
- **OpenAPI/Swagger** - `docs/openapi.yaml` - Vollständige REST-API Spezifikation
- **gRPC-Docs** - `docs/grpc-services.md` - Alle Services dokumentiert
- **Deployment-Guide** - `docs/deployment.md` - Podman/Docker Setup
- **Troubleshooting-Guide** - `docs/troubleshooting.md` - Häufige Probleme und Lösungen

### Dokumentation

```
docs/
├── openapi.yaml        # OpenAPI 3.0 Spezifikation
├── grpc-services.md    # gRPC Service-Dokumentation
├── deployment.md       # Deployment-Anleitung
└── troubleshooting.md  # Troubleshooting-Guide
```

---

## Changelog

| Datum | Änderung |
|-------|----------|
| 2025-12-06 | Initiale Erstellung des Plans |
| 2025-12-06 | CLI, TUI, Tests, sqlite-vec abgeschlossen |
| 2025-12-06 | Foundation-Integration: Error-Handling + zentrale Config |
| 2025-12-06 | **Phase 1.1 abgeschlossen:** Proto-Generierung (protobuf 33.1) |
| 2025-12-06 | **Phase 1.2-1.4 abgeschlossen:** Russell, Turing, Bayes gRPC-Handler implementiert |
| 2025-12-06 | **Phase 2.1-2.3 abgeschlossen:** Hypatia, Leibniz, Babbage gRPC-Handler implementiert |
| 2025-12-06 | **Phase 3 abgeschlossen:** Kant API Gateway (REST→gRPC, SSE, WebSocket) implementiert |
| 2025-12-06 | **Phase 4.1 abgeschlossen:** CLI gRPC-Integration (chat, search, analyze, agent, models, status) |
| 2025-12-06 | **Phase 4.2 abgeschlossen:** TUI gRPC-Integration (Chat, Search, Agent, Status Views) |
| 2025-12-06 | **Phase 5.1 abgeschlossen:** Integrationstests für alle Services (Turing, Hypatia, Leibniz, Babbage, Kant, E2E) |
| 2025-12-06 | **Lücken geschlossen:** Bayes Metrics Storage + Kant /api/v1/services Endpoint |
| 2025-12-06 | **Phase 5.2 abgeschlossen:** Performance-Optimierung (Connection Pooling, Caching, Batch-Processing, Vektorsuche-Indexierung) |
| 2025-12-06 | **Phase 5.3 abgeschlossen:** Dokumentation (OpenAPI, gRPC-Docs, Deployment-Guide, Troubleshooting-Guide) |
| 2025-12-06 | **Service-zu-Service Kommunikation:** Leibniz→Turing/Hypatia/Babbage, Russell-Registrierung, Bayes-Logging |
| 2025-12-09 | **Platon Service dokumentiert:** Pipeline Processing Service mit Handler-Chain, Policy Management, PII-Erkennung |
| 2025-12-09 | **8 Services:** Kant, Russell, Turing, Hypatia, Leibniz, Babbage, Bayes, Platon |
