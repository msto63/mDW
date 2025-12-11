# Intelligente Agent-Orchestrierung

## Konzept-Übersicht

Dieses Dokument beschreibt das Konzept für eine intelligente, RAG-ähnliche Agent-Orchestrierung im mDW-System.

## Architektur-Überblick

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              USER PROMPT                                     │
│                   "Recherchiere aktuelle KI-Trends und                       │
│                    fasse sie zusammen, dann übersetze ins Englische"         │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         PHASE 1: TASK DECOMPOSITION                          │
│                              (Aristoteles)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  LLM analysiert Prompt und zerlegt in atomare Aufgaben:                     │
│                                                                              │
│  Task 1: "Recherchiere aktuelle KI-Trends im Internet"                      │
│  Task 2: "Fasse die recherchierten Informationen zusammen"                  │
│  Task 3: "Übersetze die Zusammenfassung ins Englische"                      │
│                                                                              │
│  Abhängigkeiten: Task 2 benötigt Output von Task 1                          │
│                  Task 3 benötigt Output von Task 2                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PHASE 2: AGENT MATCHING (RAG-Style)                     │
│                              (Aristoteles)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Für jede Task:                                                              │
│                                                                              │
│  1. Task-Embedding erstellen (Hypatia/Babbage)                              │
│     Task 1 → [0.23, 0.87, 0.12, ...]                                        │
│                                                                              │
│  2. Vergleich mit Agent-Embeddings (Vektor-Similarity)                      │
│     ┌─────────────────────────────────────────────────────────────────┐     │
│     │ Agent-Registry (vorberechnet)                                   │     │
│     │                                                                 │     │
│     │ web-researcher:    [0.21, 0.89, 0.15, ...] → Similarity: 0.94  │     │
│     │ summarizer:        [0.45, 0.23, 0.67, ...] → Similarity: 0.31  │     │
│     │ translator:        [0.12, 0.34, 0.78, ...] → Similarity: 0.22  │     │
│     │ code-reviewer:     [0.78, 0.12, 0.45, ...] → Similarity: 0.08  │     │
│     └─────────────────────────────────────────────────────────────────┘     │
│                                                                              │
│  3. Bester Agent für Task 1: web-researcher (0.94)                          │
│                                                                              │
│  Ergebnis:                                                                   │
│  ┌──────────┬─────────────────────┬────────────────────┐                    │
│  │ Task     │ Beschreibung        │ Zugewiesener Agent │                    │
│  ├──────────┼─────────────────────┼────────────────────┤                    │
│  │ Task 1   │ KI-Trends recherch. │ web-researcher     │                    │
│  │ Task 2   │ Zusammenfassen      │ summarizer         │                    │
│  │ Task 3   │ Übersetzen          │ translator         │                    │
│  └──────────┴─────────────────────┴────────────────────┘                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       PHASE 3: PIPELINE EXECUTION                            │
│                              (Leibniz)                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ExecutionContext = {                                                        │
│      original_prompt: "Recherchiere aktuelle KI-Trends...",                 │
│      accumulated_output: "",                                                 │
│      current_task_index: 0,                                                  │
│      tasks: [Task1, Task2, Task3]                                           │
│  }                                                                           │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ TASK 1: web-researcher                                              │    │
│  │ Input:  "Recherchiere aktuelle KI-Trends im Internet"              │    │
│  │ Context: (leer - erste Task)                                        │    │
│  │ Output: "Die wichtigsten KI-Trends 2024/2025 sind:                 │    │
│  │          1. Multimodale KI-Modelle...                              │    │
│  │          2. AI Agents und Automatisierung...                       │    │
│  │          3. Edge AI..."                                             │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                              │                                               │
│                              ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ TASK 2: summarizer                                                  │    │
│  │ Input:  "Fasse die recherchierten Informationen zusammen"          │    │
│  │ Context: [Output von Task 1]                                        │    │
│  │ Output: "**KI-Trends 2025 - Zusammenfassung**                      │    │
│  │          Die drei dominanten Trends sind multimodale Modelle,      │    │
│  │          autonome AI-Agents und Edge-Computing..."                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                              │                                               │
│                              ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ TASK 3: translator                                                  │    │
│  │ Input:  "Übersetze die Zusammenfassung ins Englische"              │    │
│  │ Context: [Output von Task 2]                                        │    │
│  │ Output: "**AI Trends 2025 - Summary**                              │    │
│  │          The three dominant trends are multimodal models,          │    │
│  │          autonomous AI agents, and edge computing..."              │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            FINAL OUTPUT                                      │
│                                                                              │
│  **AI Trends 2025 - Summary**                                               │
│  The three dominant trends are multimodal models,                           │
│  autonomous AI agents, and edge computing...                                │
│                                                                              │
│  [Optional: Execution Trace für Transparenz]                                │
│  Pipeline: web-researcher → summarizer → translator                         │
│  Duration: 12.3s                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Komponenten-Design

### 1. Task Decomposer (Aristoteles)

```go
// TaskDecomposer zerlegt komplexe Prompts in atomare Tasks
type TaskDecomposer struct {
    llm      LLMClient
    logger   *logging.Logger
}

type DecomposedTask struct {
    ID           string   `json:"id"`
    Description  string   `json:"description"`
    Dependencies []string `json:"dependencies"` // IDs der vorherigen Tasks
    Priority     int      `json:"priority"`
}

type DecompositionResult struct {
    OriginalPrompt string           `json:"original_prompt"`
    Tasks          []DecomposedTask `json:"tasks"`
    IsSequential   bool             `json:"is_sequential"`
    CanParallelize [][]string       `json:"can_parallelize"` // Gruppen parallel ausführbarer Tasks
}

// Decompose nutzt ein LLM um den Prompt zu analysieren
func (d *TaskDecomposer) Decompose(ctx context.Context, prompt string) (*DecompositionResult, error) {
    // System-Prompt für Task-Zerlegung
    systemPrompt := `Analysiere den folgenden User-Prompt und zerlege ihn in einzelne,
    atomare Aufgaben. Identifiziere Abhängigkeiten zwischen den Aufgaben.

    Antworte im JSON-Format:
    {
        "tasks": [
            {"id": "task_1", "description": "...", "dependencies": []},
            {"id": "task_2", "description": "...", "dependencies": ["task_1"]}
        ],
        "is_sequential": true/false,
        "reasoning": "..."
    }`

    // LLM-Aufruf...
}
```

### 2. Agent Registry mit Embeddings (Leibniz)

```go
// AgentRegistry verwaltet Agents und ihre Embeddings
type AgentRegistry struct {
    agents     map[string]*AgentDefinition
    embeddings map[string][]float32  // Agent-ID → Embedding
    hypatia    HypatiaClient         // Für Embedding-Generierung
    logger     *logging.Logger
}

// RegisterAgent registriert einen Agent und berechnet sein Embedding
func (r *AgentRegistry) RegisterAgent(agent *AgentDefinition) error {
    // Kombiniere Name, Description und SystemPrompt für Embedding
    textForEmbedding := fmt.Sprintf(
        "Agent: %s\nBeschreibung: %s\nFähigkeiten: %s\nTools: %s",
        agent.Name,
        agent.Description,
        agent.SystemPrompt,
        strings.Join(agent.Tools, ", "),
    )

    // Embedding über Hypatia/Babbage generieren
    embedding, err := r.hypatia.GenerateEmbedding(ctx, textForEmbedding)
    if err != nil {
        return err
    }

    r.agents[agent.ID] = agent
    r.embeddings[agent.ID] = embedding
    return nil
}

// FindBestAgent findet den besten Agent für eine Task via Cosine Similarity
func (r *AgentRegistry) FindBestAgent(ctx context.Context, taskDescription string) (*AgentMatch, error) {
    // Task-Embedding generieren
    taskEmbedding, err := r.hypatia.GenerateEmbedding(ctx, taskDescription)
    if err != nil {
        return nil, err
    }

    // Cosine Similarity mit allen Agents berechnen
    var bestMatch *AgentMatch
    for agentID, agentEmbedding := range r.embeddings {
        similarity := cosineSimilarity(taskEmbedding, agentEmbedding)
        if bestMatch == nil || similarity > bestMatch.Similarity {
            bestMatch = &AgentMatch{
                AgentID:    agentID,
                Agent:      r.agents[agentID],
                Similarity: similarity,
            }
        }
    }

    return bestMatch, nil
}
```

### 3. Pipeline Executor (Leibniz)

```go
// PipelineExecutor führt eine Agent-Pipeline aus
type PipelineExecutor struct {
    registry *AgentRegistry
    executor *AgentExecutor
    logger   *logging.Logger
}

type PipelineContext struct {
    OriginalPrompt    string
    AccumulatedOutput string
    TaskResults       map[string]*TaskResult
    CurrentTaskIndex  int
}

type TaskResult struct {
    TaskID    string
    AgentID   string
    Input     string
    Output    string
    Duration  time.Duration
    Success   bool
    Error     string
}

// Execute führt die gesamte Pipeline aus
func (e *PipelineExecutor) Execute(ctx context.Context, plan *ExecutionPlan) (*PipelineResult, error) {
    pipelineCtx := &PipelineContext{
        OriginalPrompt: plan.OriginalPrompt,
        TaskResults:    make(map[string]*TaskResult),
    }

    for i, task := range plan.Tasks {
        pipelineCtx.CurrentTaskIndex = i

        // Kontext aus vorherigen Tasks zusammenstellen
        contextFromPrevious := e.buildContext(pipelineCtx, task.Dependencies)

        // Agent-Input zusammenbauen
        agentInput := fmt.Sprintf(
            "Aufgabe: %s\n\nKontext aus vorherigen Schritten:\n%s",
            task.Description,
            contextFromPrevious,
        )

        // Agent ausführen
        result, err := e.executor.Execute(ctx, task.AssignedAgentID, agentInput)
        if err != nil {
            return nil, fmt.Errorf("task %s failed: %w", task.ID, err)
        }

        // Ergebnis speichern
        pipelineCtx.TaskResults[task.ID] = result
        pipelineCtx.AccumulatedOutput = result.Output

        e.logger.Info("Task completed",
            "task", task.ID,
            "agent", task.AssignedAgentID,
            "duration", result.Duration)
    }

    return &PipelineResult{
        FinalOutput: pipelineCtx.AccumulatedOutput,
        TaskResults: pipelineCtx.TaskResults,
        Pipeline:    plan.GetAgentSequence(),
    }, nil
}

// buildContext sammelt Output von abhängigen Tasks
func (e *PipelineExecutor) buildContext(ctx *PipelineContext, dependencies []string) string {
    if len(dependencies) == 0 {
        return "(Keine vorherigen Ergebnisse)"
    }

    var parts []string
    for _, depID := range dependencies {
        if result, ok := ctx.TaskResults[depID]; ok {
            parts = append(parts, fmt.Sprintf("--- Ergebnis von %s ---\n%s", depID, result.Output))
        }
    }
    return strings.Join(parts, "\n\n")
}
```

### 4. Orchestrator (Aristoteles)

```go
// Orchestrator koordiniert den gesamten Flow
type Orchestrator struct {
    decomposer    *TaskDecomposer
    agentRegistry *AgentRegistry  // via Leibniz gRPC
    executor      *PipelineExecutor
    logger        *logging.Logger
}

// Process verarbeitet einen User-Prompt vollständig
func (o *Orchestrator) Process(ctx context.Context, prompt string) (*OrchestrationResult, error) {
    // Phase 1: Task Decomposition
    o.logger.Info("Phase 1: Decomposing prompt into tasks")
    decomposition, err := o.decomposer.Decompose(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("decomposition failed: %w", err)
    }

    // Einfacher Prompt? Direkt ausführen ohne Pipeline
    if len(decomposition.Tasks) == 1 {
        return o.executeSingleTask(ctx, decomposition.Tasks[0])
    }

    // Phase 2: Agent Matching für jede Task
    o.logger.Info("Phase 2: Matching agents to tasks", "task_count", len(decomposition.Tasks))
    plan := &ExecutionPlan{
        OriginalPrompt: prompt,
        Tasks:          make([]*PlannedTask, len(decomposition.Tasks)),
    }

    for i, task := range decomposition.Tasks {
        match, err := o.agentRegistry.FindBestAgent(ctx, task.Description)
        if err != nil {
            return nil, fmt.Errorf("agent matching failed for task %s: %w", task.ID, err)
        }

        plan.Tasks[i] = &PlannedTask{
            ID:              task.ID,
            Description:     task.Description,
            Dependencies:    task.Dependencies,
            AssignedAgentID: match.AgentID,
            MatchConfidence: match.Similarity,
        }

        o.logger.Info("Agent assigned",
            "task", task.ID,
            "agent", match.AgentID,
            "confidence", match.Similarity)
    }

    // Phase 3: Pipeline Execution
    o.logger.Info("Phase 3: Executing pipeline")
    result, err := o.executor.Execute(ctx, plan)
    if err != nil {
        return nil, fmt.Errorf("pipeline execution failed: %w", err)
    }

    return &OrchestrationResult{
        FinalOutput: result.FinalOutput,
        Plan:        plan,
        TaskResults: result.TaskResults,
    }, nil
}
```

## Datenfluss

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           DATENFLUSS                                      │
└──────────────────────────────────────────────────────────────────────────┘

1. USER PROMPT
   │
   ▼
2. ARISTOTELES: Task Decomposition
   ├─ Input: "Recherchiere X, fasse zusammen, übersetze"
   ├─ LLM analysiert Struktur
   └─ Output: [Task1, Task2, Task3] mit Dependencies
   │
   ▼
3. ARISTOTELES → HYPATIA: Embedding-Generierung
   ├─ Für jede Task: Task-Description → Embedding-Vektor
   └─ Output: [TaskEmbedding1, TaskEmbedding2, TaskEmbedding3]
   │
   ▼
4. ARISTOTELES → LEIBNIZ: Agent Discovery
   ├─ GetAgents() → Liste aller verfügbaren Agents
   └─ Jeder Agent hat vorberechnetes Embedding
   │
   ▼
5. ARISTOTELES: Vektor-Similarity-Matching
   ├─ Für jede Task: CosineSimilarity(TaskEmbedding, AgentEmbeddings)
   └─ Output: ExecutionPlan mit Agent-Zuweisungen
   │
   ▼
6. LEIBNIZ: Pipeline Execution
   │
   ├─ Task 1: web-researcher
   │  ├─ Input: Task-Description + (kein Kontext)
   │  ├─ Agent führt aus (nutzt Tools)
   │  └─ Output: Recherche-Ergebnis
   │  │
   │  ▼
   ├─ Task 2: summarizer
   │  ├─ Input: Task-Description + Kontext(Task1.Output)
   │  ├─ Agent führt aus
   │  └─ Output: Zusammenfassung
   │  │
   │  ▼
   └─ Task 3: translator
      ├─ Input: Task-Description + Kontext(Task2.Output)
      ├─ Agent führt aus
      └─ Output: Übersetzung (FINAL)
   │
   ▼
7. FINAL OUTPUT → USER
   └─ Übersetztes Ergebnis + optionale Execution-Trace
```

## Agent-Embedding-Strategie

### Was wird embedded?

Für optimales Matching kombinieren wir mehrere Agent-Attribute:

```go
func buildAgentEmbeddingText(agent *AgentDefinition) string {
    // Strukturierter Text für besseres Embedding
    return fmt.Sprintf(`
Agent-Name: %s
Beschreibung: %s
Spezialisierung: %s
Verfügbare Tools: %s
Typische Aufgaben: %s
Schlüsselwörter: %s
`,
        agent.Name,
        agent.Description,
        extractSpecialization(agent.SystemPrompt),
        strings.Join(agent.Tools, ", "),
        extractTypicalTasks(agent.SystemPrompt),
        strings.Join(agent.Metadata["tags"], ", "),
    )
}
```

### Beispiel Agent-Embeddings

| Agent | Embedding-Text (gekürzt) |
|-------|--------------------------|
| web-researcher | "Agent für Internet-Recherche, Suche, News, aktuelle Informationen. Tools: web_search, fetch_webpage" |
| summarizer | "Agent für Zusammenfassungen, Kernaussagen extrahieren, TLDR, Texte kürzen" |
| translator | "Agent für Übersetzungen, Lokalisierung, mehrsprachig, Deutsch, Englisch" |
| code-reviewer | "Agent für Code-Review, Qualität, Sicherheit, Best Practices, Bugs finden" |

## Parallele Ausführung (Optional)

Wenn Tasks keine Abhängigkeiten haben, können sie parallel ausgeführt werden:

```
Beispiel: "Recherchiere KI-Trends UND Blockchain-Trends, dann vergleiche beides"

Task-Graph:
    ┌─────────────────┐     ┌─────────────────┐
    │ Task 1: KI      │     │ Task 2: Block   │
    │ web-researcher  │     │ web-researcher  │
    └────────┬────────┘     └────────┬────────┘
             │                       │
             └───────────┬───────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Task 3: Vergleich   │
              │ data-analyst        │
              └─────────────────────┘

Execution:
- Task 1 und Task 2: PARALLEL (keine Dependencies)
- Task 3: SEQUENTIAL (wartet auf Task 1 + Task 2)
```

## Vorteile dieses Ansatzes

1. **Automatische Agent-Auswahl** - Kein manuelles Routing nötig
2. **Dynamisch erweiterbar** - Neue Agents werden automatisch eingebunden
3. **Kontextuelle Verkettung** - Jeder Agent erhält relevanten Kontext
4. **Skalierbar** - Parallele Ausführung möglich
5. **Transparent** - Execution-Trace zeigt Pipeline-Ablauf
6. **RAG-kompatibel** - Nutzt bestehende Embedding-Infrastruktur (Hypatia)

## Implementierungs-Reihenfolge

1. **Phase 1: Agent-Embeddings** (Leibniz + Hypatia)
   - Agent-Registry um Embeddings erweitern
   - Embeddings bei Agent-Load generieren
   - API für Agent-Discovery mit Embeddings

2. **Phase 2: Task Decomposer** (Aristoteles)
   - LLM-basierte Prompt-Zerlegung
   - Dependency-Graph-Erstellung
   - Integration in Intent-Analyse

3. **Phase 3: Agent Matcher** (Aristoteles)
   - Vektor-Similarity-Berechnung
   - Threshold für Mindest-Confidence
   - Fallback zu Default-Agent

4. **Phase 4: Pipeline Executor** (Leibniz)
   - Sequentielle Ausführung mit Kontext-Passing
   - Fehlerbehandlung und Retry
   - Streaming-Updates für UI

5. **Phase 5: UI-Integration** (ChatClient)
   - Pipeline-Visualisierung
   - Task-Fortschritt anzeigen
   - Execution-Trace optional anzeigen

## Offene Fragen

1. **Confidence-Threshold**: Ab welcher Similarity wird ein Agent gewählt vs. Fallback?
2. **Fehlerbehandlung**: Was passiert, wenn ein Agent in der Pipeline fehlschlägt?
3. **Token-Budget**: Wie viel Kontext kann/soll an nachfolgende Agents übergeben werden?
4. **User-Override**: Soll der User die Agent-Auswahl überschreiben können?
5. **Caching**: Sollen Agent-Embeddings gecacht werden (wahrscheinlich ja)?
