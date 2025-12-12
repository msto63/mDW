# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project Overview

**Project**: meinDENKWERK (mDW) - Lokale KI-Plattform
**Language**: Go 1.24+
**Architecture**: 9 Microservices (gRPC + REST)
**Status**: Active Development (v0.1.x)
**Platform Version**: 1.0.0

**Repository Structure**: Monorepo mit allen Microservices
**Working Directory**: Befehle vom Repository-Root ausführen

### Quick Start

```bash
# Prerequisites: Go 1.24+, protoc, Ollama

# Build
make build

# Run (Development)
make run                      # Standard-Service (kant)
make run SERVICE=turing       # Spezifischer Service
make run SERVICE=aristoteles  # Agentic Pipeline
make run-all                  # Alle Services

# Test
make test                     # Alle Tests
make test-coverage            # Mit Coverage-Report
make test-integration         # Integration Tests

# Container (Production)
make podman-up                # Alle Services starten
make podman-down              # Stoppen
```

---

## Architecture

```
mDW/
├── cmd/mdw/                    # CLI Entry Point (Cobra)
│   └── cmd/                    # CLI Commands (serve, chat, agent, analyze, search, ...)
├── internal/
│   ├── kant/                   # API Gateway (HTTP/SSE)
│   ├── russell/                # Service Discovery & Orchestration
│   ├── turing/                 # LLM Management (Ollama, OpenAI, Anthropic)
│   ├── hypatia/                # RAG Service (Vektor-Suche, sqlite-vec)
│   ├── leibniz/                # Agentic AI + MCP + Web Research
│   ├── aristoteles/            # Agentic Pipeline (Intent → Routing → Execution)
│   ├── babbage/                # NLP Service
│   ├── bayes/                  # Logging Service
│   ├── platon/                 # Pipeline Processing (Pre-/Post-Processing)
│   └── tui/                    # Terminal UI (Bubble Tea)
│       ├── chatclient/         # Chat Interface
│       ├── voiceassistant/     # Voice Assistant
│       ├── agentbuilder/       # Agent Builder
│       ├── controlcenter/      # Control Center
│       └── logviewer/          # Log Viewer
├── pkg/core/                   # Shared: gRPC, health, discovery, config, version
├── api/proto/                  # Protobuf Definitions (9 Proto-Dateien)
├── api/gen/                    # Generated gRPC Code
├── foundation/                 # Foundation Library (logging, error, i18n, utils, tcol)
├── containers/                 # Containerfiles per Service
├── configs/
│   ├── config.toml             # Hauptkonfiguration
│   └── agents/                 # 12 Agent-Konfigurationen (YAML)
├── test/integration/           # Integration & E2E Tests
└── podman-compose.yml          # Container Orchestration
```

---

## Service Port Convention

### Port-Nummern-System

**Schema**: `9XYZ` wobei:
- `9` = Microservice-Präfix
- `XY` = Service-ID (zweistellig)
- `Z` = Protokoll-Suffix (0=gRPC, 1-9=variabel)

| Service | gRPC Port | HTTP Port | Service-ID | Beschreibung |
|---------|-----------|-----------|------------|--------------|
| **Kant** | - | 8080 | 00 | API Gateway (nur HTTP) |
| **Russell** | 9100 | 9101 | 10 | Service Discovery & Orchestration |
| **Bayes** | 9120 | 9121 | 12 | Logging & Metrics |
| **Platon** | 9130 | 9131 | 13 | Pipeline Processing (Pre-/Post-Processing) |
| **Leibniz** | 9140 | 9141 | 14 | Agentic AI + MCP + Web Research |
| **Babbage** | 9150 | 9151 | 15 | NLP Processing |
| **Aristoteles** | 9160 | 9161 | 16 | Agentic Pipeline (Intent → Routing) |
| **Turing** | 9200 | 9201 | 20 | LLM Management |
| **Hypatia** | 9220 | 9221 | 22 | RAG Service |

### Port-Reservierungen

```
8000-8099: HTTP Gateways (Kant)
9100-9199: Infrastructure Services (Russell, Bayes, Platon, Leibniz, Babbage, Aristoteles)
9200-9299: AI/ML Services (Turing, Hypatia)
9300-9399: Future expansion
```

### Externe Dienste

| Service | Port | Beschreibung |
|---------|------|--------------|
| Ollama | 11434 | LLM Backend |
| PostgreSQL | 5432 | Datenbank (optional) |
| Qdrant | 6333 | Vektordatenbank (optional) |
| Searx-ng | 8888 | Web-Search (für Leibniz) |

---

## Services im Detail

### Aristoteles - Agentic Pipeline (NEU)

Intelligentes Prompt-Routing mit Intent-Analyse und Multi-Agent-Orchestration.

```
internal/aristoteles/
├── server/           # gRPC Server
├── service/          # Business Logic
├── intent/           # Intent-Analyse via LLM
├── strategy/         # Strategie-Auswahl
├── enrichment/       # Web/RAG-Anreicherung
├── quality/          # Quality-Evaluation
├── router/           # Routing & Execution
├── pipeline/         # Pipeline-Engine
├── clients/          # Service-Clients (Turing, Hypatia, Leibniz, Babbage, Platon)
├── decomposer/       # Task-Decomposition
└── orchestrator/     # Multi-Agent Orchestration
```

**Intent-Typen** (12):
1. `DIRECT_LLM` → Turing
2. `CODE_GENERATION` → Turing (qwen2.5:7b)
3. `CODE_ANALYSIS` → Turing (qwen2.5:7b)
4. `WEB_RESEARCH` → Leibniz web-search
5. `RAG_QUERY` → Hypatia
6. `TASK_DECOMPOSITION` → Leibniz planning
7. `SUMMARIZATION` → Babbage/Turing
8. `TRANSLATION` → Babbage
9. `MULTI_STEP` → Multi-Agent Orchestration
10. `CREATIVE` → Turing (erhöhte Temperature)
11. `FACTUAL` → RAG + Turing
12. `CONVERSATION` → Turing

### Leibniz - Agentic AI

Agent-Execution mit MCP-Support und Web-Research.

```
internal/leibniz/
├── server/           # gRPC Server
├── service/          # Business Logic
├── agent/            # Agent-Execution
├── agentloader/      # Agent-Loading + RAG-Style Agent Selection
├── evaluator/        # Agent Performance Evaluation
├── tools/            # Built-in Tools
├── servicetools/     # Service-Tools (RAG, NLP)
├── mcp/              # Model Context Protocol
├── websearch/        # Web-Search Agent (Searx-ng)
├── platon/           # Platon-Integration
└── store/            # Agent-Store (SQLite)
```

**12 vorkonfigurierte Agents** (`configs/agents/`):
- `default.yaml` - Standard-Agent
- `web-researcher.yaml` - Web-Research
- `code-reviewer.yaml` - Code-Review
- `creative-writer.yaml` - Kreatives Schreiben
- `data-analyst.yaml` - Datenanalyse
- `documentation-writer.yaml` - Dokumentation
- `spellchecker.yaml` - Rechtschreibprüfung
- `sql-expert.yaml` - SQL-Experte
- `summarizer.yaml` - Zusammenfassungen
- `translator.yaml` - Übersetzungen
- `tutor.yaml` - Lernassistent
- `brainstorm.yaml` - Brainstorming

### Platon - Pipeline Processing

```
internal/platon/
├── server/server.go       # gRPC Server (Process, ProcessPre, ProcessPost)
├── service/service.go     # Business Logic
├── chain/
│   ├── chain.go           # Handler-Chain (Chain-of-Responsibility)
│   ├── context.go         # Processing Context
│   └── types.go           # Type Definitions
└── handlers/
    ├── base.go            # BaseHandler + DynamicHandler
    ├── policy.go          # PolicyHandler (PII, Safety, Content, Custom)
    └── audit.go           # Audit Handler

Features:
- Pre-/Post-Processing Pipeline für LLM-Anfragen
- Handler-Chain mit Prioritäten und Abbruch-Logik
- Policy-basierte Validierung (Regex + LLM)
- PII-Erkennung (Email, Telefon, IBAN, Kreditkarte)
```

---

## Quality Standards (KPIs)

### Code-Qualitätsmetriken

| Metrik | Ziel | Kritisch | Beschreibung |
|--------|------|----------|--------------|
| **Test Coverage** | >= 80% | < 70% | Unit-Test-Abdeckung |
| **Cyclomatic Complexity** | <= 10 | > 15 | Komplexität pro Funktion |
| **Lines per File** | <= 500 | > 800 | Zeilen pro Datei |
| **Lines per Function** | <= 50 | > 80 | Zeilen pro Funktion |
| **Build Warnings** | 0 | > 5 | Compiler-Warnungen |
| **Test Pass Rate** | 100% | < 95% | Erfolgreiche Tests |
| **Lint Errors** | 0 | > 0 | golangci-lint Fehler |

### Messung

```bash
# Test Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Cyclomatic Complexity
gocyclo -over 10 .

# Lines per File
find . -name "*.go" -exec wc -l {} \; | sort -n

# Lint
golangci-lint run
```

---

## Versioning Convention

### Semantic Versioning (SemVer)

**PFLICHT**: Alle Komponenten müssen versioniert werden im Format `x.y.z`:
- **x** (Major): Inkompatible API-Änderungen
- **y** (Minor): Neue Features, rückwärtskompatibel
- **z** (Patch): Bug Fixes, kleine Verbesserungen

### Automatische Versionierung bei Build

Bei jedem `make build` wird die Patch-Version automatisch hochgezählt:
- Version wird aus `VERSION` Datei gelesen (aktuell: 0.1.x)
- Patch-Nummer wird inkrementiert
- Neue Version wird in Binary eingebettet via `-ldflags`

### Zentrale Version-Management

```go
// pkg/core/version/version.go
package version

const (
    Platform    = "1.0.0"
    Kant        = "1.0.0"
    Russell     = "1.0.0"
    Turing      = "1.0.0"
    Hypatia     = "1.0.0"
    Babbage     = "1.0.0"
    Leibniz     = "1.0.0"
    Bayes       = "1.0.0"
    Platon      = "1.0.0"
    Aristoteles = "1.0.0"
)
```

### Wichtige Regeln

1. **Keine Version = kein Release**: Komponenten ohne Version dürfen nicht released werden
2. **Version im Statusbereich**: TUI-Anwendungen zeigen Version immer in der Statuszeile
3. **Version in Logs**: Services loggen ihre Version beim Start
4. **Version in Health**: Health-Endpoints enthalten die Version

---

## Development Guidelines

### Foundation-First Policy

**IMMER zuerst Foundation-Pakete prüfen, bevor neue Funktionalität implementiert wird**

```go
// Entscheidungsbaum:
// Brauche Funktionalität?
// ├─> foundation/core/*     → Bestehende Implementierung nutzen
// ├─> foundation/utils/*    → Bestehende Utilities nutzen
// ├─> foundation/tcol/*     → Query/DSL Engine nutzen
// ├─> pkg/core/*            → Shared Core Packages nutzen
// ├─> Go stdlib             → Standardbibliothek nutzen
// └─> Neu & wiederverwendbar? → Zu Foundation hinzufügen
//                            → Komponenten-spezifisch? → In Komponente lassen
```

### Foundation-Module

```
foundation/
├── core/
│   ├── log/           # Logging Framework (Level, Format, Timer)
│   ├── error/         # Error Handling + Codes + Severity
│   ├── config/        # Config Management + Watch + Validation
│   ├── i18n/          # Internationalization + Locale + Watch
│   ├── validation/    # Validation Chain + Common Validators
│   └── errors/        # Error Standards + Utils
├── tcol/              # Query/DSL Language Engine
│   ├── parser/        # Lexer + Parser
│   ├── ast/           # AST Nodes + Visitor
│   ├── executor/      # Executor
│   ├── registry/      # Registry
│   └── client/        # Client
├── utils/
│   ├── filex/         # File Utilities
│   ├── mapx/          # Map Utilities (generics)
│   ├── mathx/         # Math + Business + Currency + Decimal
│   ├── slicex/        # Slice Utilities (generics)
│   ├── stringx/       # String Utilities + Case Conversion
│   ├── timex/         # Time Utilities
│   └── validationx/   # Validation Utilities
├── test/              # Test Infrastructure
└── examples/          # Foundation Examples
```

### pkg/core Packages

```
pkg/core/
├── logging/           # Logger wrapper (via foundation)
├── config/            # Central Config Management
├── health/            # Health Check Registry
├── discovery/         # Service Discovery Client
├── grpc/              # gRPC Utilities
├── registration/      # Service Registration
├── bayeslog/          # Bayes Logging Integration
├── cache/             # Caching
└── version/           # Central Version Management
```

### Integration

```go
// Error Handling (PFLICHT für alle Service-Fehler)
import mdwerror "github.com/msto63/mDW/foundation/core/error"
return mdwerror.Wrap(err, "operation failed").
    WithCode(mdwerror.CodeServiceInitialization).
    WithOperation("server.New")

// Logging (Foundation-Wrapper nutzen)
import "github.com/msto63/mDW/pkg/core/logging"
logger := logging.New("service-name")
logger.Info("Processing request", "userId", userId)

// Config (zentral laden, dann an Services übergeben)
import "github.com/msto63/mDW/pkg/core/config"
cfg, err := config.LoadFromEnv()  // Lädt aus MDW_CONFIG oder Default-Pfade

// Health Checks
import "github.com/msto63/mDW/pkg/core/health"
registry := health.NewRegistry("service", "1.0.0")

// Version
import "github.com/msto63/mDW/pkg/core/version"
fmt.Println(version.Platform)    // "1.0.0"
fmt.Println(version.Aristoteles) // "1.0.0"
```

### Prohibited Patterns

```go
// VERBOTEN: Direktes fmt.Println für Logging
fmt.Println("Debug message")

// ERFORDERLICH: Logger verwenden
logger.Debug("Debug message", "key", value)

// VERBOTEN: Panic in Library-Code
panic("something went wrong")

// ERFORDERLICH: Errors mit Foundation zurückgeben
return mdwerror.Wrap(err, "something went wrong").
    WithCode(mdwerror.CodeInternal)

// VERALTET: Einfaches fmt.Errorf für Service-Fehler
return fmt.Errorf("failed to do X: %w", err)

// ERFORDERLICH: Foundation Error mit Code und Operation
return mdwerror.Wrap(err, "failed to do X").
    WithCode(mdwerror.CodeExternalServiceError).
    WithOperation("service.DoX")

// VERBOTEN: Globale Variablen für State
var globalState = make(map[string]string)

// ERFORDERLICH: Dependency Injection
type Service struct {
    state map[string]string
}
```

### Required Patterns

```go
// Context immer als ersten Parameter
func (s *Service) DoSomething(ctx context.Context, input string) error

// Errors wrappen mit Foundation mdwerror
if err != nil {
    return mdwerror.Wrap(err, "failed to process input").
        WithCode(mdwerror.CodeInternal).
        WithOperation("service.DoSomething").
        WithDetail("input", input)
}

// Interfaces für Testbarkeit
type Store interface {
    Get(ctx context.Context, id string) (*Item, error)
    Set(ctx context.Context, item *Item) error
}

// Table-driven Tests
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "result", false},
        {"empty input", "", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

---

## File Header Convention

```go
// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     [package name]
// Description: [Brief description]
// Author:      Mike Stoffels with Claude
// Created:     [YYYY-MM-DD]
// License:     MIT
// ============================================================================

package packagename
```

**Hinweis**: Header nur für wichtige/zentrale Dateien verwenden, nicht für jede Datei.

---

## Testing Standards

### Test-Datei-Konvention

```
internal/service/
├── service.go
├── service_test.go      # Unit Tests
└── service_integration_test.go  # Integration Tests (optional)
```

### Test-Namenskonvention

```go
// Format: TestFunctionName_Scenario_ExpectedBehavior
func TestService_Create_WithValidInput_ReturnsSuccess(t *testing.T)
func TestService_Create_WithEmptyName_ReturnsError(t *testing.T)
func TestService_Get_NotFound_ReturnsNil(t *testing.T)
```

### Test-Template

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    InputType{Value: "test"},
            expected: OutputType{Result: "success"},
            wantErr:  false,
        },
        {
            name:     "empty input",
            input:    InputType{},
            expected: OutputType{},
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Test-Anforderungen

- [ ] Happy Path getestet
- [ ] Null/Empty Input getestet
- [ ] Edge Cases getestet
- [ ] Error Handling getestet
- [ ] Concurrent Access getestet (wenn relevant)

### Test-Ausführung

```bash
# Alle Tests
go test ./...

# Mit Verbose Output
go test -v ./...

# Spezifisches Paket
go test ./internal/turing/...

# Mit Coverage
go test -cover ./...

# Coverage-Report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration Tests
make test-integration
make test-integration-turing
make test-integration-russell
```

---

## Naming Conventions

### Go-Konventionen

| Element | Konvention | Beispiel |
|---------|------------|----------|
| Package | lowercase, kurz | `config`, `health` |
| Exported Types | PascalCase | `ServiceConfig`, `HealthCheck` |
| Unexported Types | camelCase | `serviceImpl`, `healthChecker` |
| Interfaces | PascalCase, oft mit -er | `Reader`, `Store`, `Handler` |
| Constants | PascalCase oder UPPER_CASE | `MaxRetries`, `DEFAULT_TIMEOUT` |
| Variablen | camelCase | `userID`, `httpClient` |
| Funktionen | PascalCase (exported) | `NewService`, `HandleRequest` |
| Methoden | PascalCase (exported) | `(s *Service) Start()` |
| Test-Funktionen | Test + PascalCase | `TestServiceStart` |
| Benchmark | Benchmark + PascalCase | `BenchmarkProcess` |

### Datei-Konventionen

| Typ | Konvention | Beispiel |
|-----|------------|----------|
| Go-Dateien | snake_case | `service_config.go` |
| Test-Dateien | `_test.go` Suffix | `service_test.go` |
| Proto-Dateien | snake_case | `turing_service.proto` |
| Config-Dateien | snake_case | `config.toml` |

### Datenbank-Konventionen

| Element | Konvention | Beispiel |
|---------|------------|----------|
| Tabellen | snake_case, Plural | `users`, `chat_messages` |
| Spalten | snake_case | `created_at`, `user_id` |
| Indices | `idx_table_column` | `idx_users_email` |
| Foreign Keys | `fk_table_ref` | `fk_messages_user` |

---

## TODO-STUB Convention

Für unimplementierte Features:

```go
// TODO-STUB: [Feature] not implemented
// Current: [Was aktuell passiert]
// Required: [Was implementiert werden muss]
func (s *Service) UnimplementedFeature(ctx context.Context) error {
    s.logger.Warn("TODO-STUB: Feature not implemented")
    return fmt.Errorf("not implemented")
}
```

---

## Digital Sovereignty

### Erlaubte Abhängigkeiten

- MIT/Apache/BSD Lizenzen
- Keine Telemetrie
- Offline-fähig
- Aktiv gewartet
- Open Source

### Verbotene Abhängigkeiten

- Proprietäre closed-source Libraries
- Cloud-spezifische SDKs (AWS SDK, Azure SDK, etc.)
- Pflicht-Telemetrie
- Vendor Lock-in

---

## Service Communication

### Protokoll-Hierarchie

1. **gRPC** (Service-zu-Service) - Bevorzugt
2. **REST/HTTP** (Client-zu-Gateway) - Nur Kant
3. **WebSocket/SSE** (Streaming) - Chat, Agent

### gRPC-Muster

```go
// Client erstellen
conn, err := grpc.Dial(address, grpc.WithInsecure())
if err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}
defer conn.Close()

client := pb.NewTuringServiceClient(conn)

// Mit Timeout aufrufen
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

resp, err := client.Chat(ctx, &pb.ChatRequest{...})
```

### Health Check Pattern

```go
// Jeder Service muss Health Checks implementieren
registry := health.NewRegistry("service-name", "1.0.0")

registry.RegisterFunc("database", func(ctx context.Context) health.CheckResult {
    if err := db.Ping(ctx); err != nil {
        return health.CheckResult{
            Status:  health.StatusUnhealthy,
            Message: err.Error(),
        }
    }
    return health.CheckResult{
        Status:  health.StatusHealthy,
        Message: "Database connected",
    }
})
```

---

## Project Structure

### Standard-Service-Struktur

```
internal/{service}/
├── server/
│   └── server.go          # gRPC Server Setup
├── service/
│   └── service.go         # Business Logic
├── handler/               # (nur Kant)
│   └── handler.go         # HTTP Handler
├── version.go             # Service Version (optional)
└── {subpackage}/
    └── {feature}.go       # Feature-spezifischer Code
```

### Neue Service erstellen

1. Verzeichnis erstellen: `internal/{name}/`
2. Proto definieren: `api/proto/{name}.proto`
3. Server implementieren: `internal/{name}/server/server.go`
4. Service implementieren: `internal/{name}/service/service.go`
5. Tests schreiben: `internal/{name}/service/service_test.go`
6. Version hinzufügen: `pkg/core/version/version.go`
7. In `cmd/mdw/cmd/serve.go` registrieren

---

## Proto-Definitionen

```
api/proto/
├── common.proto       # Shared types
├── aristoteles.proto  # Agentic Pipeline (Intent, Strategy, Pipeline)
├── babbage.proto      # NLP Service
├── bayes.proto        # Logging Service
├── hypatia.proto      # RAG Service
├── kant.proto         # (optional - HTTP Gateway)
├── leibniz.proto      # Agent Execution
├── platon.proto       # Pipeline Processing
├── russell.proto      # Service Discovery & Orchestration
└── turing.proto       # LLM Management
```

### Proto-Generierung

```bash
# Plugins installieren
make proto-install

# Proto-Dateien generieren
make proto
```

---

## Common Issues & Troubleshooting

### Proto-Generierung fehlgeschlagen

```bash
# protoc installieren
brew install protobuf  # macOS
apt-get install protobuf-compiler  # Linux

# Go-Plugins installieren
make proto-install

# Generieren
make proto
```

### gRPC-Verbindungsfehler

```bash
# Service läuft?
make status

# Port belegt?
lsof -i :9200

# Im Container: Service-Namen statt localhost
"turing:9200"  # (im Container)
"localhost:9200"  # (lokal)
```

### Test-Fehler

```bash
# Verbose Output
go test -v ./...

# Spezifischen Test
go test -v -run TestSpecificFunction ./...

# Race Conditions
go test -race ./...
```

### Build-Fehler

```bash
# Dependencies aktualisieren
go mod tidy

# Cache leeren
go clean -cache

# Neu bauen
make clean && make build
```

---

## Key Files

| Datei | Beschreibung |
|-------|--------------|
| `CLAUDE.md` | Dieses Dokument - Entwicklungsrichtlinien |
| `PLAN.md` | Entwicklungsplan und aktueller Stand |
| `VERSION` | Aktuelle Build-Version (auto-increment) |
| `configs/config.toml` | Hauptkonfiguration |
| `configs/agents/*.yaml` | Agent-Konfigurationen (12 Agents) |
| `api/proto/*.proto` | gRPC Service-Definitionen (9 Protos) |
| `pkg/core/version/version.go` | Zentrale Service-Versionen |
| `Makefile` | Build-Befehle |
| `podman-compose.yml` | Container-Orchestrierung |

---

## Makefile Targets

```bash
# Build
make build              # Build mit Auto-Version-Increment
make build-linux        # Cross-compile für Linux

# Run
make run                # Standard-Service (kant)
make run SERVICE=name   # Spezifischer Service
make run-all            # Alle Services
make dev                # Hot Reload (requires air)

# Test
make test               # Alle Tests
make test-coverage      # Mit Coverage-Report
make test-integration   # Integration Tests

# Code Quality
make lint               # golangci-lint
make fmt                # gofmt
make vet                # go vet

# Proto
make proto              # Proto-Generierung
make proto-install      # Protoc-Plugins installieren

# Container
make podman-build       # Container bauen
make podman-up          # Services starten
make podman-down        # Services stoppen
make podman-logs        # Logs anzeigen
make podman-ps          # Container Status

# Utility
make clean              # Build-Artifacts löschen
make deps               # Dependencies aktualisieren
make version            # Version anzeigen
make status             # Service-Status
make help               # Hilfe
```

---

## Environment Variables

| Variable | Beschreibung | Default |
|----------|--------------|---------|
| `MDW_CONFIG` | Pfad zur Config-Datei | `./configs/config.toml` |
| `MDW_SERVICE` | Service zum Starten | `kant` |
| `MDW_LOG_LEVEL` | Log-Level | `info` |
| `OLLAMA_HOST` | Ollama API Endpoint | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI API Key | - |
| `ANTHROPIC_API_KEY` | Anthropic API Key | - |

---

## Dependencies (go.mod)

**Go Version**: 1.24.0

**Key Dependencies**:
- `google.golang.org/grpc` v1.77.0 - gRPC Framework
- `google.golang.org/protobuf` v1.36.10 - Protocol Buffers
- `github.com/charmbracelet/bubbletea` v1.3.10 - TUI Framework
- `github.com/charmbracelet/bubbles` v0.21.0 - TUI Components
- `github.com/charmbracelet/lipgloss` v1.1.0 - TUI Styling
- `github.com/spf13/cobra` v1.8.1 - CLI Framework
- `github.com/BurntSushi/toml` v1.5.0 - TOML Parser
- `github.com/mattn/go-sqlite3` v1.14.32 - SQLite Driver
- `github.com/gorilla/websocket` v1.5.3 - WebSocket
- `github.com/google/uuid` v1.6.0 - UUID
- `gopkg.in/yaml.v3` v3.0.1 - YAML Parser
- `fyne.io/systray` v1.11.1 - System Tray
- `github.com/gordonklaus/portaudio` - Voice Input (Voice Assistant)
- `github.com/msto63/mDW/foundation` - Local Foundation Module

---

## Support

**Projekt**: meinDENKWERK (mDW)
**Lizenz**: MIT
**Sprache**: Deutsch (Kommentare), Englisch (Code)
**Digital Sovereignty**: Kein Vendor Lock-in | Open-Source | Lokal installierbar

---

**meinDENKWERK** - Lokale KI-Plattform für souveräne Datenverarbeitung
