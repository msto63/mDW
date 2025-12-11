# Agentic Pipeline: Intelligentes Prompt-Routing und -Processing

## Konzept: Automatische Prompt-Analyse und iterative Agent-Verarbeitung

**Version:** 2.0
**Datum:** 2025-12-11
**Autor:** Mike Stoffels mit Claude
**Status:** Konzept - Architekturentscheidung: Separater Service (Aristoteles)

---

## 1. Übersicht

### 1.1 Vision

Die Agentic Pipeline ist ein intelligentes System, das jeden eingehenden Prompt analysiert und automatisch entscheidet, welche Verarbeitungsstrategie optimal ist. Der Prompt durchläuft dabei eine konfigurierbare Pipeline, in der er von spezialisierten Agenten analysiert, angereichert und verfeinert wird, bevor das finale Ergebnis an Turing (LLM) übergeben wird.

### 1.2 Kernkonzept

```
User Prompt
     ↓
┌─────────────────────────────────────────────────────────────────┐
│                  ARISTOTELES SERVICE (NEU)                       │
│                     Port: 9160 (gRPC)                            │
│                                                                  │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │
│  │   Intent    │ → │   Agent     │ → │  Enrichment │ → ...     │
│  │  Analyzer   │   │  Selector   │   │    Stage    │           │
│  └─────────────┘   └─────────────┘   └─────────────┘           │
│         ↓                ↓                 ↓                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Pipeline Context (State)                    │   │
│  │  • Intent: web_search | task_decomposition | direct_llm │   │
│  │  • Enrichments: Fakten, Kontext, Recherche-Ergebnisse   │   │
│  │  • Routing: Ziel-Agent(en), Pipeline-Konfiguration      │   │
│  │  • Iteration: Schleifenkontrolle, Qualitätsschwellen    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────────────────────────────┐
│                    EXECUTION LAYER                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Direct    │  │    Agent    │  │   Multi-    │             │
│  │    LLM      │  │  Execution  │  │   Agent     │             │
│  │  (Turing)   │  │  (Leibniz)  │  │   Orch.     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────────────────────────────┐
│  PLATON (Policy Enforcement) - Optional für Post-Processing     │
└─────────────────────────────────────────────────────────────────┘
     ↓
Final Response
```

### 1.3 Abgrenzung der Services

| Service | Verantwortlichkeit | Fokus |
|---------|-------------------|-------|
| **Aristoteles** (NEU) | Intelligentes Routing & Orchestrierung | WAS soll passieren |
| **Platon** | Policy-Enforcement (PII, Safety) | WIE es sicher passiert |
| **Leibniz** | Agent-Ausführung | Agenten führen Tasks aus |
| **Turing** | LLM-Kommunikation | Modell-Inferenz |

### 1.4 Warum ein separater Service?

**Architekturprinzipien:**

1. **Single Responsibility**: Platon = Policies, Aristoteles = Routing/Orchestrierung
2. **Iterative Schleifen**: Verfeinerungsloops passen nicht in lineare Handler-Chains
3. **LLM-Abhängigkeit**: Intent-Analyse braucht selbst LLM-Aufrufe - gehört nicht in Policy-Service
4. **Zukunftssicherheit**: Multi-Agent-Orchestrierung, parallele Ausführung, A/B-Testing

**Vergleich der Optionen:**

| Kriterium | Platon-Erweiterung | Separater Service |
|-----------|-------------------|-------------------|
| Komplexität initial | Niedrig | Mittel |
| Wartbarkeit langfristig | Schwierig | Einfach |
| Iterative Loops | Schwierig | Native |
| Parallele Agents | Nicht möglich | Native |
| Unabhängige Skalierung | Nein | Ja |
| Klare Verantwortlichkeit | Vermischt | Klar getrennt |

---

## 2. Aristoteles Service - Architektur

### 2.1 Service-Überblick

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARISTOTELES SERVICE                           │
│                     Port: 9160 (gRPC)                            │
│                     Port: 9161 (HTTP/REST)                       │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   PIPELINE ENGINE                        │    │
│  │                                                          │    │
│  │  ┌────────────┐                                         │    │
│  │  │  Intent    │ → Schnelle LLM-basierte Klassifikation  │    │
│  │  │  Analyzer  │   (llama3.2:3b für minimale Latenz)     │    │
│  │  └────────────┘                                         │    │
│  │        ↓                                                │    │
│  │  ┌────────────┐                                         │    │
│  │  │  Strategy  │ → Wählt Verarbeitungsstrategie          │    │
│  │  │  Selector  │   (direct, agent, multi-agent)          │    │
│  │  └────────────┘                                         │    │
│  │        ↓                                                │    │
│  │  ┌────────────┐                                         │    │
│  │  │ Enrichment │ → Web-Recherche, RAG, Fakten-Check      │    │
│  │  │   Loop     │   (iterativ bis Qualität erreicht)      │    │
│  │  └────────────┘                                         │    │
│  │        ↓                                                │    │
│  │  ┌────────────┐                                         │    │
│  │  │  Quality   │ → Evaluiert Anreicherungs-Qualität      │    │
│  │  │ Evaluator  │   (Entscheidet: weiter oder fertig)     │    │
│  │  └────────────┘                                         │    │
│  │        ↓                                                │    │
│  │  ┌────────────┐                                         │    │
│  │  │  Router    │ → Leitet an Turing/Leibniz/Multi-Agent  │    │
│  │  │            │   mit angereichertem Prompt             │    │
│  │  └────────────┘                                         │    │
│  │                                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   SERVICE CLIENTS                        │    │
│  │  • Turing (LLM) - für Intent-Analyse & finale Antwort   │    │
│  │  • Leibniz (Agents) - für spezialisierte Agenten        │    │
│  │  • Platon (Policy) - für PII/Safety-Checks              │    │
│  │  • Hypatia (RAG) - für Wissensabruf                     │    │
│  │  • Babbage (NLP) - für Textanalyse                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Verzeichnisstruktur

```
internal/aristoteles/
├── server/
│   └── server.go              # gRPC Server
├── service/
│   └── service.go             # Business Logic
├── pipeline/
│   ├── engine.go              # Pipeline-Engine
│   ├── context.go             # Pipeline-Context mit State
│   └── stages.go              # Stage-Definitionen
├── intent/
│   ├── analyzer.go            # Intent-Analyse via LLM
│   ├── classifier.go          # Intent-Klassifikation
│   └── prompts.go             # LLM-Prompts für Analyse
├── strategy/
│   ├── selector.go            # Strategie-Auswahl
│   └── types.go               # Strategie-Typen
├── enrichment/
│   ├── enricher.go            # Anreicherungs-Koordinator
│   ├── web.go                 # Web-Recherche
│   ├── rag.go                 # RAG-Integration
│   └── facts.go               # Fakten-Extraktion
├── quality/
│   ├── evaluator.go           # Qualitäts-Bewertung
│   └── thresholds.go          # Schwellenwerte
├── router/
│   └── router.go              # Service-Routing
└── clients/
    ├── turing.go              # Turing-Client
    ├── leibniz.go             # Leibniz-Client
    ├── platon.go              # Platon-Client
    └── hypatia.go             # Hypatia-Client
```

### 2.3 gRPC API

```protobuf
// api/proto/aristoteles.proto

syntax = "proto3";
package aristoteles;
option go_package = "github.com/msto63/mDW/api/gen/aristoteles";

service AristotelesService {
    // Hauptmethode: Verarbeitet einen Prompt durch die Pipeline
    rpc Process(ProcessRequest) returns (ProcessResponse);

    // Streaming-Version für Echtzeit-Feedback
    rpc ProcessStream(ProcessRequest) returns (stream ProcessEvent);

    // Nur Intent-Analyse (ohne Ausführung)
    rpc AnalyzeIntent(IntentRequest) returns (IntentResponse);

    // Pipeline-Status und Metriken
    rpc GetPipelineStatus(StatusRequest) returns (StatusResponse);

    // Health Check
    rpc Health(HealthRequest) returns (HealthResponse);
}

message ProcessRequest {
    string request_id = 1;
    string prompt = 2;
    string conversation_id = 3;      // Für Kontext-Tracking
    map<string, string> metadata = 4;
    ProcessOptions options = 5;
}

message ProcessOptions {
    bool skip_intent_analysis = 1;   // Direkter LLM-Aufruf
    string force_intent = 2;         // Intent überschreiben
    repeated string force_agents = 3; // Agenten erzwingen
    int32 max_iterations = 4;        // Max Verfeinerungs-Iterationen
    float quality_threshold = 5;     // Mindest-Qualität (0.0-1.0)
    bool enable_web_search = 6;      // Web-Recherche erlauben
    bool enable_rag = 7;             // RAG-Suche erlauben
    string preferred_model = 8;      // Bevorzugtes LLM-Modell
}

message ProcessResponse {
    string request_id = 1;
    string response = 2;             // Finale Antwort
    IntentAnalysis intent = 3;       // Intent-Analyse-Ergebnis
    RoutingDecision routing = 4;     // Routing-Entscheidung
    repeated Enrichment enrichments = 5; // Anreicherungen
    PipelineMetrics metrics = 6;     // Performance-Metriken
    repeated PipelineStep steps = 7; // Alle Pipeline-Schritte
}

message ProcessEvent {
    string event_type = 1;           // "intent", "enrichment", "routing", "response"
    string stage = 2;                // Aktuelle Pipeline-Stage
    string message = 3;              // Status-Nachricht
    map<string, string> data = 4;    // Event-spezifische Daten
    float progress = 5;              // Fortschritt (0.0-1.0)
}

message IntentAnalysis {
    string intent = 1;               // direct_llm, web_research, etc.
    float confidence = 2;            // Konfidenz (0.0-1.0)
    string reasoning = 3;            // Begründung
    repeated string suggested_agents = 4;
    bool needs_enrichment = 5;
    string enrichment_type = 6;      // web_search, knowledge_base, none
}

message RoutingDecision {
    string target_service = 1;       // turing, leibniz, multi_agent
    repeated string agents = 2;      // Gewählte Agenten
    string model = 3;                // Gewähltes LLM-Modell
    string enriched_prompt = 4;      // Angereicherter Prompt
}

message Enrichment {
    string source = 1;               // web_search, rag, facts
    string content = 2;              // Angereicherte Daten
    map<string, string> metadata = 3;
    float relevance_score = 4;       // Relevanz (0.0-1.0)
}

message PipelineStep {
    string stage = 1;
    string action = 2;
    int64 duration_ms = 3;
    bool success = 4;
    string error = 5;
    map<string, string> details = 6;
}

message PipelineMetrics {
    int64 total_duration_ms = 1;
    int32 iterations = 2;
    float final_quality_score = 3;
    int32 llm_calls = 4;
    int32 agent_calls = 5;
}
```

---

## 3. Pipeline-Engine

### 3.1 Pipeline-Stages

```go
// internal/aristoteles/pipeline/engine.go

type PipelineEngine struct {
    stages     []Stage
    clients    *Clients
    config     Config
    logger     *logging.Logger
    metrics    *metrics.Collector
}

type Stage interface {
    Name() string
    Process(ctx *PipelineContext) error
    ShouldRun(ctx *PipelineContext) bool
}

// Pipeline-Stages in Reihenfolge
var DefaultStages = []Stage{
    &IntentAnalyzerStage{},      // 1. Intent erkennen
    &StrategySelectorStage{},    // 2. Strategie wählen
    &EnrichmentStage{},          // 3. Anreichern (iterativ)
    &QualityEvaluatorStage{},    // 4. Qualität prüfen
    &PolicyCheckStage{},         // 5. Policies (via Platon)
    &RouterStage{},              // 6. Routing & Ausführung
}

func (e *PipelineEngine) Process(ctx context.Context, req *ProcessRequest) (*ProcessResponse, error) {
    pctx := NewPipelineContext(ctx, req)

    for _, stage := range e.stages {
        if !stage.ShouldRun(pctx) {
            continue
        }

        start := time.Now()
        err := stage.Process(pctx)
        duration := time.Since(start)

        pctx.AddStep(PipelineStep{
            Stage:      stage.Name(),
            DurationMs: duration.Milliseconds(),
            Success:    err == nil,
            Error:      errorString(err),
        })

        if err != nil {
            return nil, fmt.Errorf("stage %s failed: %w", stage.Name(), err)
        }

        // Prüfe ob Pipeline früh beendet werden soll
        if pctx.ShouldTerminate() {
            break
        }
    }

    return pctx.ToResponse(), nil
}
```

### 3.2 Pipeline-Context

```go
// internal/aristoteles/pipeline/context.go

type PipelineContext struct {
    ctx           context.Context
    requestID     string
    originalPrompt string
    currentPrompt  string

    // Intent-Analyse
    Intent          IntentAnalysis

    // Routing
    TargetService   string
    TargetAgents    []string
    TargetModel     string

    // Anreicherungen
    Enrichments     []Enrichment

    // Iteration Control
    Iteration       int
    MaxIterations   int
    QualityScore    float64
    QualityThreshold float64

    // Ausführungsergebnis
    Response        string

    // Audit
    Steps           []PipelineStep

    // Flags
    terminate       bool
    skipEnrichment  bool
}

func (c *PipelineContext) ShouldIterate() bool {
    return c.Iteration < c.MaxIterations &&
           c.QualityScore < c.QualityThreshold
}

func (c *PipelineContext) EnrichPrompt(enrichment Enrichment) {
    c.Enrichments = append(c.Enrichments, enrichment)
    c.currentPrompt = c.buildEnrichedPrompt()
}

func (c *PipelineContext) buildEnrichedPrompt() string {
    if len(c.Enrichments) == 0 {
        return c.originalPrompt
    }

    var sb strings.Builder
    sb.WriteString("KONTEXT (aus Recherche):\n")
    for _, e := range c.Enrichments {
        sb.WriteString(fmt.Sprintf("- [%s]: %s\n", e.Source, e.Content))
    }
    sb.WriteString("\nURSPRÜNGLICHE ANFRAGE:\n")
    sb.WriteString(c.originalPrompt)

    return sb.String()
}
```

### 3.3 Iterative Verfeinerungsschleife

```go
// internal/aristoteles/pipeline/stages.go

type EnrichmentStage struct {
    webSearcher  *enrichment.WebSearcher
    ragClient    *clients.HypatiaClient
    factChecker  *enrichment.FactChecker
}

func (s *EnrichmentStage) Process(ctx *PipelineContext) error {
    if !ctx.Intent.NeedsEnrichment {
        return nil
    }

    // Iterative Anreicherung
    for ctx.ShouldIterate() {
        ctx.Iteration++

        var enrichment Enrichment
        var err error

        switch ctx.Intent.EnrichmentType {
        case "web_search":
            enrichment, err = s.webSearcher.Search(ctx.ctx, ctx.currentPrompt)
        case "knowledge_base":
            enrichment, err = s.ragClient.Search(ctx.ctx, ctx.currentPrompt)
        case "fact_check":
            enrichment, err = s.factChecker.Check(ctx.ctx, ctx.currentPrompt)
        }

        if err != nil {
            return err
        }

        ctx.EnrichPrompt(enrichment)

        // Qualität evaluieren
        quality, err := s.evaluateQuality(ctx)
        if err != nil {
            return err
        }
        ctx.QualityScore = quality

        // Prüfe ob Qualität erreicht
        if quality >= ctx.QualityThreshold {
            break
        }
    }

    return nil
}
```

---

## 4. Intent-Analyse

### 4.1 Intent-Kategorien

| Intent | Beschreibung | Routing | Modell |
|--------|--------------|---------|--------|
| `direct_llm` | Einfache Fragen, Erklärungen | Direkt zu Turing | Default |
| `web_research` | Aktuelle Informationen | Web-Researcher Agent | llama3.2:8b |
| `code_generation` | Code schreiben | Code-Writer Agent | qwen2.5-coder:7b |
| `code_analysis` | Code-Review, Debugging | Code-Reviewer Agent | qwen2.5-coder:7b |
| `task_decomposition` | Komplexe Aufgabe zerlegen | Task-Planner Agent | deepseek-r1:7b |
| `knowledge_retrieval` | Aus Wissensdatenbank | Hypatia (RAG) | Default |
| `multi_agent` | Mehrere Agenten koordiniert | Multi-Agent Orch. | Variabel |

### 4.2 LLM-basierte Intent-Erkennung

```go
// internal/aristoteles/intent/analyzer.go

type IntentAnalyzer struct {
    turingClient *clients.TuringClient
    model        string // llama3.2:3b für schnelle Klassifikation
    logger       *logging.Logger
}

const intentPrompt = `Du bist ein Intent-Klassifikator für ein KI-System.
Analysiere die folgende Benutzeranfrage und bestimme die beste Verarbeitungsstrategie.

VERFÜGBARE STRATEGIEN:
1. direct_llm: Allgemeine Fragen, Erklärungen, Konversation ohne externe Daten
2. web_research: Benötigt aktuelle Informationen (News, Preise, Wetter, Ereignisse)
3. code_generation: Neuen Code schreiben, Implementierung erstellen
4. code_analysis: Bestehenden Code analysieren, debuggen, reviewen
5. task_decomposition: Komplexe Aufgabe in Teilschritte zerlegen
6. knowledge_retrieval: Informationen aus Wissensdatenbank abrufen
7. multi_agent: Benötigt mehrere spezialisierte Agenten

BENUTZERANFRAGE:
"""
{{.Prompt}}
"""

Antworte NUR im folgenden JSON-Format:
{
  "intent": "<strategy>",
  "confidence": <0.0-1.0>,
  "reasoning": "<kurze Begründung auf Deutsch>",
  "suggested_agents": ["<agent_id>", ...],
  "needs_enrichment": <true/false>,
  "enrichment_type": "<web_search|knowledge_base|fact_check|none>"
}`

func (a *IntentAnalyzer) Analyze(ctx context.Context, prompt string) (*IntentAnalysis, error) {
    // Prompt für Intent-Analyse erstellen
    analysisPrompt := strings.Replace(intentPrompt, "{{.Prompt}}", prompt, 1)

    // Schnelle LLM-Abfrage mit kleinem Modell
    response, err := a.turingClient.Chat(ctx, a.model, []Message{
        {Role: "user", Content: analysisPrompt},
    })
    if err != nil {
        return nil, fmt.Errorf("intent analysis failed: %w", err)
    }

    // JSON parsen
    var analysis IntentAnalysis
    if err := json.Unmarshal([]byte(response), &analysis); err != nil {
        // Fallback zu direct_llm bei Parse-Fehlern
        a.logger.Warn("Failed to parse intent analysis, falling back to direct_llm",
            "error", err, "response", response)
        return &IntentAnalysis{
            Intent:     "direct_llm",
            Confidence: 0.5,
            Reasoning:  "Fallback wegen Parse-Fehler",
        }, nil
    }

    return &analysis, nil
}
```

### 4.3 Beispiel-Klassifikationen

```
Prompt: "Erkläre mir Rekursion in Python"
→ {
    "intent": "direct_llm",
    "confidence": 0.95,
    "reasoning": "Allgemeine Programmiererklärung ohne externe Daten",
    "suggested_agents": [],
    "needs_enrichment": false,
    "enrichment_type": "none"
  }

Prompt: "Was sind die aktuellen Nachrichten zu KI-Regulierung in der EU?"
→ {
    "intent": "web_research",
    "confidence": 0.92,
    "reasoning": "Fragt nach aktuellen Informationen (Nachrichten, EU)",
    "suggested_agents": ["web-researcher"],
    "needs_enrichment": true,
    "enrichment_type": "web_search"
  }

Prompt: "Schreibe eine Go-Funktion für Fibonacci mit Memoization"
→ {
    "intent": "code_generation",
    "confidence": 0.94,
    "reasoning": "Explizite Code-Erstellung angefordert",
    "suggested_agents": ["code-writer"],
    "needs_enrichment": false,
    "enrichment_type": "none"
  }

Prompt: "Erstelle eine REST-API mit Auth, Datenbank und Tests"
→ {
    "intent": "task_decomposition",
    "confidence": 0.88,
    "reasoning": "Komplexe Aufgabe mit mehreren Komponenten",
    "suggested_agents": ["task-planner"],
    "needs_enrichment": false,
    "enrichment_type": "none"
  }

Prompt: "Recherchiere Go-Frameworks 2025 und erstelle einen Vergleichsbericht"
→ {
    "intent": "multi_agent",
    "confidence": 0.85,
    "reasoning": "Benötigt Web-Recherche UND strukturierte Aufbereitung",
    "suggested_agents": ["web-researcher", "task-planner"],
    "needs_enrichment": true,
    "enrichment_type": "web_search"
  }
```

---

## 5. Integration in mDW

### 5.1 Gesamtarchitektur mit Aristoteles

```
┌─────────────────────────────────────────────────────────────────┐
│                      mDW ARCHITEKTUR                             │
│                                                                  │
│  ┌──────────┐                                                   │
│  │  Client  │ (TUI, Web, CLI)                                   │
│  └────┬─────┘                                                   │
│       ↓                                                         │
│  ┌──────────┐                                                   │
│  │   Kant   │ API Gateway (:8080)                               │
│  │          │ → Routet zu Aristoteles (wenn aktiviert)          │
│  └────┬─────┘                                                   │
│       ↓                                                         │
│  ┌────────────────┐                                             │
│  │  ARISTOTELES   │ Agentic Pipeline (:9160)                    │
│  │                │ → Intent-Analyse                            │
│  │                │ → Strategie-Auswahl                         │
│  │                │ → Anreicherung                              │
│  │                │ → Routing                                   │
│  └───────┬────────┘                                             │
│          │                                                      │
│    ┌─────┼─────────────────────────────┐                       │
│    ↓     ↓                             ↓                        │
│  ┌─────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐                   │
│  │Turing│ │ Leibniz │ │ Hypatia │ │ Platon  │                   │
│  │(LLM) │ │(Agents) │ │  (RAG)  │ │(Policy) │                   │
│  └─────┘ └─────────┘ └─────────┘ └─────────┘                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Request-Flow

```
1. Client sendet Anfrage an Kant
   POST /api/v1/chat { "message": "..." }

2. Kant prüft: Aristoteles aktiviert?
   JA → Weiterleitung an Aristoteles
   NEIN → Direkt an Turing

3. Aristoteles verarbeitet:
   a) Intent-Analyse (llama3.2:3b, ~200ms)
   b) Strategie-Auswahl
   c) Ggf. Anreicherung (Web-Recherche, RAG)
   d) Routing-Entscheidung

4. Aristoteles routet:
   - direct_llm → Turing.Chat()
   - code_* → Leibniz.Execute(code-writer/reviewer)
   - web_research → Leibniz.Execute(web-researcher)
   - multi_agent → Multi-Agent-Orchestrierung

5. Aristoteles sammelt Response
   - Ggf. Post-Processing via Platon (PII-Filter)
   - Metrics und Audit-Log

6. Response zurück an Kant → Client
```

### 5.3 Kant-Integration

```go
// internal/kant/handler/handler.go

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
    // ... Request parsen ...

    // Aristoteles-Integration
    if h.config.EnableAristotle {
        resp, err := h.clients.Aristoteles.Process(ctx, &aristotelespb.ProcessRequest{
            RequestId: requestID,
            Prompt:    userMessage,
            Options: &aristotelespb.ProcessOptions{
                EnableWebSearch: h.config.EnableWebSearch,
                EnableRag:       h.config.EnableRAG,
                MaxIterations:   h.config.MaxIterations,
                QualityThreshold: h.config.QualityThreshold,
            },
        })

        if err != nil {
            // Fallback zu direktem Turing-Aufruf
            h.logger.Warn("Aristoteles failed, falling back to Turing", "error", err)
        } else {
            // Aristoteles hat verarbeitet
            return h.sendResponse(w, resp.Response, resp.Metrics)
        }
    }

    // Fallback: Direkter Turing-Aufruf
    turingResp, err := h.clients.Turing.Chat(ctx, &turingpb.ChatRequest{
        Model:    h.config.DefaultModel,
        Messages: messages,
    })
    // ...
}
```

---

## 6. Konfiguration

### 6.1 config.toml

```toml
# Aristoteles - Agentic Pipeline Service
[aristoteles]
host = "0.0.0.0"
port = 9160
http_port = 9161

# Pipeline-Einstellungen
[aristoteles.pipeline]
max_iterations = 3
quality_threshold = 0.8
enable_web_search = true
enable_rag = true
default_timeout = "30s"

# Intent-Analyse
[aristoteles.intent]
model = "llama3.2:3b"        # Schnelles Modell für Klassifikation
timeout = "3s"
confidence_threshold = 0.7   # Mindest-Konfidenz

# Strategie-Modelle (welches LLM für welchen Intent)
[aristoteles.models]
direct_llm = "llama3.2:8b"
code_generation = "qwen2.5-coder:7b"
code_analysis = "qwen2.5-coder:7b"
task_decomposition = "deepseek-r1:7b"
web_research = "llama3.2:8b"

# Service-Verbindungen
[aristoteles.services]
turing_addr = "localhost:9200"
leibniz_addr = "localhost:9140"
platon_addr = "localhost:9130"
hypatia_addr = "localhost:9220"
babbage_addr = "localhost:9150"

# Kant - API Gateway
[kant]
port = 8080
enable_aristotle = true     # Aristoteles aktivieren
fallback_on_error = true    # Bei Fehler zu Turing fallback
```

### 6.2 Port-Konvention

| Service | gRPC Port | HTTP Port | Beschreibung |
|---------|-----------|-----------|--------------|
| Kant | - | 8080 | API Gateway |
| Russell | 9100 | 9101 | Service Discovery |
| Bayes | 9120 | 9121 | Logging |
| Platon | 9130 | 9131 | Pipeline/Policy |
| Leibniz | 9140 | 9141 | Agentic AI |
| Babbage | 9150 | 9151 | NLP |
| **Aristoteles** | **9160** | **9161** | **Agentic Pipeline** |
| Turing | 9200 | 9201 | LLM |
| Hypatia | 9220 | 9221 | RAG |

---

## 7. UI-Integration (ChatClient TUI)

### 7.1 Status-Anzeige

```
┌─────────────────────────────────────────────────────────────────┐
│  mDW Chat                                          llama3.2:8b  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User: Was sind die aktuellen Nachrichten zu KI?               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ [Aristoteles] Verarbeite Anfrage...                      │   │
│  │                                                          │   │
│  │  1. Intent: web_research (92%)                          │   │
│  │  2. Agent: web-researcher                               │   │
│  │  3. Recherche läuft... ████████░░ 80%                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Assistant: Basierend auf meiner aktuellen Recherche...        │
│                                                                 │
│  Quellen:                                                       │
│  - heise.de: "EU AI Act tritt in Kraft..."                     │
│  - golem.de: "Neue Regulierungen für KI..."                    │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  > _                                          [Aristoteles: ON] │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Routing-Indikator

| Symbol | Bedeutung |
|--------|-----------|
| 💬 | Direct LLM (Standard-Chat) |
| 🔍 | Web-Recherche aktiv |
| 💻 | Code-Generierung |
| 🔧 | Code-Analyse |
| 📋 | Aufgabenzerlegung |
| 📚 | Wissensabruf (RAG) |
| 🤖 | Multi-Agent |

---

## 8. Multi-LLM-Strategie

### 8.1 Modell-Zuweisung

Die agent-spezifische Modellauswahl ist bereits in Leibniz implementiert (`ModelAwareLLMFunc`). Aristoteles nutzt diese Funktionalität:

```go
// Aristoteles setzt das Modell basierend auf Intent
func (r *Router) selectModel(intent string) string {
    switch intent {
    case "code_generation", "code_analysis":
        return "qwen2.5-coder:7b"
    case "task_decomposition":
        return "deepseek-r1:7b"
    case "web_research":
        return "llama3.2:8b"
    default:
        return "llama3.2:8b"
    }
}

// Weiterleitung an Leibniz mit Modell
func (r *Router) routeToLeibniz(ctx *PipelineContext) error {
    model := r.selectModel(ctx.Intent.Intent)

    resp, err := r.leibnizClient.ExecuteWithAgent(ctx.ctx, &leibnizpb.ExecuteRequest{
        AgentId: ctx.TargetAgents[0],
        Message: ctx.currentPrompt,
        Model:   model,  // Agent nutzt dieses Modell
    })
    // ...
}
```

### 8.2 VRAM-Management

Aristoteles berücksichtigt VRAM-Limits bei der Modell-Auswahl:

```go
// Prüfe ob Modell geladen werden kann
func (r *Router) canUseModel(model string) bool {
    status, _ := r.turingClient.GetModelStatus(ctx)

    // Wenn Modell bereits geladen → OK
    if status.LoadedModels[model] {
        return true
    }

    // Prüfe ob genug VRAM frei
    modelSize := r.getModelSize(model)
    return status.FreeVRAM >= modelSize
}
```

---

## 9. Implementierungs-Roadmap

### Phase 1: Foundation (2 Wochen)

- [ ] Aristoteles Service-Grundstruktur
  - [ ] gRPC Server (`internal/aristoteles/server/`)
  - [ ] Service-Grundgerüst (`internal/aristoteles/service/`)
  - [ ] Proto-Definitionen (`api/proto/aristoteles.proto`)
  - [ ] Health-Check und Russell-Registrierung

- [ ] Pipeline-Engine Basis
  - [ ] PipelineContext und State-Management
  - [ ] Stage-Interface und Ausführung
  - [ ] Basis-Metriken

- [ ] Intent-Analyzer (MVP)
  - [ ] LLM-basierte Klassifikation
  - [ ] Basis-Intents: direct_llm, web_research, code_generation
  - [ ] Turing-Client für Intent-Analyse

### Phase 2: Routing & Integration (2 Wochen)

- [ ] Strategy-Selector
  - [ ] Intent-zu-Strategie-Mapping
  - [ ] Modell-Auswahl pro Intent

- [ ] Router-Implementierung
  - [ ] Turing-Client (Direct LLM)
  - [ ] Leibniz-Client (Agents)
  - [ ] Platon-Client (Policy-Checks)

- [ ] Kant-Integration
  - [ ] Aristoteles-Client in Kant
  - [ ] Konfiguration enable_aristotle
  - [ ] Fallback-Logik

### Phase 3: Enrichment & Iteration (2 Wochen)

- [ ] Enrichment-Stage
  - [ ] Web-Recherche Integration
  - [ ] RAG via Hypatia
  - [ ] Prompt-Anreicherung

- [ ] Iterative Verfeinerung
  - [ ] Quality-Evaluator
  - [ ] Schleifenlogik mit Abbruchbedingungen
  - [ ] Max-Iterations-Limit

- [ ] Multi-Agent Orchestrierung
  - [ ] Parallele Agent-Ausführung
  - [ ] Ergebnis-Aggregation

### Phase 4: UI & Optimierung (1 Woche)

- [ ] ChatClient TUI Integration
  - [ ] Pipeline-Status-Anzeige
  - [ ] Routing-Indikatoren
  - [ ] Streaming-Events

- [ ] Performance-Optimierung
  - [ ] Intent-Caching
  - [ ] Connection-Pooling
  - [ ] Metriken und Monitoring

---

## 10. Zusammenfassung

### Architekturentscheidung

**Aristoteles** wird als dedizierter Service für die Agentic Pipeline implementiert:

- **Klare Verantwortlichkeit**: Routing & Orchestrierung getrennt von Policies
- **Native Iteration**: Verfeinerungsschleifen ohne Workarounds
- **Zukunftssicher**: Multi-Agent, parallele Ausführung, A/B-Testing möglich
- **Unabhängig skalierbar**: Kann bei Bedarf horizontal skaliert werden

### Kernfunktionen

1. **Intent-Analyse**: LLM-basierte Klassifikation mit llama3.2:3b (~200ms)
2. **Strategie-Auswahl**: Automatische Modell- und Agent-Auswahl
3. **Iterative Anreicherung**: Web-Recherche, RAG, Fakten-Check
4. **Intelligentes Routing**: Zu Turing, Leibniz, oder Multi-Agent

### Nächste Schritte

1. Proto-Definition erstellen
2. Service-Grundstruktur aufsetzen
3. Intent-Analyzer implementieren
4. Kant-Integration
