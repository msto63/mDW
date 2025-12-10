# Agentic Pipeline: Intelligentes Prompt-Routing und -Processing

## Konzept: Automatische Prompt-Analyse und iterative Agent-Verarbeitung

**Version:** 1.0
**Datum:** 2025-12-10
**Autor:** Mike Stoffels mit Claude
**Status:** Konzept

---

## 1. Übersicht

### 1.1 Vision

Die Agentic Pipeline ist ein intelligentes System, das jeden eingehenden Prompt analysiert und automatisch entscheidet, welche Verarbeitungsstrategie optimal ist. Der Prompt durchläuft dabei eine konfigurierbare Pipeline, in der er von spezialisierten Agenten analysiert, angereichert und verfeinert wird, bevor das finale Ergebnis an Turing (LLM) übergeben wird.

### 1.2 Kernkonzept

```
User Prompt
     ↓
┌─────────────────────────────────────────────────────────────────┐
│                     AGENTIC PIPELINE                            │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │   Intent    │ → │   Agent     │ → │  Enrichment │ → ...    │
│  │  Analyzer   │   │  Selector   │   │    Stage    │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│         ↓                ↓                 ↓                   │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              Processing Context (State)                  │  │
│  │  • Intent: web_search | task_decomposition | direct_llm │  │
│  │  • Enrichments: Fakten, Kontext, Recherche-Ergebnisse   │  │
│  │  • Routing: Ziel-Agent(en), Pipeline-Konfiguration      │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────────────────────────────┐
│                    EXECUTION LAYER                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Direct    │  │    Agent    │  │   Multi-    │            │
│  │    LLM      │  │  Execution  │  │   Agent     │            │
│  │  (Turing)   │  │  (Leibniz)  │  │   Orch.     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
     ↓
Final Response
```

### 1.3 Abgrenzung zu Multi-Agent Orchestration

| Aspekt | Agentic Pipeline | Multi-Agent Orchestration |
|--------|------------------|---------------------------|
| **Fokus** | Prompt-Routing & Pre-Processing | Task-Ausführung & Koordination |
| **Wann** | VOR der eigentlichen Verarbeitung | WÄHREND der Verarbeitung |
| **Ziel** | Beste Strategie wählen | Komplexe Aufgaben lösen |
| **Output** | Routing-Entscheidung + angereicherter Prompt | Fertiges Ergebnis |

Die Agentic Pipeline ist der **Eintrittspunkt**, der entscheidet, OB Multi-Agent Orchestration überhaupt nötig ist.

---

## 2. Architektur-Optionen

### 2.1 Option A: Platon-Erweiterung (Empfohlen)

Platon bietet bereits eine Handler-Chain mit Pre/Post-Processing. Die Agentic Pipeline kann als **spezialisierte Handler** in diese bestehende Infrastruktur integriert werden.

```
┌─────────────────────────────────────────────────────────────────┐
│                      PLATON SERVICE                             │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   PRE-PROCESSING CHAIN                   │   │
│  │                                                          │   │
│  │  Priority 10:  ┌──────────────────────────────────────┐ │   │
│  │                │  IntentAnalyzerHandler               │ │   │
│  │                │  - LLM-basierte Intent-Erkennung     │ │   │
│  │                │  - Setzt ctx.State["intent"]         │ │   │
│  │                └──────────────────────────────────────┘ │   │
│  │                              ↓                          │   │
│  │  Priority 20:  ┌──────────────────────────────────────┐ │   │
│  │                │  AgentSelectorHandler                │ │   │
│  │                │  - Wählt passende Agenten            │ │   │
│  │                │  - Setzt ctx.State["target_agents"]  │ │   │
│  │                └──────────────────────────────────────┘ │   │
│  │                              ↓                          │   │
│  │  Priority 30:  ┌──────────────────────────────────────┐ │   │
│  │                │  EnrichmentHandler                   │ │   │
│  │                │  - Web-Recherche bei Bedarf          │ │   │
│  │                │  - Kontext-Anreicherung              │ │   │
│  │                │  - Modifiziert ctx.Prompt            │ │   │
│  │                └──────────────────────────────────────┘ │   │
│  │                              ↓                          │   │
│  │  Priority 100: ┌──────────────────────────────────────┐ │   │
│  │                │  PolicyHandler (PII, Safety)         │ │   │
│  │                └──────────────────────────────────────┘ │   │
│  │                              ↓                          │   │
│  │  Priority 1000:┌──────────────────────────────────────┐ │   │
│  │                │  AuditHandler                        │ │   │
│  │                └──────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              ↓                                  │
│                    ProcessingContext mit                        │
│                    Intent + Agents + Enrichments                │
└─────────────────────────────────────────────────────────────────┘
```

**Vorteile:**
- Nutzt bestehende Infrastruktur (Chain-of-Responsibility)
- ProcessingContext für State-Sharing bereits vorhanden
- Audit-Trail automatisch integriert
- Kein neuer Service nötig
- Platon-Client in Leibniz bereits implementiert

**Nachteile:**
- Platon wird komplexer
- Iterative Agent-Schleifen nicht direkt unterstützt

### 2.2 Option B: Separater Orchestrator-Service

Ein neuer Service **Aristoteles** (oder ähnlich) als dedizierter Pipeline-Orchestrator.

```
┌─────────────────────────────────────────────────────────────────┐
│                  ARISTOTELES SERVICE (NEU)                      │
│                     Port: 9160 (gRPC)                           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   PIPELINE ENGINE                        │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐        │   │
│  │  │  Intent    │→ │  Strategy  │→ │  Enrichment│→ ...   │   │
│  │  │  Analyzer  │  │  Selector  │  │  Loop      │        │   │
│  │  └────────────┘  └────────────┘  └────────────┘        │   │
│  │                                                          │   │
│  │  Features:                                               │   │
│  │  • Iterative Agent-Schleifen                            │   │
│  │  • Konditionelle Verzweigungen                          │   │
│  │  • Parallele Agent-Ausführung                           │   │
│  │  • Qualitäts-Checkpoints                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              ↓                                  │
│  Kommuniziert mit: Platon (Policies), Leibniz (Agents),        │
│                    Turing (LLM), Babbage (NLP)                  │
└─────────────────────────────────────────────────────────────────┘
```

**Vorteile:**
- Maximale Flexibilität
- Klare Trennung der Verantwortlichkeiten
- Eigene Optimierung möglich
- Iterative Loops native unterstützt

**Nachteile:**
- Neuer Service = mehr Komplexität
- Zusätzlicher Netzwerk-Hop
- Mehr Code zu maintainen

### 2.3 Option C: Hybrid-Ansatz

Kombination: Einfache Fälle via Platon-Handler, komplexe via Orchestrator.

```
User Prompt
     ↓
┌─────────────────────────────────────────────────────────────────┐
│  PLATON: IntentAnalyzerHandler (Priority 10)                    │
│  → Schnelle Intent-Erkennung                                    │
│  → Entscheidet: simple vs. complex                              │
└─────────────────────────────────────────────────────────────────┘
     ↓                                    ↓
┌─────────────────┐              ┌─────────────────────────────┐
│  SIMPLE PATH    │              │  COMPLEX PATH               │
│  (direct_llm)   │              │  (multi_agent, iterative)   │
│                 │              │                              │
│  Platon →       │              │  Aristoteles →               │
│  Turing         │              │  Agent-Loop →                │
│                 │              │  Turing                      │
└─────────────────┘              └─────────────────────────────┘
```

**Empfehlung:** Start mit Option A (Platon-Erweiterung), später Option C bei Bedarf.

---

## 3. Iterative Agent-Pipeline

### 3.1 Konzept: Prompt-Verfeinerungsschleife

Ein Prompt kann mehrfach durch Agenten laufen, bis er "fertig" ist:

```
┌─────────────────────────────────────────────────────────────────┐
│                 ITERATIVE REFINEMENT LOOP                       │
│                                                                 │
│      ┌──────────────────────────────────────────────────┐      │
│      │                                                   │      │
│      ↓                                                   │      │
│  ┌────────┐    ┌────────────┐    ┌──────────┐    ┌─────┴────┐ │
│  │ Prompt │ →  │  Agent A   │ →  │ Evaluate │ →  │ Fertig?  │ │
│  │        │    │ (Analyze)  │    │ Quality  │    │          │ │
│  └────────┘    └────────────┘    └──────────┘    └────┬─────┘ │
│                                                        │       │
│                                         Nein ←────────┤       │
│                                                        │       │
│                                         Ja ───────────→ Exit  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Beispiel: Web-Recherche mit Verfeinerung

```
User: "Was sind die besten Go-Web-Frameworks 2025?"

Iteration 1:
├─ IntentAnalyzer: intent=web_research, confidence=0.95
├─ AgentSelector: agents=[web-researcher]
├─ Web-Researcher führt Suche durch
├─ Evaluator: "Ergebnisse vorhanden, aber unstrukturiert"
└─ Entscheidung: Weiter verfeinern

Iteration 2:
├─ Agent: task-planner
├─ Task-Planner strukturiert die Recherche-Ergebnisse
├─ Evaluator: "Gut strukturiert, bereit für LLM"
└─ Entscheidung: An Turing übergeben

Final:
├─ Angereicherter Prompt an Turing
├─ Prompt enthält: Original + Recherche-Daten + Struktur
└─ Turing generiert finale Antwort
```

### 3.3 Processing Context State

```go
type PipelineState struct {
    // Intent-Analyse
    Intent          string            // "web_research", "task_decomposition", etc.
    IntentConfidence float64          // 0.0 - 1.0
    IntentReasoning  string           // Begründung der Entscheidung

    // Routing
    TargetAgents    []string          // Ausgewählte Agenten
    TargetService   string            // "turing", "leibniz", "multi_agent"

    // Anreicherungen
    Enrichments     []Enrichment      // Web-Recherche, Fakten, etc.
    ModifiedPrompt  string            // Angereicherter Prompt

    // Iteration Control
    Iteration       int               // Aktuelle Iteration
    MaxIterations   int               // Limit
    QualityScore    float64           // Aktuelle Qualität
    QualityThreshold float64          // Mindestqualität

    // Audit
    StepLog         []PipelineStep    // Alle durchlaufenen Schritte
}

type Enrichment struct {
    Source    string            // "web_search", "knowledge_base", etc.
    Content   string            // Angereicherte Daten
    Metadata  map[string]string // Zusätzliche Infos
}
```

---

## 4. Intent-Analyse

### 4.1 Intent-Kategorien

| Intent | Beschreibung | Routing |
|--------|--------------|---------|
| `direct_llm` | Einfache Fragen, Erklärungen, Code-Generierung | Direkt zu Turing |
| `web_research` | Aktuelle Informationen erforderlich | Web-Researcher Agent |
| `task_decomposition` | Komplexe Aufgabe zerlegen | Task-Planner Agent |
| `code_analysis` | Code-Review, Debugging | Code-Reviewer Agent |
| `knowledge_retrieval` | Aus Wissensdatenbank abrufen | Hypatia (RAG) |
| `multi_agent` | Mehrere Agenten koordiniert | Multi-Agent Orchestration |

### 4.2 LLM-basierte Intent-Erkennung

```go
const intentAnalysisPrompt = `Du bist ein Intent-Klassifikator für ein KI-System.
Analysiere die folgende Benutzeranfrage und bestimme die beste Verarbeitungsstrategie.

Verfügbare Strategien:
1. direct_llm: Allgemeine Fragen, Erklärungen, Code-Generierung ohne externe Daten
2. web_research: Benötigt aktuelle Informationen aus dem Internet (News, Preise, Wetter, aktuelle Ereignisse)
3. task_decomposition: Komplexe Aufgabe die in Teilschritte zerlegt werden muss
4. code_analysis: Analyse, Review oder Debugging von bestehendem Code
5. knowledge_retrieval: Informationen aus einer Wissensdatenbank abrufen
6. multi_agent: Benötigt mehrere spezialisierte Agenten

Benutzeranfrage:
"""
{{.Prompt}}
"""

Antworte NUR im folgenden JSON-Format:
{
  "intent": "<strategy>",
  "confidence": <0.0-1.0>,
  "reasoning": "<kurze Begründung>",
  "suggested_agents": ["<agent_id>", ...],
  "needs_enrichment": <true/false>,
  "enrichment_type": "<web_search|knowledge_base|none>"
}`
```

### 4.3 Beispiel-Klassifikationen

```
Prompt: "Erkläre mir Rekursion in Python"
→ {
    "intent": "direct_llm",
    "confidence": 0.95,
    "reasoning": "Allgemeine Programmiererklärung ohne externe Daten",
    "suggested_agents": [],
    "needs_enrichment": false
  }

Prompt: "Was sind die aktuellen Nachrichten zu KI-Regulierung?"
→ {
    "intent": "web_research",
    "confidence": 0.92,
    "reasoning": "Fragt nach aktuellen Informationen (Nachrichten)",
    "suggested_agents": ["web-researcher"],
    "needs_enrichment": true,
    "enrichment_type": "web_search"
  }

Prompt: "Erstelle eine vollständige REST-API mit Auth und Tests"
→ {
    "intent": "task_decomposition",
    "confidence": 0.88,
    "reasoning": "Komplexe Aufgabe mit mehreren Komponenten",
    "suggested_agents": ["task-planner"],
    "needs_enrichment": false
  }

Prompt: "Recherchiere Go-Frameworks und erstelle einen Vergleichsbericht"
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

### 5.1 Request-Flow mit Agentic Pipeline

```
┌──────────────────────────────────────────────────────────────────┐
│                       NEUER REQUEST FLOW                         │
│                                                                  │
│  Client                                                          │
│    ↓                                                             │
│  Kant API Gateway (:8080)                                        │
│    │                                                             │
│    ├─ POST /api/v1/chat                                         │
│    │   ├─ [NEU] Sende an Platon.ProcessPre() mit Pipeline-ID    │
│    │   │        "agentic-pipeline"                               │
│    │   │                                                         │
│    │   │  ┌────────────────────────────────────────────────┐    │
│    │   │  │ PLATON AGENTIC PIPELINE                        │    │
│    │   │  │                                                 │    │
│    │   │  │ 1. IntentAnalyzerHandler                       │    │
│    │   │  │    → Erkennt Intent via LLM                    │    │
│    │   │  │    → ctx.State["intent"] = "web_research"      │    │
│    │   │  │                                                 │    │
│    │   │  │ 2. AgentSelectorHandler                        │    │
│    │   │  │    → Wählt Agenten basierend auf Intent        │    │
│    │   │  │    → ctx.State["agents"] = ["web-researcher"]  │    │
│    │   │  │                                                 │    │
│    │   │  │ 3. EnrichmentHandler                           │    │
│    │   │  │    → Führt Web-Recherche durch (wenn nötig)    │    │
│    │   │  │    → Reichert Prompt mit Ergebnissen an        │    │
│    │   │  │                                                 │    │
│    │   │  │ 4. PolicyHandler                               │    │
│    │   │  │    → PII-Check, Safety-Check                   │    │
│    │   │  │                                                 │    │
│    │   │  │ Return: ProcessingContext mit Routing-Info     │    │
│    │   │  └────────────────────────────────────────────────┘    │
│    │   │                                                         │
│    │   ├─ [NEU] Lese Routing aus Response                       │
│    │   │                                                         │
│    │   └─ Route basierend auf ctx.State["target_service"]:      │
│    │       ├─ "turing"  → Turing.Chat(enrichedPrompt)           │
│    │       ├─ "leibniz" → Leibniz.Execute(agent, prompt)        │
│    │       └─ "multi"   → MultiAgent.Orchestrate(agents, task)  │
│    │                                                             │
│    └─ Return Response                                            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 5.2 Neue Platon-Handler

```go
// internal/platon/handlers/intent.go
type IntentAnalyzerHandler struct {
    *BaseHandler
    llmClient turing.TuringServiceClient
    config    IntentAnalyzerConfig
}

func (h *IntentAnalyzerHandler) Process(ctx *chain.ProcessingContext) error {
    // 1. Prompt an LLM senden für Intent-Analyse
    analysis, err := h.analyzeIntent(ctx.Context(), ctx.Prompt)
    if err != nil {
        return err
    }

    // 2. Ergebnis in Context speichern
    ctx.SetState("intent", analysis.Intent)
    ctx.SetState("intent_confidence", analysis.Confidence)
    ctx.SetState("intent_reasoning", analysis.Reasoning)
    ctx.SetState("suggested_agents", analysis.SuggestedAgents)
    ctx.SetState("needs_enrichment", analysis.NeedsEnrichment)

    return nil
}

// internal/platon/handlers/enrichment.go
type EnrichmentHandler struct {
    *BaseHandler
    webSearchClient websearch.Client
    ragClient       hypatia.HypatiaServiceClient
}

func (h *EnrichmentHandler) Process(ctx *chain.ProcessingContext) error {
    needsEnrichment, _ := ctx.GetState("needs_enrichment").(bool)
    if !needsEnrichment {
        return nil
    }

    enrichmentType, _ := ctx.GetState("enrichment_type").(string)

    switch enrichmentType {
    case "web_search":
        results, err := h.webSearchClient.Search(ctx.Context(), ctx.Prompt, 5)
        if err != nil {
            return err
        }
        ctx.SetState("enrichments", results)
        ctx.Prompt = h.enrichPrompt(ctx.Prompt, results)
        ctx.MarkModified()

    case "knowledge_base":
        // RAG-Suche via Hypatia
        // ...
    }

    return nil
}
```

### 5.3 Kant-Integration

```go
// internal/kant/handler/handler.go - handleChat() erweitern

func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
    // ... bestehender Code ...

    // NEU: Agentic Pipeline via Platon
    if h.config.EnableAgenticPipeline {
        preResp, err := h.clients.Platon.ProcessPre(ctx, &platonpb.ProcessRequest{
            Prompt:     userMessage,
            PipelineId: "agentic-pipeline",
            Metadata:   map[string]string{"source": "chat"},
        })
        if err != nil {
            // Fallback zu direktem LLM-Aufruf
            log.Warn("Agentic pipeline failed, falling back to direct LLM", "error", err)
        } else {
            // Routing basierend auf Pipeline-Ergebnis
            targetService := preResp.Metadata["target_service"]
            enrichedPrompt := preResp.ProcessedPrompt

            switch targetService {
            case "leibniz":
                return h.routeToLeibniz(w, r, preResp)
            case "multi_agent":
                return h.routeToMultiAgent(w, r, preResp)
            default:
                userMessage = enrichedPrompt // Angereicherter Prompt
            }
        }
    }

    // Bestehender Turing-Aufruf mit (evtl. angereichertem) Prompt
    // ...
}
```

---

## 6. Konfiguration

### 6.1 Pipeline-Definition

```toml
# configs/config.toml

[platon.pipelines.agentic-pipeline]
enabled = true
description = "Intelligentes Prompt-Routing mit Agent-Anreicherung"

[[platon.pipelines.agentic-pipeline.handlers]]
name = "intent-analyzer"
priority = 10
type = "pre"
config = { model = "llama3.2:3b", timeout = "3s" }

[[platon.pipelines.agentic-pipeline.handlers]]
name = "agent-selector"
priority = 20
type = "pre"
config = { }

[[platon.pipelines.agentic-pipeline.handlers]]
name = "enrichment"
priority = 30
type = "pre"
config = { max_results = 5, enable_web_search = true }

[[platon.pipelines.agentic-pipeline.handlers]]
name = "policy-pii"
priority = 100
type = "both"

[[platon.pipelines.agentic-pipeline.handlers]]
name = "audit"
priority = 1000
type = "both"
```

### 6.2 Kant-Konfiguration

```toml
[kant]
port = 8080
enable_agentic_pipeline = true
default_pipeline = "agentic-pipeline"
fallback_on_error = true  # Bei Pipeline-Fehler direkt zu Turing
```

---

## 7. UI-Integration (ChatClient TUI)

### 7.1 Status-Anzeige

```
┌─────────────────────────────────────────────────────────────────┐
│  mDW Chat                                           llama3.2:8b │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User: Was sind die aktuellen Nachrichten zu KI?               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔍 Web-Recherche wird durchgeführt...                   │   │
│  │    Intent: web_research (95% confidence)                │   │
│  │    Agent: web-researcher                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Assistant: Basierend auf meiner aktuellen Recherche...        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  > _                                                            │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Routing-Indikator

| Symbol | Bedeutung |
|--------|-----------|
| 💬 | Direct LLM (Standard-Chat) |
| 🔍 | Web-Recherche aktiv |
| 📋 | Aufgabenzerlegung aktiv |
| 🔧 | Code-Analyse aktiv |
| 🤖 | Multi-Agent aktiv |

---

## 8. Implementierungs-Roadmap

### Phase 1: Foundation (MVP)
- [ ] IntentAnalyzerHandler in Platon
- [ ] Basis-Intent-Kategorien (direct_llm, web_research)
- [ ] Kant-Integration für Pipeline-Aufruf
- [ ] Einfaches UI-Feedback im ChatClient

### Phase 2: Enrichment
- [ ] EnrichmentHandler mit Web-Recherche
- [ ] AgentSelectorHandler
- [ ] Prompt-Anreicherung mit Recherche-Ergebnissen
- [ ] RAG-Integration via Hypatia

### Phase 3: Iteration
- [ ] Iterative Verfeinerungsschleife
- [ ] Qualitäts-Evaluation
- [ ] Konfigurierbare Max-Iterationen
- [ ] Multi-Agent Routing

### Phase 4: Optimierung
- [ ] Caching für Intent-Klassifikation
- [ ] Parallele Agent-Ausführung
- [ ] Performance-Metriken
- [ ] A/B-Testing verschiedener Routing-Strategien

---

## 9. Multi-LLM-Strategie: Spezialisierte Modelle pro Agent

### 9.1 Ollama Multi-Modell-Fähigkeit

Ollama unterstützt das gleichzeitige Laden mehrerer Modelle im VRAM:

```bash
# Konfiguration für Multi-Modell
export OLLAMA_MAX_LOADED_MODELS=4      # Max. 4 Modelle gleichzeitig
export OLLAMA_NUM_PARALLEL=4           # Parallele Requests pro Modell
export OLLAMA_KEEP_ALIVE="10m"         # Modelle 10 Min im RAM halten
```

**Wichtig:** Neue Modelle müssen komplett in den verfügbaren VRAM passen. Bei unzureichendem VRAM wird teilweise auf CPU ausgelagert (Performance-Einbuße).

### 9.2 Empfohlene Modelle pro Agent/Aufgabe

| Agent | Modell | VRAM | Begründung |
|-------|--------|------|------------|
| **Intent-Analyzer** | `llama3.2:3b` | ~2GB | Schnell, für Klassifikation optimiert |
| **Web-Researcher** | `llama3.2:8b` | ~5GB | Gute Zusammenfassung von Recherchen |
| **Code-Writer** | `qwen2.5-coder:7b` | ~5GB | 88.4% HumanEval, 92+ Programmiersprachen |
| **Code-Reviewer** | `qwen2.5-coder:7b` | ~5GB | Spezialisiert auf Code-Analyse |
| **Task-Planner** | `deepseek-r1:7b` | ~5GB | Starkes logisches Reasoning |
| **General Chat** | `llama3.2:8b` | ~5GB | Allrounder für Standard-Anfragen |

### 9.3 VRAM-Planung

**Beispiel: 24GB VRAM (RTX 3090/4090)**
```
llama3.2:3b    (Intent)     ~2GB
llama3.2:8b    (General)    ~5GB
qwen2.5-coder:7b (Coding)   ~5GB
deepseek-r1:7b (Reasoning)  ~5GB
─────────────────────────────────
Total:                      ~17GB (7GB Reserve für KV-Cache)
```

**Beispiel: 12GB VRAM (RTX 3060/4070)**
```
llama3.2:3b    (Intent)     ~2GB
llama3.2:8b    (General)    ~5GB
─────────────────────────────────
Total:                      ~7GB (5GB Reserve)
# Coding-Modell wird bei Bedarf geladen (swap)
```

### 9.4 Turing-Integration: Model-per-Agent

```go
// internal/turing/service/service.go

// AgentModelMapping definiert welches LLM pro Agent genutzt wird
var AgentModelMapping = map[string]string{
    "intent-analyzer":  "llama3.2:3b",
    "web-researcher":   "llama3.2:8b",
    "code-writer":      "qwen2.5-coder:7b",
    "code-reviewer":    "qwen2.5-coder:7b",
    "task-planner":     "deepseek-r1:7b",
    "default":          "llama3.2:8b",
}

// GetModelForAgent gibt das optimale Modell für einen Agenten zurück
func GetModelForAgent(agentID string) string {
    if model, ok := AgentModelMapping[agentID]; ok {
        return model
    }
    return AgentModelMapping["default"]
}
```

### 9.5 Konfiguration in config.toml

```toml
[turing.models]
# Standard-Modell für unspezifische Anfragen
default = "llama3.2:8b"

# Agent-spezifische Modelle
[turing.models.agents]
intent-analyzer = "llama3.2:3b"
web-researcher = "llama3.2:8b"
code-writer = "qwen2.5-coder:7b"
code-reviewer = "qwen2.5-coder:7b"
task-planner = "deepseek-r1:7b"

[turing.ollama]
max_loaded_models = 4
keep_alive = "10m"
num_parallel = 4
```

### 9.6 Dynamische Modell-Auswahl im Pipeline-Flow

```
User: "Schreibe eine Go-Funktion für Fibonacci"

1. Intent-Analyzer (llama3.2:3b - schnell)
   → Intent: "code_generation"
   → Agent: "code-writer"

2. Agent-Selector
   → Liest Intent
   → Wählt: code-writer mit qwen2.5-coder:7b

3. Execution (via Leibniz → Turing)
   → Turing.Chat(model="qwen2.5-coder:7b", prompt=...)
   → Spezialisiertes Coding-LLM generiert optimalen Code

4. Response
   → Hochwertiger Code dank spezialisiertem Modell
```

### 9.7 Vorteile der Multi-LLM-Strategie

1. **Optimale Qualität**: Jede Aufgabe nutzt das beste verfügbare Modell
2. **Effizienz**: Kleine Modelle für einfache Tasks (Intent), große für komplexe
3. **Kosten**: Lokale Ausführung, keine API-Kosten
4. **Latenz**: Kleine Modelle für Klassifikation = schnelle Routing-Entscheidung
5. **Spezialisierung**: Coding-Modelle übertreffen General-Purpose bei Code

---

## 10. Zusammenfassung

Die Agentic Pipeline erweitert mDW um intelligentes Prompt-Routing:

1. **Automatische Intent-Erkennung** via LLM analysiert jeden Prompt
2. **Dynamisches Routing** wählt die optimale Verarbeitungsstrategie
3. **Prompt-Anreicherung** fügt relevante Informationen hinzu (Web-Recherche, RAG)
4. **Iterative Verfeinerung** verbessert Prompts durch mehrere Agent-Durchläufe
5. **Nahtlose Integration** in bestehende Platon-Pipeline-Infrastruktur

Das System macht mDW "intelligent" - es versteht die Absicht des Nutzers und wählt automatisch den besten Weg zur Antwort.
