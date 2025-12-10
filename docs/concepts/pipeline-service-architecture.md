# Pipeline Service Architektur

## Konzept fuer einen dedizierten Pipeline-Service in mDW

**Version:** 1.0
**Datum:** 2025-12-08
**Status:** Entwurf
**Autor:** Mike Stoffels mit Claude

---

## 1. Executive Summary

Dieses Konzept beschreibt die Auslagerung der Pipeline-Funktionalitaet aus dem Leibniz-Service in einen dedizierten Service. Die Pipeline ist die zentrale Komponente fuer die Verarbeitung, Validierung und Transformation von Prompts und Responses - eine Kernfunktion fuer die Arbeit mit KI.

### 1.1 Kernaussagen

- Die Pipeline wird aus Leibniz extrahiert und als eigenstaendiger Service implementiert
- Der neue Service heisst **"Platon"** (Philosoph der Wahrheit und Ideale)
- Andere Services (Policy-Engine, Agents, etc.) koennen sich ueber ein Plugin-System in die Pipeline einklinken
- Die Architektur folgt dem Chain-of-Responsibility-Pattern

### 1.2 Vorteile

| Vorteil | Beschreibung |
|---------|--------------|
| **Separation of Concerns** | Leibniz fokussiert auf Agentic AI, Platon auf Processing |
| **Wiederverwendbarkeit** | Pipeline kann von allen Services genutzt werden |
| **Erweiterbarkeit** | Neue Use Cases koennen einfach hinzugefuegt werden |
| **Skalierbarkeit** | Unabhaengige Skalierung moeglich |
| **Testbarkeit** | Kleinere, fokussierte Komponenten |

---

## 2. Ist-Zustand Analyse

### 2.1 Aktuelle Architektur

```
                          AKTUELL (IST)
+----------------------------------------------------------------------+
|                                                                       |
|   Kant (Gateway)                                                      |
|       |                                                               |
|       v                                                               |
|   +------------------+                                                |
|   |     Leibniz      |  <-- Problem: Zu viele Verantwortlichkeiten   |
|   +------------------+                                                |
|   | - Agent Runtime  |                                                |
|   | - MCP Client     |                                                |
|   | - Pipeline       |  <-- Sollte hier nicht sein                   |
|   | - Policy Engine  |  <-- Sollte hier nicht sein                   |
|   | - Audit Logging  |  <-- Sollte hier nicht sein                   |
|   +------------------+                                                |
|           |                                                           |
|           v                                                           |
|   +------------------+                                                |
|   |     Turing       |                                                |
|   |   (LLM Service)  |                                                |
|   +------------------+                                                |
|                                                                       |
+----------------------------------------------------------------------+
```

### 2.2 Probleme der aktuellen Architektur

1. **Single Responsibility Principle verletzt**
   - Leibniz macht zu viel: Agents, MCP, Pipeline, Policies
   - Schwer zu warten und zu testen

2. **Enge Kopplung**
   - Pipeline ist fest in Leibniz integriert
   - Kann nicht unabhaengig genutzt werden

3. **Begrenzte Erweiterbarkeit**
   - Neue Use Cases muessen immer in Leibniz eingebaut werden
   - Kein Plugin-Mechanismus

4. **Skalierungsprobleme**
   - Pipeline und Agent-Runtime skalieren gemeinsam
   - Keine granulare Ressourcenzuweisung

---

## 3. Ziel-Architektur

### 3.1 Ueberblick

```
                          ZIEL-ARCHITEKTUR (SOLL)
+------------------------------------------------------------------------------+
|                                                                               |
|   +--------+                                                                  |
|   |  Kant  |  HTTP/REST                                                       |
|   +--------+                                                                  |
|       |                                                                       |
|       v                                                                       |
|   +------------------+     +------------------+     +------------------+      |
|   |     Platon       |<--->|    Leibniz       |<--->|     Turing       |      |
|   |    (Pipeline)    |     |    (Agents)      |     |      (LLM)       |      |
|   +------------------+     +------------------+     +------------------+      |
|           |                                                                   |
|           |  Plugin-Chain                                                     |
|           |                                                                   |
|   +-------+-------+-------+-------+-------+                                   |
|   |       |       |       |       |       |                                   |
|   v       v       v       v       v       v                                   |
| +-----+ +-----+ +-----+ +-----+ +-----+ +-----+                               |
| |Polic| |Audit| |Route| |Trans| |Valid| | ... |  <-- Pluggable Handlers      |
| |  y  | |     | |  r  | |form | |ate  | |     |                               |
| +-----+ +-----+ +-----+ +-----+ +-----+ +-----+                               |
|                                                                               |
+------------------------------------------------------------------------------+
```

### 3.2 Service-Verantwortlichkeiten

| Service | Port | Verantwortlichkeit |
|---------|------|-------------------|
| **Kant** | 8080 | HTTP Gateway, Request Routing |
| **Russell** | 9100 | Service Discovery, Orchestrierung |
| **Bayes** | 9120 | Logging, Metriken |
| **Platon** | 9130 | **NEU:** Pipeline Processing, Chain Management |
| **Leibniz** | 9140 | Agentic AI, MCP Client |
| **Babbage** | 9150 | NLP Processing |
| **Turing** | 9200 | LLM Management |
| **Hypatia** | 9220 | RAG, Vector Search |

### 3.3 Request Flow

```
Sequenzdiagramm: Chat-Anfrage mit Pipeline

User        Kant       Platon      Leibniz      Turing
  |           |           |           |           |
  |--Request->|           |           |           |
  |           |--Process->|           |           |
  |           |           |           |           |
  |           |  [Pre-Processing Chain]           |
  |           |  +------------------------+       |
  |           |  | 1. Policy Check        |       |
  |           |  | 2. Context Enrichment  |       |
  |           |  | 3. Prompt Transform    |       |
  |           |  +------------------------+       |
  |           |           |           |           |
  |           |           |--Execute->|           |
  |           |           |           |--Chat---->|
  |           |           |           |<--Response|
  |           |           |<--Result--|           |
  |           |           |           |           |
  |           |  [Post-Processing Chain]          |
  |           |  +------------------------+       |
  |           |  | 1. Response Validation |       |
  |           |  | 2. Content Filter      |       |
  |           |  | 3. Audit Logging       |       |
  |           |  +------------------------+       |
  |           |           |           |           |
  |           |<--Result--|           |           |
  |<-Response-|           |           |           |
```

---

## 4. Platon Service Design

### 4.1 Kernkomponenten

```
internal/platon/
|-- server/
|   |-- server.go           # gRPC Server
|   |-- grpc_handlers.go    # gRPC Methoden
|
|-- service/
|   |-- service.go          # Business Logic
|   |-- config.go           # Konfiguration
|
|-- chain/
|   |-- chain.go            # Chain-of-Responsibility
|   |-- handler.go          # Handler Interface
|   |-- context.go          # Processing Context
|
|-- handlers/               # Built-in Handler
|   |-- policy_handler.go   # Policy Enforcement
|   |-- audit_handler.go    # Audit Logging
|   |-- transform_handler.go # Transformationen
|   |-- routing_handler.go  # Routing Decisions
|   |-- validation_handler.go # Validierung
|
|-- registry/
|   |-- registry.go         # Handler Registry
|   |-- plugin.go           # Plugin Loading
|
|-- store/
|   |-- store.go            # Persistenz Interface
|   |-- sqlite_store.go     # SQLite Implementation
```

### 4.2 Handler Interface

```go
// Handler ist das zentrale Interface fuer Pipeline-Schritte
type Handler interface {
    // Name gibt den eindeutigen Namen des Handlers zurueck
    Name() string

    // Type gibt den Handler-Typ zurueck (pre, post, both)
    Type() HandlerType

    // Priority bestimmt die Ausfuehrungsreihenfolge (niedriger = frueher)
    Priority() int

    // Process verarbeitet den Request/Response
    Process(ctx *ProcessingContext) error

    // ShouldProcess entscheidet, ob dieser Handler ausgefuehrt werden soll
    ShouldProcess(ctx *ProcessingContext) bool
}

// HandlerType definiert wann der Handler ausgefuehrt wird
type HandlerType int

const (
    HandlerTypePre  HandlerType = iota  // Vor der Hauptverarbeitung
    HandlerTypePost                      // Nach der Hauptverarbeitung
    HandlerTypeBoth                      // Vor und nach
)

// ProcessingContext enthaelt alle Informationen waehrend der Verarbeitung
type ProcessingContext struct {
    // Request-Daten
    RequestID    string
    Prompt       string
    Metadata     map[string]interface{}

    // Response-Daten (nach Hauptverarbeitung)
    Response     string

    // Processing State
    Phase        ProcessingPhase  // Pre oder Post
    Blocked      bool
    BlockReason  string
    Modified     bool

    // Shared State zwischen Handlern
    State        map[string]interface{}

    // Logging & Audit
    AuditLog     []AuditEntry
    StartTime    time.Time
}
```

### 4.3 Chain-of-Responsibility Implementation

```go
// Chain verwaltet die Handler-Kette
type Chain struct {
    preHandlers  []Handler
    postHandlers []Handler
    logger       logging.Logger
}

// NewChain erstellt eine neue Chain
func NewChain(logger logging.Logger) *Chain {
    return &Chain{
        preHandlers:  make([]Handler, 0),
        postHandlers: make([]Handler, 0),
        logger:       logger,
    }
}

// Register fuegt einen Handler hinzu
func (c *Chain) Register(h Handler) {
    switch h.Type() {
    case HandlerTypePre:
        c.preHandlers = append(c.preHandlers, h)
        c.sortByPriority(c.preHandlers)
    case HandlerTypePost:
        c.postHandlers = append(c.postHandlers, h)
        c.sortByPriority(c.postHandlers)
    case HandlerTypeBoth:
        c.preHandlers = append(c.preHandlers, h)
        c.postHandlers = append(c.postHandlers, h)
        c.sortByPriority(c.preHandlers)
        c.sortByPriority(c.postHandlers)
    }
}

// ProcessPre fuehrt die Pre-Processing Chain aus
func (c *Chain) ProcessPre(ctx *ProcessingContext) error {
    ctx.Phase = PhasePre
    return c.processChain(ctx, c.preHandlers)
}

// ProcessPost fuehrt die Post-Processing Chain aus
func (c *Chain) ProcessPost(ctx *ProcessingContext) error {
    ctx.Phase = PhasePost
    return c.processChain(ctx, c.postHandlers)
}

func (c *Chain) processChain(ctx *ProcessingContext, handlers []Handler) error {
    for _, h := range handlers {
        if ctx.Blocked {
            c.logger.Info("Chain aborted - request blocked",
                "handler", h.Name(),
                "reason", ctx.BlockReason)
            return nil
        }

        if !h.ShouldProcess(ctx) {
            continue
        }

        start := time.Now()
        err := h.Process(ctx)
        duration := time.Since(start)

        ctx.AuditLog = append(ctx.AuditLog, AuditEntry{
            Handler:   h.Name(),
            Phase:     ctx.Phase,
            Duration:  duration,
            Error:     err,
            Modified:  ctx.Modified,
        })

        if err != nil {
            return err
        }
    }
    return nil
}
```

### 4.4 Built-in Handler Beispiele

#### Policy Handler

```go
// PolicyHandler prueft Policies gegen den Content
type PolicyHandler struct {
    engine *PolicyEngine
    logger logging.Logger
}

func (h *PolicyHandler) Name() string { return "policy" }
func (h *PolicyHandler) Type() HandlerType { return HandlerTypeBoth }
func (h *PolicyHandler) Priority() int { return 10 } // Frueh ausfuehren

func (h *PolicyHandler) Process(ctx *ProcessingContext) error {
    var text string
    if ctx.Phase == PhasePre {
        text = ctx.Prompt
    } else {
        text = ctx.Response
    }

    result := h.engine.Check(text)

    if result.Decision == DecisionBlock {
        ctx.Blocked = true
        ctx.BlockReason = result.Reason
        return nil
    }

    if result.Decision == DecisionRedact {
        if ctx.Phase == PhasePre {
            ctx.Prompt = result.ModifiedText
        } else {
            ctx.Response = result.ModifiedText
        }
        ctx.Modified = true
    }

    return nil
}

func (h *PolicyHandler) ShouldProcess(ctx *ProcessingContext) bool {
    // Policy-Check immer ausfuehren
    return true
}
```

#### Audit Handler

```go
// AuditHandler protokolliert alle Verarbeitungsschritte
type AuditHandler struct {
    store  AuditStore
    logger logging.Logger
}

func (h *AuditHandler) Name() string { return "audit" }
func (h *AuditHandler) Type() HandlerType { return HandlerTypePost }
func (h *AuditHandler) Priority() int { return 1000 } // Am Ende ausfuehren

func (h *AuditHandler) Process(ctx *ProcessingContext) error {
    entry := &AuditRecord{
        RequestID:  ctx.RequestID,
        Timestamp:  time.Now(),
        Duration:   time.Since(ctx.StartTime),
        Prompt:     ctx.Prompt,
        Response:   ctx.Response,
        Blocked:    ctx.Blocked,
        Modified:   ctx.Modified,
        AuditLog:   ctx.AuditLog,
        Metadata:   ctx.Metadata,
    }

    return h.store.Save(ctx.RequestID, entry)
}
```

---

## 5. Integration mit bestehenden Services

### 5.1 Kant Integration

Kant ruft Platon vor dem eigentlichen Service-Call auf:

```go
// handler/chat.go - Kant Chat Handler

func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. Parse Request
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 2. Pre-Processing via Platon
    preResult, err := h.platonClient.ProcessPre(ctx, &platon.PreRequest{
        RequestID: uuid.New().String(),
        Prompt:    req.Message,
        Metadata:  req.Metadata,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 2a. Pruefe ob blockiert
    if preResult.Blocked {
        json.NewEncoder(w).Encode(ChatResponse{
            Error: preResult.BlockReason,
        })
        return
    }

    // 3. Hauptverarbeitung via Turing/Leibniz
    llmResp, err := h.turingClient.Chat(ctx, &turing.ChatRequest{
        Message: preResult.ProcessedPrompt,
        Model:   req.Model,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 4. Post-Processing via Platon
    postResult, err := h.platonClient.ProcessPost(ctx, &platon.PostRequest{
        RequestID: preResult.RequestID,
        Response:  llmResp.Message,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 5. Response
    json.NewEncoder(w).Encode(ChatResponse{
        Message:  postResult.ProcessedResponse,
        Metadata: postResult.Metadata,
    })
}
```

### 5.2 Leibniz Integration

Leibniz kann Platon fuer Agent-spezifische Processing nutzen:

```go
// Leibniz Agent Execute
func (s *Service) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    // Optional: Agent-spezifisches Pre-Processing
    if req.PipelineID != "" {
        preResult, err := s.platonClient.ProcessPre(ctx, &platon.PreRequest{
            PipelineID: req.PipelineID,
            Prompt:     req.Task,
        })
        if err != nil {
            return nil, err
        }
        req.Task = preResult.ProcessedPrompt
    }

    // Agent ausfuehren
    result, err := s.agent.Execute(ctx, req.Task)

    // Optional: Agent-spezifisches Post-Processing
    if req.PipelineID != "" {
        postResult, err := s.platonClient.ProcessPost(ctx, &platon.PostRequest{
            PipelineID: req.PipelineID,
            Response:   result.Output,
        })
        if err != nil {
            return nil, err
        }
        result.Output = postResult.ProcessedResponse
    }

    return result, nil
}
```

### 5.3 gRPC Service Definition

```protobuf
// api/proto/platon.proto

syntax = "proto3";

package mdw.platon;

option go_package = "github.com/msto63/mDW/api/gen/platon";

service PlatonService {
    // Pipeline Management
    rpc ListPipelines(ListPipelinesRequest) returns (ListPipelinesResponse);
    rpc GetPipeline(GetPipelineRequest) returns (Pipeline);
    rpc CreatePipeline(CreatePipelineRequest) returns (Pipeline);
    rpc UpdatePipeline(UpdatePipelineRequest) returns (Pipeline);
    rpc DeletePipeline(DeletePipelineRequest) returns (DeletePipelineResponse);

    // Handler Management
    rpc ListHandlers(ListHandlersRequest) returns (ListHandlersResponse);
    rpc RegisterHandler(RegisterHandlerRequest) returns (Handler);
    rpc UnregisterHandler(UnregisterHandlerRequest) returns (UnregisterHandlerResponse);

    // Processing
    rpc ProcessPre(ProcessPreRequest) returns (ProcessPreResponse);
    rpc ProcessPost(ProcessPostRequest) returns (ProcessPostResponse);
    rpc Process(ProcessRequest) returns (ProcessResponse);  // Kompletter Durchlauf

    // Policy Management (delegiert an Policy-Handler)
    rpc ListPolicies(ListPoliciesRequest) returns (ListPoliciesResponse);
    rpc CreatePolicy(CreatePolicyRequest) returns (Policy);
    rpc TestPolicy(TestPolicyRequest) returns (TestPolicyResponse);

    // Audit
    rpc GetAuditLog(GetAuditLogRequest) returns (GetAuditLogResponse);
}

message ProcessPreRequest {
    string request_id = 1;
    string pipeline_id = 2;  // Optional: Spezifische Pipeline
    string prompt = 3;
    map<string, string> metadata = 4;
}

message ProcessPreResponse {
    string request_id = 1;
    string processed_prompt = 2;
    bool blocked = 3;
    string block_reason = 4;
    bool modified = 5;
    repeated HandlerResult handler_results = 6;
}

message ProcessPostRequest {
    string request_id = 1;
    string pipeline_id = 2;
    string response = 3;
}

message ProcessPostResponse {
    string request_id = 1;
    string processed_response = 2;
    bool blocked = 3;
    string block_reason = 4;
    bool modified = 5;
    repeated HandlerResult handler_results = 6;
}

message HandlerResult {
    string handler_name = 1;
    int64 duration_ms = 2;
    bool modified = 3;
    string error = 4;
}

message Pipeline {
    string id = 1;
    string name = 2;
    string description = 3;
    bool enabled = 4;
    repeated string pre_handlers = 5;   // Handler-Namen fuer Pre-Processing
    repeated string post_handlers = 6;  // Handler-Namen fuer Post-Processing
    map<string, string> config = 7;     // Handler-spezifische Konfiguration
}

message Handler {
    string name = 1;
    string type = 2;  // "pre", "post", "both"
    int32 priority = 3;
    string description = 4;
    bool enabled = 5;
    map<string, string> config = 6;
}
```

---

## 6. Plugin-System fuer externe Handler

### 6.1 Plugin Interface

Externe Services koennen eigene Handler registrieren:

```go
// Plugin-Registrierung via gRPC
type HandlerPlugin struct {
    Name        string
    Type        HandlerType
    Priority    int
    Endpoint    string  // gRPC Endpoint des Handlers
    Description string
}

// Platon ruft externe Handler via gRPC auf
type ExternalHandler struct {
    plugin HandlerPlugin
    client ExternalHandlerClient
}

func (h *ExternalHandler) Process(ctx *ProcessingContext) error {
    req := &ExternalProcessRequest{
        RequestID: ctx.RequestID,
        Phase:     ctx.Phase.String(),
        Prompt:    ctx.Prompt,
        Response:  ctx.Response,
        Metadata:  ctx.Metadata,
        State:     ctx.State,
    }

    resp, err := h.client.Process(ctx, req)
    if err != nil {
        return err
    }

    // Update Context mit Ergebnis
    if ctx.Phase == PhasePre {
        ctx.Prompt = resp.ProcessedText
    } else {
        ctx.Response = resp.ProcessedText
    }
    ctx.Blocked = resp.Blocked
    ctx.BlockReason = resp.BlockReason
    ctx.Modified = resp.Modified

    // Merge State
    for k, v := range resp.State {
        ctx.State[k] = v
    }

    return nil
}
```

### 6.2 Beispiel: Leibniz als Handler-Plugin

Leibniz kann sich als Handler in Platon registrieren:

```go
// Leibniz registriert sich als Handler
func (s *Server) RegisterWithPlaton(ctx context.Context) error {
    _, err := s.platonClient.RegisterHandler(ctx, &platon.RegisterHandlerRequest{
        Name:        "leibniz-agent-router",
        Type:        "pre",
        Priority:    50,
        Endpoint:    "localhost:9140",
        Description: "Routet Anfragen an spezialisierte Agents",
    })
    return err
}
```

### 6.3 Handler-Typen Registry

```go
// Built-in Handler-Typen
var BuiltInHandlers = map[string]func(config map[string]string) Handler{
    "policy":     NewPolicyHandler,
    "audit":      NewAuditHandler,
    "transform":  NewTransformHandler,
    "routing":    NewRoutingHandler,
    "validation": NewValidationHandler,
    "rate-limit": NewRateLimitHandler,
    "cache":      NewCacheHandler,
}

// Dynamische Handler-Registrierung
type HandlerRegistry struct {
    handlers map[string]Handler
    plugins  map[string]*ExternalHandler
    mu       sync.RWMutex
}

func (r *HandlerRegistry) Get(name string) (Handler, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Erst built-in suchen
    if h, ok := r.handlers[name]; ok {
        return h, true
    }

    // Dann Plugins
    if h, ok := r.plugins[name]; ok {
        return h, true
    }

    return nil, false
}
```

---

## 7. Use Cases und Szenarien

### 7.1 Use Case: Content Moderation

```
Pipeline: "content-moderation"
Pre-Handlers:
  1. profanity-filter (Priority: 10)
  2. pii-detector (Priority: 20)
  3. topic-classifier (Priority: 30)

Post-Handlers:
  1. response-validator (Priority: 10)
  2. fact-checker (Priority: 20)
  3. tone-analyzer (Priority: 30)
```

### 7.2 Use Case: Enterprise Compliance

```
Pipeline: "enterprise-compliance"
Pre-Handlers:
  1. authentication-check (Priority: 5)
  2. authorization-check (Priority: 10)
  3. data-classification (Priority: 20)
  4. gdpr-compliance (Priority: 30)

Post-Handlers:
  1. response-classification (Priority: 10)
  2. data-masking (Priority: 20)
  3. audit-logging (Priority: 100)
```

### 7.3 Use Case: Multi-Agent Routing

```
Pipeline: "agent-router"
Pre-Handlers:
  1. intent-classifier (Priority: 10)
  2. agent-selector (Priority: 20)
  3. context-enrichment (Priority: 30)

Post-Handlers:
  1. response-aggregator (Priority: 10)
  2. quality-check (Priority: 20)
```

### 7.4 Use Case: RAG Enhancement

```
Pipeline: "rag-enhanced"
Pre-Handlers:
  1. query-expansion (Priority: 10)
  2. knowledge-retrieval (Priority: 20)
  3. context-injection (Priority: 30)

Post-Handlers:
  1. source-citation (Priority: 10)
  2. confidence-scoring (Priority: 20)
```

---

## 8. Migration von Leibniz nach Platon

### 8.1 Migrationsschritte

| Phase | Beschreibung | Aufwand |
|-------|--------------|---------|
| **Phase 1** | Platon Service Grundgeruest erstellen | 2 Tage |
| **Phase 2** | Handler Interface und Chain implementieren | 2 Tage |
| **Phase 3** | Built-in Handler migrieren (Policy, Audit) | 2 Tage |
| **Phase 4** | gRPC API implementieren | 1 Tag |
| **Phase 5** | Kant Integration | 1 Tag |
| **Phase 6** | Pipeline aus Leibniz entfernen | 1 Tag |
| **Phase 7** | Tests und Dokumentation | 2 Tage |
| **Phase 8** | REST API Handler in Kant migrieren | 1 Tag |

**Gesamtaufwand: ca. 12 Tage**

### 8.2 Detaillierter Migrationsplan

#### Phase 1: Platon Service Grundgeruest (2 Tage)

**Tag 1:**
- [ ] Verzeichnisstruktur erstellen (`internal/platon/`)
- [ ] Server-Grundgeruest implementieren
- [ ] gRPC Server Setup
- [ ] Health Check implementieren
- [ ] In `cmd/mdw/cmd/serve.go` integrieren

**Tag 2:**
- [ ] Konfiguration implementieren
- [ ] Service bei Russell registrieren
- [ ] Basis-Logging einrichten
- [ ] Proto-Datei erstellen (`api/proto/platon.proto`)

#### Phase 2: Handler Interface und Chain (2 Tage)

**Tag 3:**
- [ ] Handler Interface definieren
- [ ] ProcessingContext implementieren
- [ ] Chain-of-Responsibility Pattern implementieren

**Tag 4:**
- [ ] Handler Registry implementieren
- [ ] Plugin-Mechanismus fuer externe Handler
- [ ] Unit Tests fuer Chain

#### Phase 3: Built-in Handler migrieren (2 Tage)

**Tag 5:**
- [ ] PolicyHandler aus Leibniz extrahieren
- [ ] PolicyEngine nach Platon verschieben
- [ ] Policy Storage migrieren

**Tag 6:**
- [ ] AuditHandler implementieren
- [ ] ValidationHandler implementieren
- [ ] TransformHandler implementieren

#### Phase 4: gRPC API implementieren (1 Tag)

**Tag 7:**
- [ ] Proto generieren
- [ ] gRPC Handler implementieren
- [ ] ProcessPre/ProcessPost Methoden
- [ ] Pipeline CRUD Operationen

#### Phase 5: Kant Integration (1 Tag)

**Tag 8:**
- [ ] Platon Client in Kant erstellen
- [ ] Chat Handler anpassen
- [ ] Agent Handler anpassen
- [ ] Integration Tests

#### Phase 6: Pipeline aus Leibniz entfernen (1 Tag)

**Tag 9:**
- [ ] Pipeline-Code aus Leibniz entfernen
- [ ] Leibniz nutzt Platon Client
- [ ] Backward Compatibility pruefen
- [ ] Tests anpassen

#### Phase 7: Tests und Dokumentation (2 Tage)

**Tag 10:**
- [ ] Unit Tests vervollstaendigen
- [ ] Integration Tests schreiben
- [ ] E2E Tests anpassen

**Tag 11:**
- [ ] API Dokumentation
- [ ] Architektur-Dokumentation
- [ ] CLAUDE.md aktualisieren

#### Phase 8: REST API Handler migrieren (1 Tag)

**Tag 12:**
- [ ] Pipeline REST Endpoints von Leibniz nach Platon verschieben
- [ ] Kant Handler fuer Platon-Endpunkte anpassen
- [ ] Alte Leibniz-Endpunkte deprecaten

### 8.3 Backward Compatibility

Waehrend der Migration:

```go
// Kant kann beide APIs nutzen
type PipelineClient interface {
    ProcessPre(ctx context.Context, req *ProcessPreRequest) (*ProcessPreResponse, error)
    ProcessPost(ctx context.Context, req *ProcessPostRequest) (*ProcessPostResponse, error)
}

// Adapter fuer alte Leibniz API
type LeibnizPipelineAdapter struct {
    leibnizClient leibniz.LeibnizServiceClient
}

// Adapter fuer neue Platon API
type PlatonPipelineClient struct {
    platonClient platon.PlatonServiceClient
}

// Feature Flag fuer Migration
func (h *Handler) getPipelineClient() PipelineClient {
    if config.UsePlatonPipeline {
        return h.platonClient
    }
    return h.leibnizAdapter
}
```

---

## 9. Konfiguration

### 9.1 Platon Service Config

```toml
# configs/config.toml

[platon]
host = "0.0.0.0"
port = 9130
http_port = 9131

[platon.storage]
type = "sqlite"  # oder "postgres"
path = "./data/platon.db"

[platon.handlers]
# Built-in Handler aktivieren
policy = true
audit = true
validation = true
transform = true

[platon.handlers.policy]
default_action = "allow"  # "allow", "block", "redact"
strict_mode = false

[platon.handlers.audit]
retention_days = 30
async_logging = true

[platon.pipelines]
# Default Pipeline
default = ["policy", "audit"]
```

### 9.2 Environment Variables

```bash
# Platon Service
MDW_PLATON_HOST=0.0.0.0
MDW_PLATON_PORT=9130
MDW_PLATON_HTTP_PORT=9131
MDW_PLATON_STORAGE_TYPE=sqlite
MDW_PLATON_STORAGE_PATH=./data/platon.db
```

---

## 10. Monitoring und Observability

### 10.1 Metriken

```go
// Prometheus Metriken
var (
    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "platon_processing_duration_seconds",
            Help: "Duration of pipeline processing",
        },
        []string{"pipeline", "phase", "handler"},
    )

    processedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "platon_processed_total",
            Help: "Total number of processed requests",
        },
        []string{"pipeline", "result"}, // result: success, blocked, error
    )

    handlerErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "platon_handler_errors_total",
            Help: "Total number of handler errors",
        },
        []string{"handler"},
    )
)
```

### 10.2 Logging

```go
// Strukturiertes Logging
logger.Info("Pipeline processing completed",
    "request_id", ctx.RequestID,
    "pipeline", pipelineID,
    "duration_ms", duration.Milliseconds(),
    "handlers_executed", len(ctx.AuditLog),
    "blocked", ctx.Blocked,
    "modified", ctx.Modified,
)
```

### 10.3 Health Checks

```go
func (s *Server) healthCheck(ctx context.Context) health.CheckResult {
    // Pruefen ob Handler verfuegbar
    handlerCount := s.registry.Count()

    // Pruefen ob Storage erreichbar
    if err := s.store.Ping(ctx); err != nil {
        return health.CheckResult{
            Status:  health.StatusUnhealthy,
            Message: "Storage unavailable: " + err.Error(),
        }
    }

    return health.CheckResult{
        Status:  health.StatusHealthy,
        Message: fmt.Sprintf("Platon operational with %d handlers", handlerCount),
    }
}
```

---

## 11. Sicherheitsaspekte

### 11.1 Handler-Isolation

- Handler laufen mit eingeschraenkten Berechtigungen
- Timeouts fuer Handler-Ausfuehrung
- Circuit Breaker fuer externe Handler

### 11.2 Input Validation

```go
func validateProcessRequest(req *ProcessPreRequest) error {
    if req.RequestID == "" {
        return errors.New("request_id is required")
    }
    if len(req.Prompt) > MaxPromptLength {
        return errors.New("prompt exceeds maximum length")
    }
    return nil
}
```

### 11.3 Audit Trail

- Alle Verarbeitungsschritte werden protokolliert
- Unveraenderliche Audit-Eintraege
- Retention Policy konfigurierbar

---

## 12. Alternativen-Bewertung

### 12.1 Pipeline in Russell integrieren

| Pro | Contra |
|-----|--------|
| Kein neuer Service | Russell wird zu komplex |
| Orchestrierung passt thematisch | Vermischung von Routing und Processing |
| | Russell ist fuer Service Discovery gedacht |

**Bewertung:** Nicht empfohlen - Russell hat andere Verantwortlichkeiten

### 12.2 Pipeline in Turing integrieren

| Pro | Contra |
|-----|--------|
| Nah am LLM | Turing sollte nur LLM-spezifisch sein |
| | Nicht alle Pipelines brauchen LLM |
| | Enge Kopplung |

**Bewertung:** Nicht empfohlen - Turing ist fuer LLM-Management

### 12.3 Pipeline als Middleware in Kant

| Pro | Contra |
|-----|--------|
| Einfach zu implementieren | Kant wird zu komplex |
| Kein neuer Service | Schlechte Separation of Concerns |
| | Nicht wiederverwendbar |

**Bewertung:** Nur fuer einfache Faelle - nicht skalierbar

### 12.4 Dedizierter Platon Service (Empfohlen)

| Pro | Contra |
|-----|--------|
| Klare Verantwortlichkeiten | Ein weiterer Service |
| Wiederverwendbar | Mehr Netzwerk-Overhead |
| Unabhaengig skalierbar | |
| Plugin-System moeglich | |

**Bewertung:** Empfohlen - beste langfristige Loesung

---

## 13. Zusammenfassung

### 13.1 Entscheidung

**Empfehlung: Dedizierter Platon Service**

Die Pipeline-Funktionalitaet wird aus Leibniz extrahiert und in einen neuen Service "Platon" ausgelagert. Dies folgt dem Single Responsibility Principle und ermoeglicht maximale Flexibilitaet fuer zukuenftige Use Cases.

### 13.2 Naechste Schritte

1. **Review dieses Konzepts** durch alle Stakeholder
2. **Priorisierung** der Migration im Entwicklungsplan
3. **Start mit Phase 1** nach Freigabe
4. **Iterative Umsetzung** mit kontinuierlichem Testing

### 13.3 Erfolgsmetriken

| Metrik | Ziel |
|--------|------|
| Pipeline-Latenz | < 50ms fuer Standard-Handler |
| Test-Coverage | >= 80% |
| API-Kompatibilitaet | 100% waehrend Migration |
| Downtime | 0 waehrend Migration |

---

## Anhang

### A. Glossar

| Begriff | Beschreibung |
|---------|--------------|
| **Handler** | Einzelner Verarbeitungsschritt in der Pipeline |
| **Chain** | Verkettete Ausfuehrung von Handlern |
| **Pipeline** | Konfigurierte Sammlung von Handlern |
| **Pre-Processing** | Verarbeitung vor der Hauptaktion (LLM, Agent) |
| **Post-Processing** | Verarbeitung nach der Hauptaktion |
| **Plugin** | Externer Handler, der via gRPC angebunden ist |

### B. Referenzen

- [Chain of Responsibility Pattern](https://refactoring.guru/design-patterns/chain-of-responsibility)
- [Middleware Pattern in Go](https://go.dev/doc/articles/wiki/)
- [gRPC Best Practices](https://grpc.io/docs/guides/)

### C. Versionsverlauf

| Version | Datum | Autor | Aenderungen |
|---------|-------|-------|-------------|
| 1.0 | 2025-12-08 | Mike Stoffels | Initiale Version |
