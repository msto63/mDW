# Konzept: Agent Builder TUI

**Projektname:** mDW Agent Builder
**Version:** 1.0
**Autor:** Mike Stoffels mit Claude
**Datum:** 2025-12-07
**Status:** Konzept

---

## 1. Zusammenfassung

Der **Agent Builder** ist eine Terminal-basierte Benutzeroberfläche (TUI) zur Erstellung, Verwaltung und Test von KI-Agenten im Leibniz-Service. Die Anwendung ermöglicht es Benutzern, ohne Programmierkenntnisse eigene Agenten zu konfigurieren und deren Verhalten interaktiv zu testen.

---

## 2. Ziele

| Ziel | Beschreibung |
|------|--------------|
| **Benutzerfreundlichkeit** | Intuitive TUI für die Agent-Erstellung ohne Code |
| **Vollständigkeit** | Alle Agent-Parameter konfigurierbar |
| **Verwaltung** | CRUD-Operationen für bestehende Agenten |
| **Testing** | Integrierte Test-Umgebung für Agenten |
| **Konsistenz** | Design konsistent mit ChatClient und LogViewer |

---

## 3. Agent-Parameter

### 3.1 Basis-Parameter

| Parameter | Typ | Beschreibung | Default | Validierung |
|-----------|-----|--------------|---------|-------------|
| `name` | string | Eindeutiger Name des Agenten | - | Pflicht, max. 64 Zeichen |
| `description` | string | Beschreibung des Agenten | "" | Max. 500 Zeichen |
| `system_prompt` | string | System-Prompt für das LLM | Template | Pflicht, max. 10.000 Zeichen |

### 3.2 Konfigurations-Parameter

| Parameter | Typ | Beschreibung | Default | Bereich |
|-----------|-----|--------------|---------|---------|
| `model` | string | LLM-Modell | "llama3.2:3b" | Aus Turing-Modell-Liste |
| `temperature` | float | Kreativitäts-Parameter | 0.7 | 0.0 - 2.0 |
| `max_iterations` | int | Maximale Iterationen | 10 | 1 - 50 |
| `timeout_seconds` | int | Timeout in Sekunden | 120 | 10 - 600 |
| `streaming_enabled` | bool | Streaming-Ausgabe | true | true/false |

### 3.3 Wissens-Parameter

| Parameter | Typ | Beschreibung | Default |
|-----------|-----|--------------|---------|
| `use_knowledge_base` | bool | RAG aktivieren | false |
| `knowledge_collection` | string | Hypatia-Collection | "" |

### 3.4 Tools

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `tools` | []string | Liste aktivierter Tools |

**Verfügbare Tool-Kategorien:**
- **Builtin Tools**: Basis-Tools wie `calculator`, `datetime`, `web_search`
- **MCP Tools**: Tools aus MCP-Servern (Model Context Protocol)
- **Custom Tools**: Benutzerdefinierte Tools

### 3.5 Metadaten

| Parameter | Typ | Beschreibung |
|-----------|-----|--------------|
| `metadata` | map[string]string | Zusätzliche Key-Value-Paare |
| `created_at` | timestamp | Erstellungszeitpunkt (auto) |
| `updated_at` | timestamp | Aktualisierungszeitpunkt (auto) |

---

## 4. Anwendungsstruktur

### 4.1 Hauptansichten

```
┌─────────────────────────────────────────────────────────────────┐
│  mDW Agent Builder                                    [Leibniz] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐  ┌────────────────────────────────────────┐  │
│  │ Agent-Liste  │  │ Detail-Bereich                         │  │
│  │              │  │                                        │  │
│  │ > Assistent  │  │  Name: Assistent                       │  │
│  │   Coder      │  │  Beschreibung: Hilft bei Aufgaben      │  │
│  │   Recherche  │  │  Modell: llama3.2:3b                   │  │
│  │   Übersetzer │  │  Temperature: 0.7                      │  │
│  │              │  │  Max Iterations: 10                    │  │
│  │ [+ Neu]      │  │  Tools: calculator, web_search         │  │
│  │              │  │                                        │  │
│  └──────────────┘  └────────────────────────────────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [N]eu [E]dit [D]el [T]est [C]lone │ Tab: Wechseln │ Q: Beenden │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Editor-Ansicht

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent Editor - Neuer Agent                           [Leibniz] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Basis-Informationen                                            │
│  ─────────────────────────────────────────────────────────────  │
│  Name:         [________________________]                       │
│  Beschreibung: [________________________]                       │
│                                                                 │
│  LLM-Konfiguration                                              │
│  ─────────────────────────────────────────────────────────────  │
│  Modell:       [llama3.2:3b        ▼]                          │
│  Temperature:  [====○─────────] 0.7                             │
│  Max Steps:    [=====○────────] 10                              │
│  Timeout (s):  [======○───────] 120                             │
│                                                                 │
│  System-Prompt                                                  │
│  ─────────────────────────────────────────────────────────────  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Du bist ein hilfreicher KI-Assistent...                   │  │
│  │ ...                                                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Tab: Nächstes Feld │ Ctrl+S: Speichern │ Esc: Abbrechen        │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Tool-Auswahl

```
┌─────────────────────────────────────────────────────────────────┐
│  Tool-Auswahl                                         [Leibniz] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Builtin Tools                                                  │
│  ─────────────────────────────────────────────────────────────  │
│  [x] calculator      - Mathematische Berechnungen               │
│  [x] datetime        - Datum und Uhrzeit                        │
│  [ ] web_search      - Web-Suche                                │
│  [ ] file_read       - Dateien lesen                            │
│  [ ] file_write      - Dateien schreiben                        │
│                                                                 │
│  MCP Tools (3 Server verbunden)                                 │
│  ─────────────────────────────────────────────────────────────  │
│  [ ] mcp_filesystem  - Dateisystem-Zugriff                      │
│  [ ] mcp_github      - GitHub-Integration                       │
│  [ ] mcp_database    - Datenbank-Abfragen                       │
│                                                                 │
│  Custom Tools                                                   │
│  ─────────────────────────────────────────────────────────────  │
│  [ ] custom_api      - Benutzerdefinierte API                   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Space: Togglen │ A: Alle │ N: Keine │ Enter: Bestätigen         │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Agent-Testing

### 5.1 Test-Modi

| Modus | Beschreibung | Anwendungsfall |
|-------|--------------|----------------|
| **Quick Test** | Einzelne Nachricht, schnelle Antwort | Schnellprüfung |
| **Interactive Chat** | Konversations-Modus | Verhaltenstest |
| **Batch Test** | Mehrere vordefinierte Prompts | Regression |
| **Benchmark** | Performance-Messung | Optimierung |

### 5.2 Test-Ansicht

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent Test - "Assistent"                             [Leibniz] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Test-Konversation                                       │    │
│  │                                                         │    │
│  │ USER: Was ist 25 * 17?                                  │    │
│  │                                                         │    │
│  │ AGENT [Thinking]:                                       │    │
│  │ > Ich soll 25 * 17 berechnen. Dafür nutze ich den      │    │
│  │   Calculator.                                           │    │
│  │                                                         │    │
│  │ AGENT [Tool Call]: calculator                           │    │
│  │ > Input: {"operation": "multiply", "a": 25, "b": 17}   │    │
│  │ > Output: 425                                           │    │
│  │                                                         │    │
│  │ AGENT [Response]:                                       │    │
│  │ > 25 * 17 = 425                                         │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Statistiken          │ Iterations: 2  │ Tokens: 156    │    │
│  │ Duration: 1.2s       │ Tools: 1       │ Status: OK     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  > Test-Eingabe: [____________________________________] [Send]  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Enter: Senden │ Ctrl+B: Batch │ Ctrl+R: Reset │ Esc: Zurück    │
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 Batch-Testing

```
┌─────────────────────────────────────────────────────────────────┐
│  Batch Test - "Assistent"                             [Leibniz] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Test-Suite: Standard-Tests                                     │
│  ─────────────────────────────────────────────────────────────  │
│                                                                 │
│  #  │ Prompt                    │ Status    │ Duration │ Tokens │
│  ───┼───────────────────────────┼───────────┼──────────┼────────│
│  1  │ "Was ist 2+2?"            │ PASS      │ 0.8s     │ 45     │
│  2  │ "Welches Datum ist heute?"│ PASS      │ 1.1s     │ 62     │
│  3  │ "Erkläre Quantenphysik"   │ PASS      │ 3.2s     │ 312    │
│  4  │ "Schreibe ein Gedicht"    │ RUNNING   │ ...      │ ...    │
│  5  │ "Übersetze: Hello World"  │ PENDING   │ -        │ -      │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│  Fortschritt: [████████░░░░░░░░░░░░] 60% (3/5)                  │
│  Gesamt-Zeit: 5.1s │ Durchschnitt: 1.7s │ Fehler: 0            │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Space: Pause │ R: Neustart │ E: Export │ Esc: Abbrechen        │
└─────────────────────────────────────────────────────────────────┘
```

### 5.4 Test-Validierung

**Automatische Validierungen:**
- Response vorhanden (nicht leer)
- Timeout nicht überschritten
- Keine Fehler-Status
- Tool-Aufrufe erfolgreich

**Optionale Validierungen:**
- Erwartete Schlüsselwörter in Antwort
- Maximale Token-Anzahl
- Bestimmte Tools verwendet/nicht verwendet
- Antwort-Format (JSON, Markdown, etc.)

---

## 6. Vorlagen-System

### 6.1 Standard-Vorlagen

| Vorlage | Beschreibung | Tools |
|---------|--------------|-------|
| **Allgemein** | Hilfreicher Assistent | calculator, datetime |
| **Coder** | Programmier-Assistent | file_read, file_write |
| **Recherche** | Web-Recherche Agent | web_search |
| **Übersetzer** | Mehrsprachiger Übersetzer | - |
| **Analyst** | Daten-Analyse | calculator, database |

### 6.2 Vorlage speichern als

Benutzer können eigene Agenten als Vorlagen speichern:
- Name der Vorlage
- Kategorie
- Beschreibung
- Alle Parameter werden übernommen

---

## 7. Import/Export

### 7.1 Export-Formate

| Format | Beschreibung | Anwendung |
|--------|--------------|-----------|
| **JSON** | Vollständige Agent-Definition | Backup, Sharing |
| **YAML** | Lesbare Konfiguration | Versionierung |
| **Markdown** | Dokumentation | README |

### 7.2 Export-Beispiel (JSON)

```json
{
  "id": "agent-123",
  "name": "Assistent",
  "description": "Hilfreicher KI-Assistent",
  "system_prompt": "Du bist ein hilfreicher KI-Assistent...",
  "tools": ["calculator", "datetime"],
  "config": {
    "model": "llama3.2:3b",
    "temperature": 0.7,
    "max_iterations": 10,
    "timeout_seconds": 120,
    "streaming_enabled": true,
    "use_knowledge_base": false
  },
  "metadata": {
    "author": "user",
    "version": "1.0"
  },
  "created_at": "2025-12-07T12:00:00Z",
  "updated_at": "2025-12-07T12:00:00Z"
}
```

---

## 8. Technische Architektur

### 8.1 Package-Struktur

```
internal/tui/agentbuilder/
├── model.go          # Bubble Tea Model
├── view.go           # View-Rendering
├── update.go         # Update-Logik
├── messages.go       # Message-Typen
├── styles.go         # Lipgloss-Styles
├── components/
│   ├── list.go       # Agent-Liste
│   ├── editor.go     # Editor-Komponente
│   ├── toolpicker.go # Tool-Auswahl
│   ├── tester.go     # Test-Interface
│   └── slider.go     # Parameter-Slider
└── templates/
    └── templates.go  # Standard-Vorlagen
```

### 8.2 Abhängigkeiten

```go
import (
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/lipgloss"

    pb "github.com/msto63/mDW/api/gen/leibniz"
)
```

### 8.3 gRPC-Integration

```go
// Leibniz Service Calls
client.CreateAgent(ctx, &pb.CreateAgentRequest{...})
client.UpdateAgent(ctx, &pb.UpdateAgentRequest{...})
client.DeleteAgent(ctx, &pb.DeleteAgentRequest{...})
client.GetAgent(ctx, &pb.GetAgentRequest{...})
client.ListAgents(ctx, &pb.Empty{})
client.Execute(ctx, &pb.ExecuteRequest{...})
client.StreamExecute(ctx, &pb.ExecuteRequest{...})
client.ListTools(ctx, &pb.Empty{})
```

---

## 9. Tastenkürzel

### 9.1 Globale Tastenkürzel

| Taste | Aktion |
|-------|--------|
| `Tab` | Zwischen Panels wechseln |
| `Shift+Tab` | Rückwärts wechseln |
| `Ctrl+S` | Speichern |
| `Ctrl+N` | Neuer Agent |
| `Ctrl+T` | Test starten |
| `Ctrl+Q` / `Q` | Beenden |
| `?` / `F1` | Hilfe anzeigen |

### 9.2 Listen-Ansicht

| Taste | Aktion |
|-------|--------|
| `j` / `↓` | Nach unten |
| `k` / `↑` | Nach oben |
| `Enter` | Agent bearbeiten |
| `n` | Neuer Agent |
| `d` | Agent löschen |
| `c` | Agent klonen |
| `t` | Agent testen |
| `/` | Suchen |

### 9.3 Editor-Ansicht

| Taste | Aktion |
|-------|--------|
| `Tab` | Nächstes Feld |
| `Shift+Tab` | Vorheriges Feld |
| `Ctrl+S` | Speichern |
| `Esc` | Abbrechen |
| `Ctrl+E` | System-Prompt erweitern |

### 9.4 Test-Ansicht

| Taste | Aktion |
|-------|--------|
| `Enter` | Nachricht senden |
| `Ctrl+R` | Konversation zurücksetzen |
| `Ctrl+B` | Batch-Test starten |
| `Ctrl+X` | Ausführung abbrechen |
| `Esc` | Zurück zum Editor |

---

## 10. CLI-Integration

### 10.1 Kommando

```bash
mdw agents                    # Agent Builder starten
mdw agents --leibniz-addr localhost:9140  # Custom Leibniz-Adresse
```

### 10.2 Start-Script

```bash
#!/bin/bash
# StartAgentBuilder
cd "$(dirname "$0")" || exit 1
if [ ! -f "./bin/mdw" ]; then
    make build || exit 1
fi
exec ./bin/mdw agents "$@"
```

---

## 11. Erweiterungsmöglichkeiten

### 11.1 Zukünftige Features

| Feature | Beschreibung | Priorität |
|---------|--------------|-----------|
| **Agent Sharing** | Agenten über URL teilen | Mittel |
| **Version History** | Änderungsverlauf pro Agent | Niedrig |
| **Collaborative Editing** | Mehrere Benutzer gleichzeitig | Niedrig |
| **Performance Dashboard** | Langzeit-Statistiken | Mittel |
| **A/B Testing** | Zwei Agent-Versionen vergleichen | Niedrig |
| **Prompt Engineering** | KI-gestützte Prompt-Optimierung | Hoch |

### 11.2 Integration mit anderen TUIs

- **ChatClient**: Agent direkt im Chat verwenden
- **LogViewer**: Agent-Execution-Logs anzeigen
- **ControlCenter**: Agent-Service-Status überwachen

---

## 12. Fazit

Der Agent Builder TUI ermöglicht eine intuitive Erstellung und Verwaltung von KI-Agenten ohne Programmierkenntnisse. Durch die Integration von Testing-Funktionen können Benutzer ihre Agenten iterativ verbessern und deren Verhalten validieren. Das konsistente Design mit den anderen mDW-TUIs sorgt für eine einheitliche Benutzererfahrung.

---

**Nächste Schritte:**
1. Review des Konzepts
2. UI-Mockups erstellen
3. Package-Struktur aufsetzen
4. Basis-Implementierung
5. Testing-Framework
6. Integration und Tests
