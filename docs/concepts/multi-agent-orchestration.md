# Multi-Agent Orchestration System

## Konzept: Koordinierte Agenten-Zusammenarbeit in mDW

**Version:** 1.0
**Datum:** 2025-12-10
**Autor:** Mike Stoffels mit Claude
**Status:** Konzept

---

## 1. Übersicht

### 1.1 Vision

Das Multi-Agent Orchestration System ermöglicht es mDW, komplexe Aufgaben und Projekte durch koordinierte Zusammenarbeit mehrerer spezialisierter Agenten zu lösen. Ein zentraler **Koordinator-Agent** (Orchestrator) zerlegt Aufgaben strategisch, delegiert sie an Spezialisten, bewertet Ergebnisse und führt diese zu einem qualitätsgesicherten Gesamtergebnis zusammen.

### 1.2 Kernprinzipien

1. **Strategische Zerlegung**: Große Aufgaben werden in handhabbare Teilaufgaben zerlegt
2. **Spezialisierung**: Jede Teilaufgabe wird dem am besten geeigneten Agenten zugewiesen
3. **Unabhängige Qualitätssicherung**: Ergebnisse werden von anderen Agenten überprüft
4. **Iterative Verbesserung**: Ergebnisse werden so lange verfeinert, bis sie den Anforderungen entsprechen
5. **Dynamische Erweiterung**: Neue Agenten können zur Laufzeit erzeugt werden
6. **Tool-Generierung**: Spezielle Shell-Skripte und Python-Tools können dynamisch erstellt werden

### 1.3 Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Multi-Agent Orchestration                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     KOORDINATOR (Orchestrator)                   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │  Strategist │  │  Delegator  │  │  Evaluator  │              │   │
│  │  │   Module    │  │   Module    │  │   Module    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   Agent     │  │    Tool     │  │  Result     │              │   │
│  │  │   Factory   │  │   Factory   │  │  Aggregator │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│                    ┌───────────────┼───────────────┐                   │
│                    ▼               ▼               ▼                   │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                        AGENTEN-POOL                               │  │
│  │                                                                   │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│  │  │   Statische │  │  Dynamische │  │   Review    │               │  │
│  │  │   Agenten   │  │   Agenten   │  │   Agenten   │               │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │  │
│  │                                                                   │  │
│  │  ┌───────────────────────────────────────────────────────────┐   │  │
│  │  │ • Web-Researcher    • Code-Writer      • Quality-Reviewer │   │  │
│  │  │ • Data-Analyst      • Script-Generator • Domain-Expert    │   │  │
│  │  │ • Document-Writer   • Test-Creator     • Security-Auditor │   │  │
│  │  └───────────────────────────────────────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                    │                                    │
│                    ┌───────────────┼───────────────┐                   │
│                    ▼               ▼               ▼                   │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                         TOOL-POOL                                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│  │  │   Statische │  │  Generierte │  │  Temporäre  │               │  │
│  │  │    Tools    │  │    Tools    │  │    Tools    │               │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Komponenten im Detail

### 2.1 Koordinator-Agent (Orchestrator)

Der Koordinator ist das zentrale Steuerungselement des Systems.

#### 2.1.1 Module des Koordinators

```go
// OrchestratorConfig definiert die Konfiguration des Koordinators
type OrchestratorConfig struct {
    MaxConcurrentAgents  int           // Maximale parallele Agenten
    MaxIterations        int           // Maximale Verbesserungsiterationen
    QualityThreshold     float64       // Mindestqualität (0.0-1.0)
    Timeout              time.Duration // Gesamt-Timeout für Projekt
    EnableDynamicAgents  bool          // Dynamische Agenten-Erzeugung
    EnableToolGeneration bool          // Dynamische Tool-Erzeugung
    SandboxMode          bool          // Isolierte Ausführung
}

// Orchestrator koordiniert Multi-Agenten-Projekte
type Orchestrator struct {
    strategist   *StrategistModule   // Aufgaben-Zerlegung
    delegator    *DelegatorModule    // Agenten-Zuweisung
    evaluator    *EvaluatorModule    // Qualitätsbewertung
    agentFactory *AgentFactory       // Dynamische Agenten
    toolFactory  *ToolFactory        // Dynamische Tools
    aggregator   *ResultAggregator   // Ergebnis-Zusammenführung

    agentPool    *AgentPool          // Verfügbare Agenten
    toolPool     *ToolPool           // Verfügbare Tools
    projectState *ProjectState       // Aktueller Projektstatus
}
```

#### 2.1.2 Strategist Module

Das Strategist-Modul analysiert Aufgaben und erstellt Ausführungspläne.

```go
// TaskDecomposition repräsentiert eine zerlegte Aufgabe
type TaskDecomposition struct {
    OriginalTask    string           // Ursprüngliche Aufgabe
    SubTasks        []SubTask        // Zerlegte Teilaufgaben
    Dependencies    []Dependency     // Abhängigkeiten zwischen Teilaufgaben
    ExecutionPlan   *ExecutionPlan   // Ausführungsreihenfolge
    EstimatedEffort time.Duration    // Geschätzter Aufwand
}

// SubTask repräsentiert eine Teilaufgabe
type SubTask struct {
    ID              string
    Description     string
    Type            TaskType         // Research, Code, Analysis, Review, etc.
    RequiredSkills  []string         // Benötigte Fähigkeiten
    InputFrom       []string         // IDs der Vorgänger-Tasks
    Priority        int              // Ausführungspriorität
    AcceptanceCrit  []string         // Akzeptanzkriterien
}

// Strategist-Methoden
func (s *StrategistModule) Analyze(ctx context.Context, task string) (*TaskAnalysis, error)
func (s *StrategistModule) Decompose(ctx context.Context, analysis *TaskAnalysis) (*TaskDecomposition, error)
func (s *StrategistModule) CreateExecutionPlan(ctx context.Context, decomp *TaskDecomposition) (*ExecutionPlan, error)
func (s *StrategistModule) ReplanOnFailure(ctx context.Context, failure *TaskFailure) (*ExecutionPlan, error)
```

#### 2.1.3 Delegator Module

Das Delegator-Modul weist Aufgaben den passenden Agenten zu.

```go
// AgentAssignment repräsentiert eine Agenten-Zuweisung
type AgentAssignment struct {
    TaskID      string
    AgentID     string
    AgentType   AgentType
    Instructions string           // Spezifische Anweisungen
    Context      map[string]any   // Kontext-Informationen
    Constraints  []Constraint     // Einschränkungen
    Deadline     time.Time
}

// Delegator-Methoden
func (d *DelegatorModule) SelectAgent(ctx context.Context, task *SubTask) (*Agent, error)
func (d *DelegatorModule) CreateAssignment(ctx context.Context, task *SubTask, agent *Agent) (*AgentAssignment, error)
func (d *DelegatorModule) Dispatch(ctx context.Context, assignment *AgentAssignment) (*TaskExecution, error)
func (d *DelegatorModule) Reassign(ctx context.Context, execution *TaskExecution, reason string) (*AgentAssignment, error)
```

#### 2.1.4 Evaluator Module

Das Evaluator-Modul bewertet Ergebnisse und entscheidet über Akzeptanz.

```go
// EvaluationResult repräsentiert eine Bewertung
type EvaluationResult struct {
    TaskID          string
    ResultID        string
    QualityScore    float64          // 0.0-1.0
    Accepted        bool
    Issues          []QualityIssue
    Suggestions     []Improvement
    ReviewerAgentID string           // Wer hat bewertet
}

// QualityIssue beschreibt ein gefundenes Problem
type QualityIssue struct {
    Severity    Severity  // Critical, Major, Minor, Info
    Category    string    // Correctness, Completeness, Style, etc.
    Description string
    Location    string    // Wo im Ergebnis
    Suggestion  string    // Verbesserungsvorschlag
}

// Evaluator-Methoden
func (e *EvaluatorModule) Evaluate(ctx context.Context, result *TaskResult) (*EvaluationResult, error)
func (e *EvaluatorModule) CrossReview(ctx context.Context, result *TaskResult, reviewerType AgentType) (*EvaluationResult, error)
func (e *EvaluatorModule) CompareResults(ctx context.Context, results []*TaskResult) (*ComparisonReport, error)
func (e *EvaluatorModule) DecideAcceptance(ctx context.Context, evaluations []*EvaluationResult) (bool, string)
```

### 2.2 Agent Factory

Die Agent Factory erzeugt dynamisch neue Agenten zur Laufzeit.

```go
// AgentBlueprint definiert einen dynamisch zu erzeugenden Agenten
type AgentBlueprint struct {
    Name            string
    Description     string
    SystemPrompt    string           // Kernpersönlichkeit und Expertise
    DomainKnowledge []KnowledgeItem  // Spezifisches Wissen
    Skills          []Skill          // Fähigkeiten
    Tools           []string         // Verfügbare Tools
    Constraints     []string         // Einschränkungen
    OutputFormat    OutputSpec       // Erwartetes Ausgabeformat
}

// KnowledgeItem repräsentiert ein Wissensstück
type KnowledgeItem struct {
    Category    string
    Content     string
    Source      string
    Reliability float64
}

// AgentFactory erzeugt dynamische Agenten
type AgentFactory struct {
    llmClient       LLMClient
    templateStore   *TemplateStore
    knowledgeBase   *KnowledgeBase
    securityPolicy  *SecurityPolicy
}

// Factory-Methoden
func (f *AgentFactory) CreateFromBlueprint(ctx context.Context, bp *AgentBlueprint) (*DynamicAgent, error)
func (f *AgentFactory) CreateForTask(ctx context.Context, task *SubTask) (*DynamicAgent, error)
func (f *AgentFactory) CreateSpecialist(ctx context.Context, domain string, skills []string) (*DynamicAgent, error)
func (f *AgentFactory) CloneAndModify(ctx context.Context, agentID string, modifications map[string]any) (*DynamicAgent, error)
func (f *AgentFactory) DestroyAgent(ctx context.Context, agentID string) error
```

#### 2.2.1 Dynamische Agenten-Erstellung

Der Prozess zur Erstellung eines dynamischen Agenten:

```
┌─────────────────────────────────────────────────────────────────┐
│                  Dynamische Agenten-Erstellung                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. ANALYSE DER ANFORDERUNGEN                                    │
│     ┌──────────────┐                                            │
│     │ Koordinator  │──► Analysiert Aufgabe                      │
│     │ erkennt      │──► Identifiziert benötigte Expertise       │
│     │ Bedarf       │──► Prüft ob statischer Agent existiert     │
│     └──────────────┘                                            │
│            │                                                     │
│            ▼                                                     │
│  2. BLUEPRINT-ERSTELLUNG                                         │
│     ┌──────────────┐                                            │
│     │ Agent        │──► Wählt passendes Template                │
│     │ Factory      │──► Generiert System-Prompt                 │
│     │              │──► Sammelt Domain-Wissen                   │
│     └──────────────┘                                            │
│            │                                                     │
│            ▼                                                     │
│  3. WISSENS-INJEKTION                                           │
│     ┌──────────────┐                                            │
│     │ Knowledge    │──► Lädt relevante Dokumente                │
│     │ Base         │──► Extrahiert Schlüsselinformationen       │
│     │              │──► Komprimiert auf Kontextgröße            │
│     └──────────────┘                                            │
│            │                                                     │
│            ▼                                                     │
│  4. INSTANZIIERUNG                                               │
│     ┌──────────────┐                                            │
│     │ Dynamic      │──► Registriert im Agenten-Pool             │
│     │ Agent        │──► Weist Tools zu                          │
│     │              │──► Startet Ausführungskontext              │
│     └──────────────┘                                            │
│            │                                                     │
│            ▼                                                     │
│  5. VALIDIERUNG                                                  │
│     ┌──────────────┐                                            │
│     │ Test-        │──► Führt Probe-Aufgabe aus                 │
│     │ Ausführung   │──► Prüft Ergebnisqualität                  │
│     │              │──► Gibt frei oder verwirft                 │
│     └──────────────┘                                            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 Tool Factory

Die Tool Factory erzeugt dynamisch Shell-Skripte und Python-Tools.

```go
// ToolBlueprint definiert ein zu generierendes Tool
type ToolBlueprint struct {
    Name          string
    Description   string
    Language      ToolLanguage     // Shell, Python
    Purpose       string           // Was soll erreicht werden
    InputSpec     []ParameterSpec  // Eingabe-Parameter
    OutputSpec    OutputSpec       // Erwartete Ausgabe
    Constraints   []string         // Sicherheitseinschränkungen
    Dependencies  []string         // Benötigte Pakete/Programme
}

// GeneratedTool repräsentiert ein erzeugtes Tool
type GeneratedTool struct {
    ID            string
    Blueprint     *ToolBlueprint
    SourceCode    string
    Checksum      string
    SecurityCheck *SecurityCheckResult
    Executable    string           // Pfad zum ausführbaren Script
    CreatedAt     time.Time
    ExpiresAt     time.Time        // Temporäre Tools verfallen
}

// ToolFactory erzeugt dynamische Tools
type ToolFactory struct {
    codeGenerator  *CodeGenerator
    securityChecker *SecurityChecker
    sandbox        *Sandbox
    tempDir        string
}

// Factory-Methoden
func (f *ToolFactory) GenerateShellScript(ctx context.Context, bp *ToolBlueprint) (*GeneratedTool, error)
func (f *ToolFactory) GeneratePythonTool(ctx context.Context, bp *ToolBlueprint) (*GeneratedTool, error)
func (f *ToolFactory) ValidateTool(ctx context.Context, tool *GeneratedTool) (*ValidationResult, error)
func (f *ToolFactory) ExecuteInSandbox(ctx context.Context, tool *GeneratedTool, input map[string]any) (*ToolResult, error)
func (f *ToolFactory) DestroyTool(ctx context.Context, toolID string) error
```

#### 2.3.1 Sicherheitsmodell für generierte Tools

```go
// SecurityPolicy definiert Sicherheitsrichtlinien
type SecurityPolicy struct {
    AllowedCommands    []string       // Erlaubte Shell-Befehle
    ForbiddenCommands  []string       // Verbotene Befehle
    AllowedPythonLibs  []string       // Erlaubte Python-Bibliotheken
    ForbiddenPythonLibs []string      // Verbotene Bibliotheken
    MaxExecutionTime   time.Duration  // Maximale Ausführungszeit
    MaxMemory          int64          // Maximaler Speicher (Bytes)
    MaxFileSize        int64          // Maximale Dateigröße
    AllowedPaths       []string       // Erlaubte Dateipfade
    NetworkPolicy      NetworkPolicy  // Netzwerk-Einschränkungen
}

// SecurityChecker prüft generierte Tools
type SecurityChecker struct {
    policy         *SecurityPolicy
    staticAnalyzer *StaticAnalyzer
    patternMatcher *PatternMatcher
}

// Sicherheitsprüfungen
func (c *SecurityChecker) AnalyzeShellScript(code string) (*SecurityCheckResult, error)
func (c *SecurityChecker) AnalyzePythonCode(code string) (*SecurityCheckResult, error)
func (c *SecurityChecker) DetectDangerousPatterns(code string) ([]SecurityWarning, error)
func (c *SecurityChecker) ValidateResourceUsage(tool *GeneratedTool) error
```

#### 2.3.2 Beispiel: Tool-Generierung

```yaml
# Beispiel Blueprint für ein Datenextraktions-Tool
name: "csv_aggregator"
description: "Aggregiert numerische Spalten aus mehreren CSV-Dateien"
language: python
purpose: |
  Liest alle CSV-Dateien aus einem Verzeichnis,
  extrahiert die angegebene numerische Spalte,
  berechnet Summe, Durchschnitt, Min, Max
input_spec:
  - name: "directory"
    type: "string"
    description: "Pfad zum Verzeichnis mit CSV-Dateien"
    required: true
  - name: "column"
    type: "string"
    description: "Name der zu aggregierenden Spalte"
    required: true
output_spec:
  format: "json"
  schema:
    sum: "float"
    average: "float"
    min: "float"
    max: "float"
    count: "int"
    files_processed: "list[string]"
constraints:
  - "Nur lokale Dateien, kein Netzwerkzugriff"
  - "Maximal 1000 Dateien"
  - "Timeout: 60 Sekunden"
dependencies:
  - "pandas"
  - "pathlib"
```

**Generiertes Python-Tool:**

```python
#!/usr/bin/env python3
"""
Generated Tool: csv_aggregator
Purpose: Aggregiert numerische Spalten aus mehreren CSV-Dateien
Generated: 2025-12-10T15:30:00Z
Security-Checked: True
"""

import json
import sys
from pathlib import Path

# Sandbox-Import-Beschränkungen
import pandas as pd

MAX_FILES = 1000

def aggregate_csv_column(directory: str, column: str) -> dict:
    """Aggregiert eine numerische Spalte aus CSV-Dateien."""
    dir_path = Path(directory)

    if not dir_path.is_dir():
        raise ValueError(f"Verzeichnis existiert nicht: {directory}")

    csv_files = list(dir_path.glob("*.csv"))[:MAX_FILES]

    if not csv_files:
        return {"error": "Keine CSV-Dateien gefunden"}

    values = []
    processed_files = []

    for csv_file in csv_files:
        try:
            df = pd.read_csv(csv_file)
            if column in df.columns:
                values.extend(df[column].dropna().tolist())
                processed_files.append(str(csv_file.name))
        except Exception as e:
            continue  # Fehlerhafte Dateien überspringen

    if not values:
        return {"error": f"Spalte '{column}' nicht gefunden"}

    return {
        "sum": sum(values),
        "average": sum(values) / len(values),
        "min": min(values),
        "max": max(values),
        "count": len(values),
        "files_processed": processed_files
    }

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(json.dumps({"error": "Usage: csv_aggregator <directory> <column>"}))
        sys.exit(1)

    result = aggregate_csv_column(sys.argv[1], sys.argv[2])
    print(json.dumps(result, indent=2))
```

### 2.4 Agenten-Pool

#### 2.4.1 Statische Agenten (vordefiniert)

| Agent | Typ | Beschreibung | Tools |
|-------|-----|--------------|-------|
| `web-researcher` | Research | Internet-Recherchen | web_search, fetch_webpage |
| `code-writer` | Development | Code-Entwicklung | file_read, file_write, execute |
| `data-analyst` | Analysis | Datenanalyse | python_execute, visualize |
| `document-writer` | Content | Dokumenterstellung | file_write, format |
| `test-creator` | Quality | Test-Erstellung | file_read, file_write |
| `security-auditor` | Security | Sicherheitsprüfung | scan, analyze |
| `quality-reviewer` | Review | Code-Review | file_read, analyze |

#### 2.4.2 Dynamische Agenten-Typen

```go
// AgentType definiert den Agenten-Typ
type AgentType string

const (
    AgentTypeResearch    AgentType = "research"
    AgentTypeDevelopment AgentType = "development"
    AgentTypeAnalysis    AgentType = "analysis"
    AgentTypeContent     AgentType = "content"
    AgentTypeReview      AgentType = "review"
    AgentTypeSecurity    AgentType = "security"
    AgentTypeSpecialist  AgentType = "specialist"  // Dynamisch erzeugt
)

// DynamicAgent repräsentiert einen dynamisch erzeugten Agenten
type DynamicAgent struct {
    ID            string
    Name          string
    Type          AgentType
    Blueprint     *AgentBlueprint
    SystemPrompt  string
    Knowledge     []KnowledgeItem
    Tools         []Tool
    State         AgentState
    Metrics       *AgentMetrics
    CreatedAt     time.Time
    ExpiresAt     time.Time        // Kann ablaufen
}
```

---

## 3. Workflow und Prozesse

### 3.1 Projekt-Ausführungs-Workflow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    PROJEKT-AUSFÜHRUNGS-WORKFLOW                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 1: INITIALISIERUNG                                         │    │
│  │                                                                   │    │
│  │  Benutzer ──► Aufgabe/Projekt ──► Koordinator                    │    │
│  │                                                                   │    │
│  │  • Aufgabe entgegennehmen                                        │    │
│  │  • Kontext analysieren                                           │    │
│  │  • Ressourcen prüfen                                             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                              │                                           │
│                              ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 2: STRATEGISCHE PLANUNG                                    │    │
│  │                                                                   │    │
│  │  Koordinator ──► Strategist Module                               │    │
│  │                                                                   │    │
│  │  • Aufgabe analysieren                                           │    │
│  │  • In Teilaufgaben zerlegen                                      │    │
│  │  • Abhängigkeiten identifizieren                                 │    │
│  │  • Ausführungsplan erstellen                                     │    │
│  │  • Benötigte Agenten identifizieren                              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                              │                                           │
│                              ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 3: RESSOURCEN-ALLOKATION                                   │    │
│  │                                                                   │    │
│  │  Koordinator ──► Delegator + Agent/Tool Factory                  │    │
│  │                                                                   │    │
│  │  • Statische Agenten zuweisen                                    │    │
│  │  • Dynamische Agenten erzeugen (falls nötig)                     │    │
│  │  • Tools generieren (falls nötig)                                │    │
│  │  • Wissen injizieren                                             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                              │                                           │
│                              ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 4: AUSFÜHRUNG                                              │    │
│  │                                                                   │    │
│  │  ┌─────────────────────────────────────────────────────────┐     │    │
│  │  │                    ITERATION LOOP                        │     │    │
│  │  │                                                          │     │    │
│  │  │   ┌──────────┐    ┌──────────┐    ┌──────────┐          │     │    │
│  │  │   │ Dispatch │───►│ Execute  │───►│ Collect  │          │     │    │
│  │  │   │ Tasks    │    │ Parallel │    │ Results  │          │     │    │
│  │  │   └──────────┘    └──────────┘    └──────────┘          │     │    │
│  │  │         │                               │                │     │    │
│  │  │         │         ┌──────────┐          │                │     │    │
│  │  │         └────────►│ Evaluate │◄─────────┘                │     │    │
│  │  │                   │ Quality  │                           │     │    │
│  │  │                   └──────────┘                           │     │    │
│  │  │                        │                                 │     │    │
│  │  │              ┌─────────┴─────────┐                       │     │    │
│  │  │              ▼                   ▼                       │     │    │
│  │  │        [Akzeptiert]        [Abgelehnt]                   │     │    │
│  │  │              │                   │                       │     │    │
│  │  │              │         ┌─────────┴─────────┐             │     │    │
│  │  │              │         ▼                   ▼             │     │    │
│  │  │              │    [Verbessern]        [Neu planen]       │     │    │
│  │  │              │         │                   │             │     │    │
│  │  │              │         └───────────────────┘             │     │    │
│  │  │              │                   │                       │     │    │
│  │  │              ▼                   │                       │     │    │
│  │  │        [Nächste Task]◄───────────┘                       │     │    │
│  │  │                                                          │     │    │
│  │  └──────────────────────────────────────────────────────────┘     │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                              │                                           │
│                              ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 5: AGGREGATION                                             │    │
│  │                                                                   │    │
│  │  Koordinator ──► Result Aggregator                               │    │
│  │                                                                   │    │
│  │  • Alle Teilergebnisse sammeln                                   │    │
│  │  • Konsistenz prüfen                                             │    │
│  │  • Zu Gesamtergebnis zusammenführen                              │    │
│  │  • Finale Qualitätsprüfung                                       │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                              │                                           │
│                              ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │ PHASE 6: ABSCHLUSS                                               │    │
│  │                                                                   │    │
│  │  • Ergebnis an Benutzer übergeben                                │    │
│  │  • Temporäre Ressourcen bereinigen                               │    │
│  │  • Metriken erfassen                                             │    │
│  │  • Dynamische Agenten/Tools entfernen                            │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Qualitätssicherungs-Prozess

```go
// QualityAssuranceProcess definiert den QA-Prozess
type QualityAssuranceProcess struct {
    // Stufe 1: Selbstprüfung durch ausführenden Agenten
    SelfReview bool

    // Stufe 2: Peer-Review durch anderen Agenten gleichen Typs
    PeerReview bool

    // Stufe 3: Cross-Review durch Agenten anderen Typs
    CrossReview bool
    CrossReviewerTypes []AgentType

    // Stufe 4: Spezialisierte Prüfung (Security, Performance, etc.)
    SpecializedReview bool
    SpecializedReviewers []string

    // Stufe 5: Finale Koordinator-Prüfung
    CoordinatorReview bool
}

// ReviewChain definiert die Prüfkette
type ReviewChain struct {
    Stages []ReviewStage
}

type ReviewStage struct {
    Name        string
    Reviewer    AgentType
    Criteria    []EvaluationCriterion
    MustPass    bool       // Muss bestanden werden
    CanImprove  bool       // Kann Verbesserungen vorschlagen
}
```

### 3.3 Iterativer Verbesserungs-Prozess

```
┌─────────────────────────────────────────────────────────────────┐
│              ITERATIVER VERBESSERUNGS-PROZESS                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Iteration 1          Iteration 2          Iteration N          │
│   ──────────          ──────────          ──────────            │
│                                                                  │
│   ┌─────────┐         ┌─────────┐         ┌─────────┐           │
│   │ Execute │         │ Improve │         │ Finalize│           │
│   └────┬────┘         └────┬────┘         └────┬────┘           │
│        │                   │                   │                 │
│        ▼                   ▼                   ▼                 │
│   ┌─────────┐         ┌─────────┐         ┌─────────┐           │
│   │ Result  │         │ Result  │         │ Result  │           │
│   │ v1      │         │ v2      │         │ vN      │           │
│   └────┬────┘         └────┬────┘         └────┬────┘           │
│        │                   │                   │                 │
│        ▼                   ▼                   ▼                 │
│   ┌─────────┐         ┌─────────┐         ┌─────────┐           │
│   │ Evaluate│         │ Evaluate│         │ Evaluate│           │
│   │ Score:  │         │ Score:  │         │ Score:  │           │
│   │   0.6   │         │   0.8   │         │   0.95  │           │
│   └────┬────┘         └────┬────┘         └────┬────┘           │
│        │                   │                   │                 │
│        ▼                   ▼                   ▼                 │
│   [Threshold:0.9]     [Threshold:0.9]     [Threshold:0.9]       │
│        │                   │                   │                 │
│   [NOT MET]           [NOT MET]           [MET ✓]               │
│        │                   │                   │                 │
│        ▼                   ▼                   ▼                 │
│   ┌─────────┐         ┌─────────┐         ┌─────────┐           │
│   │Feedback │         │Feedback │         │ ACCEPT  │           │
│   │+ Issues │         │+ Issues │         │ FINAL   │           │
│   └────┬────┘         └────┬────┘         └─────────┘           │
│        │                   │                                     │
│        └──────────►────────┘                                     │
│                                                                  │
│   Verbesserungen werden weitergegeben bis Qualität erreicht     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Datenstrukturen

### 4.1 Projekt-State

```go
// ProjectState repräsentiert den Zustand eines Projekts
type ProjectState struct {
    ID              string
    Name            string
    Description     string
    Status          ProjectStatus

    // Planung
    OriginalTask    string
    Decomposition   *TaskDecomposition
    ExecutionPlan   *ExecutionPlan

    // Ausführung
    ActiveTasks     map[string]*TaskExecution
    CompletedTasks  map[string]*TaskResult
    FailedTasks     map[string]*TaskFailure

    // Ressourcen
    AssignedAgents  map[string]*AgentAssignment
    GeneratedTools  map[string]*GeneratedTool
    DynamicAgents   map[string]*DynamicAgent

    // Qualität
    Evaluations     map[string][]*EvaluationResult
    Iterations      map[string]int  // Task-ID → Anzahl Iterationen

    // Ergebnis
    FinalResult     *ProjectResult

    // Metriken
    StartedAt       time.Time
    CompletedAt     time.Time
    Metrics         *ProjectMetrics
}

// ProjectStatus definiert den Projektstatus
type ProjectStatus string

const (
    ProjectStatusPending    ProjectStatus = "pending"
    ProjectStatusPlanning   ProjectStatus = "planning"
    ProjectStatusExecuting  ProjectStatus = "executing"
    ProjectStatusReviewing  ProjectStatus = "reviewing"
    ProjectStatusCompleted  ProjectStatus = "completed"
    ProjectStatusFailed     ProjectStatus = "failed"
    ProjectStatusCancelled  ProjectStatus = "cancelled"
)
```

### 4.2 Kommunikations-Protokoll zwischen Agenten

```go
// AgentMessage repräsentiert eine Nachricht zwischen Agenten
type AgentMessage struct {
    ID          string
    FromAgent   string
    ToAgent     string
    Type        MessageType
    Content     any
    Context     *MessageContext
    ReplyTo     string           // Optional: Antwort auf
    Timestamp   time.Time
}

type MessageType string

const (
    MessageTypeTaskAssignment  MessageType = "task_assignment"
    MessageTypeTaskResult      MessageType = "task_result"
    MessageTypeEvaluation      MessageType = "evaluation"
    MessageTypeFeedback        MessageType = "feedback"
    MessageTypeKnowledge       MessageType = "knowledge"
    MessageTypeQuery           MessageType = "query"
    MessageTypeStatus          MessageType = "status"
)

// MessageContext enthält zusätzlichen Kontext
type MessageContext struct {
    ProjectID    string
    TaskID       string
    Iteration    int
    Priority     int
    Deadline     time.Time
    Metadata     map[string]any
}
```

---

## 5. Beispiel-Szenarien

### 5.1 Szenario: Software-Projekt-Analyse

**Aufgabe:** "Analysiere das mDW-Projekt und erstelle einen technischen Bericht über die Architektur, Codequalität und Verbesserungspotentiale."

```yaml
# Projekt-Zerlegung durch Koordinator
project: "mDW Technical Analysis"
original_task: "Analysiere das mDW-Projekt..."

subtasks:
  - id: "arch-analysis"
    type: "analysis"
    description: "Analysiere die Microservice-Architektur"
    agent: "code-analyst"
    inputs: []

  - id: "code-quality"
    type: "analysis"
    description: "Bewerte die Codequalität mit Metriken"
    agent: "quality-reviewer"
    inputs: []
    tools_required:
      - generated: "go_metrics_collector"  # Dynamisch generiert

  - id: "security-scan"
    type: "security"
    description: "Führe Sicherheitsanalyse durch"
    agent: "security-auditor"
    inputs: []

  - id: "doc-review"
    type: "content"
    description: "Prüfe Dokumentation auf Vollständigkeit"
    agent: "document-reviewer"
    inputs: []

  - id: "improvement-suggest"
    type: "specialist"
    description: "Schlage Verbesserungen vor"
    agent: "dynamic:go-architecture-expert"  # Dynamisch erzeugt
    inputs:
      - "arch-analysis"
      - "code-quality"
      - "security-scan"

  - id: "report-write"
    type: "content"
    description: "Erstelle finalen Bericht"
    agent: "document-writer"
    inputs:
      - "arch-analysis"
      - "code-quality"
      - "security-scan"
      - "doc-review"
      - "improvement-suggest"

# Review-Kette
reviews:
  - task: "report-write"
    reviewers:
      - "quality-reviewer"    # Prüft Struktur und Vollständigkeit
      - "code-analyst"        # Prüft technische Korrektheit
    threshold: 0.85
    max_iterations: 3
```

### 5.2 Szenario: Datenverarbeitung mit generiertem Tool

**Aufgabe:** "Verarbeite alle Log-Dateien der letzten Woche und erstelle eine Fehlerstatistik."

```yaml
# Koordinator erkennt: Kein bestehendes Tool für Log-Parsing
# Generiert dynamisches Tool

tool_generation:
  name: "log_error_analyzer"
  language: "python"
  purpose: |
    Parst mDW Log-Dateien im JSON-Format,
    extrahiert Fehler-Einträge,
    kategorisiert nach Service und Schweregrad

  input_spec:
    - name: "log_dir"
      type: "string"
    - name: "days"
      type: "int"
      default: 7

  output_spec:
    format: "json"
    schema:
      total_errors: "int"
      by_service: "dict[string, int]"
      by_severity: "dict[string, int]"
      top_errors: "list[dict]"
      timeline: "list[dict]"

# Ausführungsplan
execution:
  1. "tool-generator" erstellt "log_error_analyzer"
  2. "security-auditor" prüft generiertes Tool
  3. "data-analyst" führt Tool aus mit logs/
  4. "data-analyst" interpretiert Ergebnisse
  5. "document-writer" erstellt Fehlerbericht
  6. "quality-reviewer" prüft Bericht
```

### 5.3 Szenario: Multi-Agenten Code-Entwicklung

**Aufgabe:** "Implementiere eine neue REST-Endpoint für Benutzer-Authentifizierung."

```yaml
project: "Auth Endpoint Implementation"

# Phase 1: Analyse
analysis_tasks:
  - agent: "code-analyst"
    task: "Analysiere bestehende Auth-Patterns im Projekt"
  - agent: "security-auditor"
    task: "Prüfe Sicherheitsanforderungen"
  - agent: "web-researcher"
    task: "Recherchiere Best Practices für Go Auth"

# Phase 2: Design
design_tasks:
  - agent: "dynamic:go-api-designer"
    task: "Erstelle API-Design basierend auf Analyse"
    inputs: [analysis_tasks]

# Phase 3: Implementierung (parallel)
implementation_tasks:
  - agent: "code-writer"
    task: "Implementiere Handler"
    inputs: [design]
  - agent: "code-writer"
    task: "Implementiere Middleware"
    inputs: [design]
  - agent: "code-writer"
    task: "Implementiere Service-Layer"
    inputs: [design]

# Phase 4: Testing
test_tasks:
  - agent: "test-creator"
    task: "Erstelle Unit-Tests"
    inputs: [implementation_tasks]
  - agent: "test-creator"
    task: "Erstelle Integration-Tests"
    inputs: [implementation_tasks]

# Phase 5: Review
review_chain:
  - reviewer: "quality-reviewer"
    focus: "Code-Qualität und Patterns"
  - reviewer: "security-auditor"
    focus: "Sicherheitslücken"
  - reviewer: "code-analyst"
    focus: "Integration mit bestehendem Code"

# Qualitätsschwelle
quality:
  threshold: 0.9
  max_iterations: 5
  mandatory_reviews:
    - "security-auditor"  # Security-Review ist Pflicht
```

---

## 6. API-Spezifikation

### 6.1 gRPC Service Definition

```protobuf
syntax = "proto3";

package mDW.orchestrator;

service OrchestratorService {
  // Projekt-Management
  rpc CreateProject(CreateProjectRequest) returns (Project);
  rpc GetProject(GetProjectRequest) returns (Project);
  rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse);
  rpc CancelProject(CancelProjectRequest) returns (CancelProjectResponse);

  // Ausführung
  rpc ExecuteTask(ExecuteTaskRequest) returns (stream TaskProgress);
  rpc GetTaskStatus(GetTaskStatusRequest) returns (TaskStatus);

  // Agenten-Management
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
  rpc CreateDynamicAgent(CreateAgentRequest) returns (Agent);
  rpc DestroyAgent(DestroyAgentRequest) returns (DestroyAgentResponse);

  // Tool-Management
  rpc ListTools(ListToolsRequest) returns (ListToolsResponse);
  rpc GenerateTool(GenerateToolRequest) returns (GeneratedTool);
  rpc ExecuteTool(ExecuteToolRequest) returns (ToolResult);
  rpc DestroyTool(DestroyToolRequest) returns (DestroyToolResponse);
}

message CreateProjectRequest {
  string name = 1;
  string description = 2;
  string task = 3;
  OrchestratorConfig config = 4;
}

message ExecuteTaskRequest {
  string project_id = 1;
  bool stream_progress = 2;
}

message TaskProgress {
  string project_id = 1;
  string task_id = 2;
  string agent_id = 3;
  string status = 4;
  float progress = 5;
  string message = 6;
  map<string, string> metadata = 7;
}

message CreateAgentRequest {
  AgentBlueprint blueprint = 1;
  string project_id = 2;
}

message GenerateToolRequest {
  ToolBlueprint blueprint = 1;
  string project_id = 2;
}
```

### 6.2 REST API (über Kant Gateway)

```yaml
# OpenAPI 3.0 Auszug
paths:
  /api/v1/orchestrator/projects:
    post:
      summary: Neues Projekt erstellen
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateProjectRequest'
      responses:
        '201':
          description: Projekt erstellt
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'

  /api/v1/orchestrator/projects/{projectId}/execute:
    post:
      summary: Projekt ausführen
      parameters:
        - name: projectId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: SSE Stream der Fortschrittsmeldungen
          content:
            text/event-stream:
              schema:
                $ref: '#/components/schemas/TaskProgress'

  /api/v1/orchestrator/agents:
    get:
      summary: Alle verfügbaren Agenten auflisten
    post:
      summary: Dynamischen Agenten erstellen

  /api/v1/orchestrator/tools/generate:
    post:
      summary: Tool generieren
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ToolBlueprint'
```

---

## 7. Sicherheitskonzept

### 7.1 Sandbox-Architektur

```
┌─────────────────────────────────────────────────────────────────┐
│                      SANDBOX-ARCHITEKTUR                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    HOST-SYSTEM (mDW)                       │  │
│  │                                                            │  │
│  │   ┌────────────────────────────────────────────────────┐  │  │
│  │   │              ORCHESTRATOR (Leibniz)                 │  │  │
│  │   │                                                     │  │  │
│  │   │   ┌───────────────────────────────────────────┐    │  │  │
│  │   │   │           SANDBOX CONTAINER                │    │  │  │
│  │   │   │                                            │    │  │  │
│  │   │   │  ┌──────────────┐  ┌──────────────┐       │    │  │  │
│  │   │   │  │  Generierte  │  │  Dynamische  │       │    │  │  │
│  │   │   │  │    Tools     │  │   Agenten    │       │    │  │  │
│  │   │   │  └──────────────┘  └──────────────┘       │    │  │  │
│  │   │   │                                            │    │  │  │
│  │   │   │  Einschränkungen:                         │    │  │  │
│  │   │   │  • Kein Netzwerk (außer whitelisted)      │    │  │  │
│  │   │   │  • Limitierter Dateizugriff               │    │  │  │
│  │   │   │  • CPU/Memory Limits                      │    │  │  │
│  │   │   │  • Zeitlimits                             │    │  │  │
│  │   │   │  • Audit-Logging aller Aktionen           │    │  │  │
│  │   │   └───────────────────────────────────────────┘    │  │  │
│  │   │                                                     │  │  │
│  │   └────────────────────────────────────────────────────┘  │  │
│  │                                                            │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Sicherheitsrichtlinien

```go
// DefaultSecurityPolicy definiert Standard-Sicherheitsrichtlinien
func DefaultSecurityPolicy() *SecurityPolicy {
    return &SecurityPolicy{
        // Shell-Befehle
        AllowedCommands: []string{
            "ls", "cat", "grep", "awk", "sed", "sort", "uniq",
            "head", "tail", "wc", "find", "xargs", "cut",
            "jq", "yq", "curl",  // Nur mit URL-Whitelist
        },
        ForbiddenCommands: []string{
            "rm", "rmdir", "mv", "cp",  // Nur in expliziten Pfaden
            "sudo", "su", "chmod", "chown",
            "dd", "mkfs", "fdisk",
            "shutdown", "reboot", "init",
            "iptables", "systemctl",
        },

        // Python
        AllowedPythonLibs: []string{
            "json", "csv", "re", "math", "statistics",
            "datetime", "pathlib", "collections",
            "pandas", "numpy", "matplotlib",
            "requests",  // Nur mit URL-Whitelist
        },
        ForbiddenPythonLibs: []string{
            "os.system", "subprocess.Popen",
            "socket", "http.server",
            "pickle",  // Sicherheitsrisiko
            "eval", "exec",
        },

        // Ressourcen
        MaxExecutionTime: 5 * time.Minute,
        MaxMemory:        512 * 1024 * 1024,  // 512 MB
        MaxFileSize:      50 * 1024 * 1024,   // 50 MB

        // Netzwerk
        NetworkPolicy: NetworkPolicy{
            AllowOutbound:   false,
            WhitelistedURLs: []string{},  // Projektspezifisch
        },
    }
}
```

### 7.3 Audit-Trail

```go
// AuditEvent repräsentiert ein Audit-Ereignis
type AuditEvent struct {
    Timestamp    time.Time
    ProjectID    string
    TaskID       string
    AgentID      string
    EventType    AuditEventType
    Action       string
    Target       string           // Betroffene Ressource
    Input        string           // Eingabe (gekürzt)
    Output       string           // Ausgabe (gekürzt)
    Success      bool
    Error        string
    Duration     time.Duration
    SecurityFlags []string        // Sicherheitsrelevante Markierungen
}

type AuditEventType string

const (
    AuditEventAgentCreated    AuditEventType = "agent_created"
    AuditEventAgentDestroyed  AuditEventType = "agent_destroyed"
    AuditEventToolGenerated   AuditEventType = "tool_generated"
    AuditEventToolExecuted    AuditEventType = "tool_executed"
    AuditEventToolDestroyed   AuditEventType = "tool_destroyed"
    AuditEventFileAccess      AuditEventType = "file_access"
    AuditEventNetworkAccess   AuditEventType = "network_access"
    AuditEventSecurityBlock   AuditEventType = "security_block"
)
```

---

## 8. Implementierungs-Roadmap

### Phase 1: Foundation (Grundlagen)

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| Orchestrator Core | Basis-Koordinator mit Strategist | Hoch |
| Agent Pool | Verwaltung statischer Agenten | Hoch |
| Task Decomposition | Aufgaben-Zerlegung | Hoch |
| Basic Delegation | Einfache Agenten-Zuweisung | Hoch |

### Phase 2: Quality & Review

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| Evaluator Module | Ergebnis-Bewertung | Hoch |
| Review Chain | Mehrstufige Prüfung | Mittel |
| Iteration Loop | Verbesserungs-Schleife | Hoch |
| Result Aggregator | Ergebnis-Zusammenführung | Mittel |

### Phase 3: Dynamic Agents

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| Agent Factory | Dynamische Agenten-Erzeugung | Mittel |
| Blueprint System | Agent-Vorlagen | Mittel |
| Knowledge Injection | Wissens-Übertragung | Mittel |
| Agent Lifecycle | Lebenszyklus-Management | Niedrig |

### Phase 4: Tool Generation

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| Tool Factory | Tool-Generierung | Mittel |
| Security Checker | Sicherheitsprüfung | Hoch |
| Sandbox Execution | Isolierte Ausführung | Hoch |
| Tool Lifecycle | Lebenszyklus-Management | Niedrig |

### Phase 5: Advanced Features

| Komponente | Beschreibung | Priorität |
|------------|--------------|-----------|
| Parallel Execution | Parallele Task-Ausführung | Mittel |
| Cross-Project Learning | Projekt-übergreifendes Lernen | Niedrig |
| Performance Optimization | Performance-Optimierungen | Niedrig |
| Advanced Analytics | Erweiterte Metriken | Niedrig |

---

## 9. Integration mit bestehenden mDW-Komponenten

### 9.1 Leibniz (Agentic AI)

Der Orchestrator wird als Erweiterung von Leibniz implementiert:

```go
// internal/leibniz/orchestrator/orchestrator.go
package orchestrator

import (
    "github.com/msto63/mDW/internal/leibniz/agent"
    "github.com/msto63/mDW/internal/leibniz/platon"
)

// Orchestrator nutzt bestehende Agent-Infrastruktur
type Orchestrator struct {
    baseAgent    *agent.Agent        // Basis-Agent-Funktionalität
    platonClient *platon.Client      // Platon für Pre-/Post-Processing
    // ...
}
```

### 9.2 Turing (LLM Management)

Alle Agenten nutzen Turing für LLM-Aufrufe:

```go
// Agenten rufen LLM über Turing auf
func (a *DynamicAgent) Think(ctx context.Context, prompt string) (string, error) {
    return a.turingClient.Chat(ctx, &turingpb.ChatRequest{
        Model:        a.config.Model,
        SystemPrompt: a.systemPrompt,
        Messages:     a.buildMessages(prompt),
    })
}
```

### 9.3 Platon (Pipeline Processing)

Alle Agenten-Interaktionen werden durch Platon gefiltert:

```go
// Pre-Processing vor Agent-Ausführung
func (o *Orchestrator) preProcess(ctx context.Context, task *SubTask) (*SubTask, error) {
    resp, err := o.platonClient.ProcessPre(ctx, &platon.ProcessRequest{
        PipelineID: "orchestrator-pre",
        Prompt:     task.Description,
    })
    // ...
}

// Post-Processing nach Agent-Ausführung
func (o *Orchestrator) postProcess(ctx context.Context, result *TaskResult) (*TaskResult, error) {
    resp, err := o.platonClient.ProcessPost(ctx, &platon.ProcessRequest{
        PipelineID: "orchestrator-post",
        Response:   result.Output,
    })
    // ...
}
```

### 9.4 Hypatia (RAG Service)

Für Wissens-Injektion in dynamische Agenten:

```go
// Wissen aus RAG laden
func (f *AgentFactory) injectKnowledge(ctx context.Context, bp *AgentBlueprint) error {
    // Relevante Dokumente aus Hypatia abrufen
    results, err := f.hypatiaClient.Search(ctx, &hypatiapb.SearchRequest{
        Query:    bp.DomainKnowledge[0].Category,
        TopK:     10,
    })
    // ...
}
```

---

## 10. Metriken und Monitoring

### 10.1 Projekt-Metriken

```go
type ProjectMetrics struct {
    // Zeitmetriken
    TotalDuration      time.Duration
    PlanningDuration   time.Duration
    ExecutionDuration  time.Duration
    ReviewDuration     time.Duration

    // Task-Metriken
    TotalTasks         int
    CompletedTasks     int
    FailedTasks        int
    RetriedTasks       int

    // Agenten-Metriken
    StaticAgentsUsed   int
    DynamicAgentsCreated int
    TotalAgentInvocations int

    // Tool-Metriken
    ToolsGenerated     int
    ToolExecutions     int

    // Qualitäts-Metriken
    AverageQualityScore float64
    TotalIterations    int
    FirstPassRate      float64  // Anteil ohne Iteration akzeptiert

    // Kosten-Metriken (LLM-Tokens)
    TotalTokensUsed    int64
    PromptTokens       int64
    CompletionTokens   int64
}
```

### 10.2 Dashboard-Integration

Die Metriken werden im mDW Control Center visualisiert:

- Projekt-Übersicht mit Status
- Echtzeit-Task-Progress
- Agenten-Auslastung
- Qualitäts-Trends
- Token-Verbrauch

---

## 11. Glossar

| Begriff | Beschreibung |
|---------|--------------|
| **Orchestrator** | Koordinator-Agent, der Multi-Agenten-Projekte steuert |
| **Strategist** | Modul zur Aufgaben-Analyse und -Zerlegung |
| **Delegator** | Modul zur Agenten-Zuweisung |
| **Evaluator** | Modul zur Qualitätsbewertung |
| **Agent Factory** | Komponente zur dynamischen Agenten-Erzeugung |
| **Tool Factory** | Komponente zur dynamischen Tool-Generierung |
| **Blueprint** | Vorlage zur Erzeugung dynamischer Agenten/Tools |
| **Review Chain** | Mehrstufige Prüfkette für Ergebnisse |
| **Sandbox** | Isolierte Ausführungsumgebung |
| **Knowledge Injection** | Übertragung von Wissen an dynamische Agenten |

---

## 12. Offene Fragen und Entscheidungen

1. **Persistenz**: Sollen Projekt-States in einer Datenbank persistiert werden?
2. **Skalierung**: Wie viele parallele Projekte/Agenten maximal?
3. **Kosten-Kontrolle**: Wie Token-Budgets pro Projekt verwalten?
4. **UI**: Eigene TUI für Orchestrator oder Integration in bestehende?
5. **MCP-Integration**: Sollen dynamische Tools als MCP-Server bereitgestellt werden?

---

**Nächste Schritte:**

1. Review dieses Konzepts
2. Priorisierung der Implementierungs-Phasen
3. Detaillierte technische Spezifikation für Phase 1
4. Prototyp-Entwicklung
