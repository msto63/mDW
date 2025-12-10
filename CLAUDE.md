# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## 🎯 Project Overview

**Project**: meinDENKWERK (mDW) - Lokale KI-Plattform
**Language**: Go 1.24+
**Architecture**: 8 Microservices (gRPC + REST)
**Status**: Active Development

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
make run-all                  # Alle Services

# Test
make test                     # Alle Tests
make test-coverage            # Mit Coverage-Report

# Container (Production)
make podman-up                # Alle Services starten
make podman-down              # Stoppen
```

---

## 🏗️ Architecture

```
mDW/
├── cmd/mdw/                    # CLI Entry Point (Cobra)
├── internal/
│   ├── kant/                   # API Gateway (HTTP/SSE)
│   ├── russell/                # Service Discovery
│   ├── turing/                 # LLM Management
│   ├── hypatia/                # RAG Service
│   ├── leibniz/                # Agentic AI + MCP
│   ├── babbage/                # NLP Service
│   ├── bayes/                  # Logging Service
│   ├── platon/                 # Pipeline Processing (Pre-/Post-Processing)
│   └── tui/                    # Terminal UI (Bubble Tea)
├── pkg/core/                   # Shared: gRPC, health, discovery, config
├── api/proto/                  # Protobuf Definitions
├── foundation/                 # TBP Foundation (logging, error, i18n, utils)
├── containers/                 # Containerfiles per Service
├── configs/config.toml         # Hauptkonfiguration
└── podman-compose.yml          # Container Orchestration
```

---

## 🔌 Service Port Convention

### Port-Nummern-System

**Schema**: `9XYZ` wobei:
- `9` = Microservice-Präfix
- `XY` = Service-ID (zweistellig)
- `Z` = Protokoll-Suffix (0=gRPC, 1-9=variabel)

| Service | gRPC Port | HTTP Port | Service-ID | Beschreibung |
|---------|-----------|-----------|------------|--------------|
| **Kant** | - | 8080 | 00 | API Gateway (nur HTTP) |
| **Russell** | 9100 | 9101 | 10 | Service Discovery |
| **Bayes** | 9120 | 9121 | 12 | Logging & Metrics |
| **Platon** | 9130 | 9131 | 13 | Pipeline Processing (Pre-/Post-Processing) |
| **Leibniz** | 9140 | 9141 | 14 | Agentic AI + MCP |
| **Babbage** | 9150 | 9151 | 15 | NLP Processing |
| **Turing** | 9200 | 9201 | 20 | LLM Management |
| **Hypatia** | 9220 | 9221 | 22 | RAG Service |

### Port-Reservierungen

```
8000-8099: HTTP Gateways (Kant)
9100-9199: Infrastructure Services (Russell, Bayes, Platon, Leibniz, Babbage)
9200-9299: AI/ML Services (Turing, Hypatia)
9300-9399: Future expansion
```

### Externe Dienste

| Service | Port | Beschreibung |
|---------|------|--------------|
| Ollama | 11434 | LLM Backend |
| PostgreSQL | 5432 | Datenbank (optional) |
| Qdrant | 6333 | Vektordatenbank (optional) |

---

## ⚙️ Quality Standards (KPIs)

### Code-Qualitätsmetriken

| Metrik | Ziel | Kritisch | Beschreibung |
|--------|------|----------|--------------|
| **Test Coverage** | ≥ 80% | < 70% | Unit-Test-Abdeckung |
| **Cyclomatic Complexity** | ≤ 10 | > 15 | Komplexität pro Funktion |
| **Lines per File** | ≤ 500 | > 800 | Zeilen pro Datei |
| **Lines per Function** | ≤ 50 | > 80 | Zeilen pro Funktion |
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

## 📦 Versioning Convention

### Semantic Versioning (SemVer)

**PFLICHT**: Alle Komponenten müssen versioniert werden im Format `x.y.z`:
- **x** (Major): Inkompatible API-Änderungen
- **y** (Minor): Neue Features, rückwärtskompatibel
- **z** (Patch): Bug Fixes, kleine Verbesserungen

### Automatische Versionierung bei Build

Bei jedem `make build` wird die Patch-Version automatisch hochgezählt:
- Version wird aus `VERSION` Datei oder `version.go` gelesen
- Patch-Nummer wird inkrementiert
- Neue Version wird in Binary eingebettet via `-ldflags`

### Versionierungspflicht

| Komponente | Versionsdatei | Anzeige |
|------------|---------------|---------|
| **mDW CLI** | `cmd/mdw/version.go` | `mdw --version` |
| **Services** | `internal/{service}/version.go` | Health-Endpoint |
| **TUI Apps** | `internal/tui/{app}/version.go` | Statuszeile |
| **Foundation** | `foundation/version.go` | Import-Konstante |
| **API Proto** | `api/proto/version.proto` | gRPC Metadata |
| **Dokumente** | Changelog in `docs/` | Header-Kommentar |

### Version in Code

```go
// version.go - Wird durch Build-Prozess aktualisiert
package mypackage

var (
    Version   = "0.1.0"  // Wird durch -ldflags überschrieben
    BuildTime = ""       // Wird durch -ldflags gesetzt
    GitCommit = ""       // Wird durch -ldflags gesetzt
)
```

### Makefile-Integration

```makefile
# Version automatisch hochzählen
VERSION := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
NEXT_VERSION := $(shell echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}')

build:
	@echo $(NEXT_VERSION) > VERSION
	go build -ldflags "-X main.Version=$(NEXT_VERSION)" -o bin/mdw ./cmd/mdw
```

### Wichtige Regeln

1. **Keine Version = kein Release**: Komponenten ohne Version dürfen nicht released werden
2. **Version im Statusbereich**: TUI-Anwendungen zeigen Version immer in der Statuszeile
3. **Version in Logs**: Services loggen ihre Version beim Start
4. **Version in Health**: Health-Endpoints enthalten die Version

---

## 🔧 Development Guidelines

### Foundation-First Policy ⭐

**IMMER zuerst Foundation-Pakete prüfen, bevor neue Funktionalität implementiert wird**

```go
// Entscheidungsbaum:
// Brauche Funktionalität?
// ├─> foundation/core/*     → Bestehende Implementierung nutzen
// ├─> foundation/utils/*    → Bestehende Utilities nutzen
// ├─> Go stdlib             → Standardbibliothek nutzen
// └─> Neu & wiederverwendbar? → Zu Foundation hinzufügen
//                            → Komponenten-spezifisch? → In Komponente lassen
```

#### Integration Status (Stand: 2025-12-06)

| Komponente | Foundation-Paket | Integration | Status |
|------------|------------------|-------------|--------|
| **Logging** | `foundation/core/log` via `pkg/core/logging` | ✅ Alle Services | OK |
| **Error-Handling** | `foundation/core/error` | ✅ Alle Services | OK |
| **Config** | `pkg/core/config` | ✅ `cmd/mdw/cmd/serve.go` + Standalone-Entrypoints | OK |
| **Health Checks** | `pkg/core/health` | ✅ Alle Services | OK |

**Foundation-Module**:

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
```

### Prohibited Patterns 🚨

```go
// ❌ VERBOTEN: Direktes fmt.Println für Logging
fmt.Println("Debug message")

// ✅ ERFORDERLICH: Logger verwenden
logger.Debug("Debug message", "key", value)

// ❌ VERBOTEN: Panic in Library-Code
panic("something went wrong")

// ✅ ERFORDERLICH: Errors mit Foundation zurückgeben
return mdwerror.Wrap(err, "something went wrong").
    WithCode(mdwerror.CodeInternal)

// ❌ VERALTET: Einfaches fmt.Errorf für Service-Fehler
return fmt.Errorf("failed to do X: %w", err)

// ✅ ERFORDERLICH: Foundation Error mit Code und Operation
return mdwerror.Wrap(err, "failed to do X").
    WithCode(mdwerror.CodeExternalServiceError).
    WithOperation("service.DoX")

// ❌ VERBOTEN: Globale Variablen für State
var globalState = make(map[string]string)

// ✅ ERFORDERLICH: Dependency Injection
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

## 📝 File Header Convention

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

## 🧪 Testing Standards

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
```

---

## 📐 Naming Conventions

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

## 🔒 TODO-STUB Convention

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

## 🛡️ Digital Sovereignty

### Erlaubte Abhängigkeiten

- ✅ MIT/Apache/BSD Lizenzen
- ✅ Keine Telemetrie
- ✅ Offline-fähig
- ✅ Aktiv gewartet
- ✅ Open Source

### Verbotene Abhängigkeiten

- ❌ Proprietäre closed-source Libraries
- ❌ Cloud-spezifische SDKs (AWS SDK, Azure SDK, etc.)
- ❌ Pflicht-Telemetrie
- ❌ Vendor Lock-in

---

## 🔌 Service Communication

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

## 📁 Project Structure

### Standard-Service-Struktur

```
internal/{service}/
├── server/
│   └── server.go          # gRPC Server Setup
├── service/
│   └── service.go         # Business Logic
├── handler/               # (nur Kant)
│   └── handler.go         # HTTP Handler
└── {subpackage}/
    └── {feature}.go       # Feature-spezifischer Code
```

### Platon Service (Pipeline Processing)

```
internal/platon/
├── server/server.go       # gRPC Server (Process, ProcessPre, ProcessPost)
├── service/service.go     # Business Logic (Pipeline, Policy, Handler Management)
├── chain/
│   ├── chain.go           # Handler-Chain (Chain-of-Responsibility Pattern)
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
- REST-API: /api/v1/platon/*
```

### Neue Service erstellen

1. Verzeichnis erstellen: `internal/{name}/`
2. Proto definieren: `api/proto/{name}.proto`
3. Server implementieren: `internal/{name}/server/server.go`
4. Service implementieren: `internal/{name}/service/service.go`
5. Tests schreiben: `internal/{name}/service/service_test.go`
6. In `cmd/mdw/cmd/serve.go` registrieren

---

## 🚨 Common Issues & Troubleshooting

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
"turing:9200"  # ✅
"localhost:9200"  # ❌ (im Container)
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

## 📚 Key Files

| Datei | Beschreibung |
|-------|--------------|
| `CLAUDE.md` | Dieses Dokument - Entwicklungsrichtlinien |
| `PLAN.md` | Entwicklungsplan und aktueller Stand |
| `configs/config.toml` | Hauptkonfiguration |
| `api/proto/*.proto` | gRPC Service-Definitionen |
| `Makefile` | Build-Befehle |
| `podman-compose.yml` | Container-Orchestrierung |

---

## 🌍 Environment Variables

| Variable | Beschreibung | Default |
|----------|--------------|---------|
| `MDW_CONFIG` | Pfad zur Config-Datei | `./configs/config.toml` |
| `MDW_SERVICE` | Service zum Starten | `kant` |
| `MDW_LOG_LEVEL` | Log-Level | `info` |
| `OLLAMA_HOST` | Ollama API Endpoint | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI API Key | - |
| `ANTHROPIC_API_KEY` | Anthropic API Key | - |

---

## 📞 Support

**Projekt**: meinDENKWERK (mDW)
**Lizenz**: MIT
**Sprache**: Deutsch (Kommentare), Englisch (Code)
**Digital Sovereignty**: ✓ Kein Vendor Lock-in | ✓ Open-Source | ✓ Lokal installierbar

---

**meinDENKWERK** - Lokale KI-Plattform für souveräne Datenverarbeitung
