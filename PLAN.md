# meinDENKWERK - Entwicklungsplan

> Letzte Aktualisierung: 2025-12-11

---

## Projektübersicht

**meinDENKWERK** ist eine vereinfachte, lokale Go-basierte KI-Plattform, abgeleitet von RDS DENKWERK (.NET).

### Kernmerkmale
- 9 Microservices mit klarer Aufgabentrennung (NEU: Aristoteles)
- Keine Authentifizierung (Single-User, lokal)
- Podman/Docker Deployment + lokale Binaries
- CLI (Cobra) + TUI (Bubble Tea)
- sqlite-vec für Vektorspeicherung
- MCP (Model Context Protocol) Unterstützung
- **NEU: Intelligente Agentic Pipeline für automatisches Prompt-Routing**

### Service-Architektur

| Service | Port | Funktion |
|---------|------|----------|
| **Kant** | 8080 | HTTP/REST API Gateway |
| **Russell** | 9100 | Service Discovery & Health |
| **Bayes** | 9120 | Logging & Metrics |
| **Platon** | 9130 | Pipeline/Policy Processing |
| **Leibniz** | 9140 | Agentic AI mit MCP |
| **Babbage** | 9150 | NLP Processing |
| **Aristoteles** | 9160 | **NEU: Agentic Pipeline (Intent, Routing, Enrichment)** |
| **Turing** | 9200 | LLM Management (Ollama) |
| **Hypatia** | 9220 | RAG Service (Vektor-Suche) |

---

## Aktueller Status

### Abgeschlossene Phasen (1-5)

Alle Basis-Services sind vollständig implementiert:
- Proto-Definitionen + Generierung
- Container-Infrastruktur
- CLI + TUI
- Core-Pakete
- Alle 8 gRPC-Services (Kant, Russell, Turing, Hypatia, Leibniz, Babbage, Bayes, Platon)
- Integrationstests
- Performance-Optimierung
- Dokumentation

### Kürzlich Abgeschlossen (2025-12-11)

- **Agent-spezifische Modellauswahl** in Leibniz implementiert
  - `ModelAwareLLMFunc` Typ für dynamische Modellwahl
  - Agenten können unterschiedliche LLMs nutzen (z.B. qwen2.5-coder für Code)
  - Unterstützung für Ollama Multi-Modell (OLLAMA_MAX_LOADED_MODELS)

---

## Phase 6: Aristoteles Service - Agentic Pipeline

### Übersicht

**Aristoteles** ist ein neuer Service für intelligentes Prompt-Routing und -Processing:

```
Client → Kant → Aristoteles → [Turing | Leibniz | Hypatia] → Response
                    │
                    ├─ Intent-Analyse
                    ├─ Strategie-Auswahl
                    ├─ Anreicherung (Web, RAG)
                    └─ Routing-Entscheidung
```

### Kernfunktionen

1. **Intent-Analyse**: LLM-basierte Klassifikation (llama3.2:3b, ~200ms)
2. **Strategie-Auswahl**: Automatische Modell- und Agent-Auswahl
3. **Iterative Anreicherung**: Web-Recherche, RAG, Fakten-Check
4. **Intelligentes Routing**: Zu Turing, Leibniz, oder Multi-Agent

### Konzept-Dokument

Siehe: `docs/concepts/agentic-pipeline.md` (Version 2.0)

---

## Implementierungsplan: Aristoteles Service

### Phase 6.1: Foundation (Woche 1-2)

#### 6.1.1 Service-Grundstruktur

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
└── version.go                 # Service-Version
```

**Tasks:**
- [ ] Proto-Definition erstellen (`api/proto/aristoteles.proto`)
- [ ] Proto-Code generieren
- [ ] gRPC Server-Grundgerüst
- [ ] Service-Struktur mit Config
- [ ] Health-Check implementieren
- [ ] Russell-Registrierung
- [ ] In `cmd/mdw/cmd/serve.go` integrieren
- [ ] Containerfile erstellen

#### 6.1.2 Pipeline-Engine Basis

```go
// Pipeline-Stages in Reihenfolge
var DefaultStages = []Stage{
    &IntentAnalyzerStage{},      // 1. Intent erkennen
    &StrategySelectorStage{},    // 2. Strategie wählen
    &EnrichmentStage{},          // 3. Anreichern (iterativ)
    &QualityEvaluatorStage{},    // 4. Qualität prüfen
    &PolicyCheckStage{},         // 5. Policies (via Platon)
    &RouterStage{},              // 6. Routing & Ausführung
}
```

**Tasks:**
- [ ] `PipelineEngine` Grundstruktur
- [ ] `PipelineContext` für State-Management
- [ ] `Stage` Interface definieren
- [ ] Basis-Metriken und Logging
- [ ] Unit-Tests für Pipeline-Engine

#### 6.1.3 Intent-Analyzer (MVP)

```
internal/aristoteles/intent/
├── analyzer.go            # Intent-Analyse via LLM
├── classifier.go          # Intent-Klassifikation
└── prompts.go             # LLM-Prompts für Analyse
```

**Tasks:**
- [ ] Turing-Client für Intent-Analyse
- [ ] Intent-Prompt definieren
- [ ] JSON-Parsing der LLM-Antwort
- [ ] Fallback bei Parse-Fehlern
- [ ] Basis-Intents: `direct_llm`, `web_research`, `code_generation`
- [ ] Unit-Tests für Intent-Analyzer

---

### Phase 6.2: Routing & Integration (Woche 3-4)

#### 6.2.1 Strategy-Selector

```
internal/aristoteles/strategy/
├── selector.go            # Strategie-Auswahl
└── types.go               # Strategie-Typen
```

**Tasks:**
- [ ] Intent-zu-Strategie-Mapping
- [ ] Modell-Auswahl pro Intent
- [ ] Agent-Auswahl pro Intent
- [ ] Konfigurierbare Mappings

#### 6.2.2 Router-Implementierung

```
internal/aristoteles/router/
└── router.go              # Service-Routing
```

**Tasks:**
- [ ] Turing-Client (Direct LLM)
- [ ] Leibniz-Client (Agents)
- [ ] Platon-Client (Policy-Checks)
- [ ] Hypatia-Client (RAG)
- [ ] Routing-Logik basierend auf Intent/Strategie

#### 6.2.3 Kant-Integration

**Tasks:**
- [ ] Aristoteles-Client in Kant
- [ ] Konfiguration `enable_aristotle`
- [ ] Fallback-Logik bei Aristoteles-Fehler
- [ ] REST-Endpunkte für Aristoteles

---

### Phase 6.3: Enrichment & Iteration (Woche 5-6)

#### 6.3.1 Enrichment-Stage

```
internal/aristoteles/enrichment/
├── enricher.go            # Anreicherungs-Koordinator
├── web.go                 # Web-Recherche
├── rag.go                 # RAG-Integration
└── facts.go               # Fakten-Extraktion
```

**Tasks:**
- [ ] Web-Recherche Integration (via Leibniz Web-Researcher)
- [ ] RAG via Hypatia
- [ ] Prompt-Anreicherung mit Ergebnissen
- [ ] Relevanz-Scoring für Enrichments

#### 6.3.2 Iterative Verfeinerung

```
internal/aristoteles/quality/
├── evaluator.go           # Qualitäts-Bewertung
└── thresholds.go          # Schwellenwerte
```

**Tasks:**
- [ ] Quality-Evaluator implementieren
- [ ] Schleifenlogik mit Abbruchbedingungen
- [ ] Max-Iterations-Limit
- [ ] Qualitäts-Schwellenwerte konfigurierbar

#### 6.3.3 Multi-Agent Orchestrierung

**Tasks:**
- [ ] Parallele Agent-Ausführung
- [ ] Ergebnis-Aggregation
- [ ] Timeout-Handling

---

### Phase 6.4: UI & Optimierung (Woche 7)

#### 6.4.1 ChatClient TUI Integration

**Tasks:**
- [ ] Pipeline-Status-Anzeige
- [ ] Routing-Indikatoren (Icons)
- [ ] Streaming-Events von Aristoteles
- [ ] [Aristoteles: ON/OFF] Toggle

#### 6.4.2 Performance-Optimierung

**Tasks:**
- [ ] Intent-Caching (gleiche Prompts)
- [ ] Connection-Pooling für Service-Clients
- [ ] Metriken und Monitoring
- [ ] Performance-Tests

#### 6.4.3 Integrationstests

```
test/integration/
└── aristoteles_test.go    # Aristoteles Service Tests
```

**Tasks:**
- [ ] Intent-Analyse Tests
- [ ] Routing Tests
- [ ] Enrichment Tests
- [ ] End-to-End Pipeline Tests

---

## Konfiguration

### config.toml Erweiterungen

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

# Kant - API Gateway
[kant]
enable_aristotle = true     # Aristoteles aktivieren
fallback_on_error = true    # Bei Fehler zu Turing fallback
```

---

## Geschätzter Aufwand

| Phase | Beschreibung | Aufwand | LOC |
|-------|--------------|---------|-----|
| 6.1 | Foundation | 2 Wochen | ~2000 |
| 6.2 | Routing & Integration | 2 Wochen | ~1500 |
| 6.3 | Enrichment & Iteration | 2 Wochen | ~1500 |
| 6.4 | UI & Optimierung | 1 Woche | ~1000 |
| **Gesamt** | | **7 Wochen** | **~6000** |

---

## Abhängigkeiten

### Voraussetzungen für Aristoteles

1. **Turing**: Für Intent-Analyse und finale LLM-Aufrufe
2. **Leibniz**: Für Agent-Ausführung (Web-Researcher, Code-Writer, etc.)
3. **Platon**: Für Policy-Checks (PII, Safety)
4. **Hypatia**: Für RAG-Anreicherung
5. **ModelAwareLLMFunc**: Bereits in Leibniz implementiert (2025-12-11)

### Neue Dateien zu erstellen

```
api/proto/aristoteles.proto
api/gen/aristoteles/aristoteles.pb.go
api/gen/aristoteles/aristoteles_grpc.pb.go

internal/aristoteles/
├── server/server.go
├── service/service.go
├── pipeline/engine.go
├── pipeline/context.go
├── pipeline/stages.go
├── intent/analyzer.go
├── intent/classifier.go
├── intent/prompts.go
├── strategy/selector.go
├── strategy/types.go
├── enrichment/enricher.go
├── enrichment/web.go
├── enrichment/rag.go
├── quality/evaluator.go
├── quality/thresholds.go
├── router/router.go
├── clients/turing.go
├── clients/leibniz.go
├── clients/platon.go
├── clients/hypatia.go
└── version.go

containers/aristoteles.containerfile

test/integration/aristoteles_test.go
```

---

## Risiken & Mitigationen

| Risiko | Impact | Mitigation |
|--------|--------|------------|
| Intent-Analyse ungenau | MITTEL | Fallback zu direct_llm, Konfidenz-Schwellenwert |
| Latenz durch zusätzlichen Hop | MITTEL | Caching, schnelles Modell (3b) für Intent |
| LLM-Modell nicht geladen | HOCH | Fallback-Kette, Pre-Loading |
| Enrichment-Loop endlos | HOCH | Max-Iterations + Timeout |
| Service-Ausfall | MITTEL | Fallback direkt zu Turing |

---

## Nächste Schritte

1. **Proto-Definition** für Aristoteles erstellen
2. **Service-Grundstruktur** aufsetzen
3. **Intent-Analyzer** implementieren
4. **Kant-Integration** für Routing

---

## Changelog

| Datum | Änderung |
|-------|----------|
| 2025-12-06 | Initiale Erstellung des Plans |
| 2025-12-06 | Phase 1-5 abgeschlossen |
| 2025-12-09 | Platon Service dokumentiert |
| 2025-12-11 | Agent-spezifische Modellauswahl in Leibniz implementiert |
| 2025-12-11 | Konzept Agentic Pipeline v2.0 (Aristoteles Service) |
| 2025-12-11 | **Phase 6 geplant:** Aristoteles Service für intelligentes Prompt-Routing |
