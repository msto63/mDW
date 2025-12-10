# Filesystem Data Source - Konzept

## Übersicht

Dieses Konzept beschreibt die Implementierung einer Dateisystem-basierten Datenquelle für das mDW RAG-System. Benutzer können beliebig viele Datenquellen konfigurieren, wobei jede Datenquelle mehrere Verzeichnisse umfassen kann. Jede Datenquelle wird automatisch einer eigenen Collection zugeordnet, was gezielte RAG-Abfragen ermöglicht.

---

## Architektur-Entscheidungen

### Implementierung: In Hypatia als Subsystem

Die Datenquellen-Verwaltung wird **in Hypatia integriert**, nicht als separater Service.

| Aspekt | Entscheidung | Begründung |
|--------|--------------|------------|
| **Ort** | Hypatia (RAG Service) | Enge Kopplung mit Ingestion-Pipeline |
| **Struktur** | Subsystem unter `internal/hypatia/datasource/` | Klare Modularisierung |
| **Kommunikation** | Direkte Funktionsaufrufe | Kein gRPC-Overhead |
| **Deployment** | Teil von Hypatia | Ein Service weniger zu verwalten |

**Paketstruktur in Hypatia:**

```
internal/hypatia/
├── server/              # gRPC/HTTP Server (existiert)
├── service/             # RAG Business Logic (existiert)
├── vectorstore/         # Qdrant Integration (existiert)
├── datasource/          # NEU: Datenquellen-Subsystem
│   ├── manager.go       # DataSourceManager (orchestriert alles)
│   ├── types.go         # Interfaces & Types
│   ├── filesystem/      # Filesystem-Implementierung
│   │   ├── source.go    # FilesystemSource
│   │   ├── watcher.go   # fsnotify Wrapper
│   │   └── scanner.go   # Directory Scanner
│   └── state/           # Persistenz
│       └── sqlite.go    # SQLite Repository
└── parser/              # Dokument-Parser (existiert/erweitern)
```

### Geschätzter Umfang

| Komponente | Zeilen (ca.) | Komplexität |
|------------|--------------|-------------|
| **Interfaces & Types** | 200-300 | Niedrig |
| **DataSourceManager** | 300-400 | Mittel |
| **FilesystemSource** | 500-700 | Mittel |
| **FileWatcher (fsnotify)** | 300-400 | Mittel |
| **Scanner** | 200-300 | Niedrig |
| **SQLite State** | 400-500 | Mittel |
| **Parser-Erweiterungen** | 300-500 | Mittel |
| **gRPC API** | 200-300 | Niedrig |
| **REST API** | 200-300 | Niedrig |
| **Tests** | 1000-1500 | - |
| **Gesamt** | **~4000-5000** | **Mittel** |

### Speicherort der Konfiguration

Datenquellen-Konfiguration und Datei-Status werden in **SQLite** gespeichert:

```
~/.mdw/
├── config.toml              # Globale Konfiguration (optional)
└── hypatia/
    └── datasources.db       # SQLite Datenbank
```

**Warum SQLite?**

| Alternative | Problem |
|-------------|---------|
| config.toml | Nicht für dynamische Daten geeignet |
| JSON-Datei | Keine Transaktionen, Race Conditions |
| Qdrant | Falscher Zweck (Vektoren, nicht Config) |
| PostgreSQL | Overkill, externe Abhängigkeit |

**SQLite Vorteile:**
- Eingebettet, keine externe Abhängigkeit
- ACID-Transaktionen
- Performant für diese Datenmenge
- Einfache Backup/Restore
- Cross-Platform

**Datenbank-Schema:**

```sql
-- Datenquellen
CREATE TABLE data_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    collection_name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'filesystem',
    config JSON NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Überwachte Pfade pro Datenquelle
CREATE TABLE source_paths (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    recursive BOOLEAN DEFAULT TRUE,
    include_patterns JSON,  -- ["*.md", "*.pdf"]
    exclude_patterns JSON,  -- ["*.tmp", ".git/*"]
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, path)
);

-- Datei-Status (für Change Detection)
CREATE TABLE file_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER NOT NULL,
    modified_at TIMESTAMP NOT NULL,
    indexed_at TIMESTAMP NOT NULL,
    document_id TEXT,  -- Referenz in Qdrant
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    UNIQUE(source_id, path)
);

-- Sync-Historie
CREATE TABLE sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    files_added INTEGER DEFAULT 0,
    files_updated INTEGER DEFAULT 0,
    files_deleted INTEGER DEFAULT 0,
    errors JSON,
    status TEXT NOT NULL  -- 'running', 'completed', 'failed'
);

-- Indices für Performance
CREATE INDEX idx_file_states_source ON file_states(source_id);
CREATE INDEX idx_file_states_status ON file_states(status);
CREATE INDEX idx_file_states_hash ON file_states(hash);
CREATE INDEX idx_sync_history_source ON sync_history(source_id);
```

---

## Kernprinzip: 1 Datenquelle = 1 Collection

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Datenquellen-Hierarchie                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ DataSource: "Projekt Alpha"                                  │   │
│  │ ├── Verzeichnis: /home/user/projects/alpha/docs             │   │
│  │ ├── Verzeichnis: /home/user/projects/alpha/specs            │   │
│  │ └── → Collection: "projekt-alpha" (automatisch)             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ DataSource: "Persönliche Notizen"                            │   │
│  │ ├── Verzeichnis: /home/user/Notes                           │   │
│  │ └── → Collection: "persoenliche-notizen" (automatisch)      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ DataSource: "Technische Dokumentation"                       │   │
│  │ ├── Verzeichnis: /opt/docs/manuals                          │   │
│  │ ├── Verzeichnis: /opt/docs/references                       │   │
│  │ ├── Verzeichnis: /opt/docs/tutorials                        │   │
│  │ └── → Collection: "technische-dokumentation" (automatisch)  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

Bei RAG-Abfragen:
┌─────────────────────────────────────────────────────────────────────┐
│  "Wie konfiguriere ich den Server?"                                 │
│                                                                      │
│  Suche in:                                                          │
│  (•) Alle Datenquellen                                              │
│  ( ) Nur ausgewählte:                                               │
│      [x] Projekt Alpha                                              │
│      [ ] Persönliche Notizen                                        │
│      [x] Technische Dokumentation                                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Vorteile dieses Modells

| Vorteil | Beschreibung |
|---------|--------------|
| **Kontextuelle Suche** | Gezielte Suche in relevanten Bereichen |
| **Performance** | Kleinerer Suchraum = schnellere Antworten |
| **Datentrennung** | Privat vs. Geschäftlich vs. Projekt-spezifisch |
| **Granulare Verwaltung** | Einzelne Quellen pausieren/löschen/synchronisieren |
| **Logische Gruppierung** | Zusammengehörige Verzeichnisse in einer Quelle |
| **Flexibilität** | Multi-Collection-Suche bei Bedarf |

---

## Architektur

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              mDW Platform                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Hypatia (RAG Service)                          │   │
│  │  ┌─────────────────────────────────────────────────────────────┐ │   │
│  │  │                   DataSource Manager                         │ │   │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐            │ │   │
│  │  │  │ Filesystem  │ │   S3/Minio  │ │   WebDAV    │  ...       │ │   │
│  │  │  │   Source    │ │   Source    │ │   Source    │            │ │   │
│  │  │  └──────┬──────┘ └─────────────┘ └─────────────┘            │ │   │
│  │  └─────────┼───────────────────────────────────────────────────┘ │   │
│  │            │                                                      │   │
│  │  ┌─────────▼───────────────────────────────────────────────────┐ │   │
│  │  │                    Sync Engine                               │ │   │
│  │  │  ┌───────────┐  ┌───────────┐  ┌───────────┐               │ │   │
│  │  │  │  Watcher  │  │  Scanner  │  │  Differ   │               │ │   │
│  │  │  └───────────┘  └───────────┘  └───────────┘               │ │   │
│  │  └─────────────────────────────────────────────────────────────┘ │   │
│  │                              │                                    │   │
│  │  ┌───────────────────────────▼─────────────────────────────────┐ │   │
│  │  │                Document Processor                            │ │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐   │ │   │
│  │  │  │  PDF   │ │  DOCX  │ │   MD   │ │  TXT   │ │  HTML  │   │ │   │
│  │  │  │ Parser │ │ Parser │ │ Parser │ │ Parser │ │ Parser │   │ │   │
│  │  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘   │ │   │
│  │  └─────────────────────────────────────────────────────────────┘ │   │
│  │                              │                                    │   │
│  │  ┌───────────────────────────▼─────────────────────────────────┐ │   │
│  │  │              Ingest Pipeline (bestehend)                     │ │   │
│  │  │  Chunking → Embedding → Vector Storage                       │ │   │
│  │  └─────────────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Komponenten

### 1. DataSource Interface

Alle Datenquellen implementieren ein gemeinsames Interface:

```go
// DataSource definiert eine Datenquelle für Dokumente
type DataSource interface {
    // ID gibt die eindeutige Kennung der Datenquelle zurück
    ID() string

    // Type gibt den Typ der Datenquelle zurück (filesystem, s3, webdav, etc.)
    Type() string

    // Name gibt den benutzerdefinierten Namen zurück
    Name() string

    // Status gibt den aktuellen Status der Datenquelle zurück
    Status() DataSourceStatus

    // Scan führt einen vollständigen Scan durch
    Scan(ctx context.Context) (*ScanResult, error)

    // Watch startet die Echtzeit-Überwachung
    Watch(ctx context.Context, handler ChangeHandler) error

    // StopWatch beendet die Überwachung
    StopWatch() error

    // GetFile lädt eine einzelne Datei
    GetFile(ctx context.Context, path string) (*FileContent, error)

    // Close schließt die Datenquelle
    Close() error
}

type DataSourceStatus string

const (
    DataSourceStatusActive   DataSourceStatus = "active"
    DataSourceStatusPaused   DataSourceStatus = "paused"
    DataSourceStatusScanning DataSourceStatus = "scanning"
    DataSourceStatusError    DataSourceStatus = "error"
)

type ChangeHandler func(event ChangeEvent)

type ChangeEvent struct {
    Type      ChangeType
    Path      string
    OldPath   string  // Für Umbenennungen
    Timestamp time.Time
    FileInfo  FileInfo
}

type ChangeType string

const (
    ChangeTypeCreated  ChangeType = "created"
    ChangeTypeModified ChangeType = "modified"
    ChangeTypeDeleted  ChangeType = "deleted"
    ChangeTypeRenamed  ChangeType = "renamed"
)
```

### 2. Filesystem DataSource

```go
// FilesystemConfig konfiguriert eine Dateisystem-Datenquelle
type FilesystemConfig struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Paths       []string          `json:"paths"`           // Überwachte Verzeichnisse
    Recursive   bool              `json:"recursive"`       // Unterverzeichnisse einschließen
    Extensions  []string          `json:"extensions"`      // Erlaubte Dateiendungen (leer = alle)
    Exclude     []string          `json:"exclude"`         // Ausgeschlossene Muster (glob)
    MaxFileSize int64             `json:"max_file_size"`   // Max. Dateigröße in Bytes
    Collection  string            `json:"collection"`      // Ziel-Collection
    Metadata    map[string]string `json:"metadata"`        // Zusätzliche Metadaten
    SyncMode    SyncMode          `json:"sync_mode"`       // watch, poll, manual
    PollInterval time.Duration    `json:"poll_interval"`   // Für Poll-Modus
}

type SyncMode string

const (
    SyncModeWatch  SyncMode = "watch"   // Echtzeit via fsnotify
    SyncModePoll   SyncMode = "poll"    // Periodisches Polling
    SyncModeManual SyncMode = "manual"  // Nur manuell
)
```

### 3. File State Tracking

Zur Erkennung von Änderungen wird der Zustand aller Dateien in einer lokalen Datenbank gespeichert:

```go
// FileState speichert den Zustand einer Datei
type FileState struct {
    ID           string    `db:"id"`            // UUID
    DataSourceID string    `db:"datasource_id"` // Zugehörige Datenquelle
    Path         string    `db:"path"`          // Relativer Pfad
    AbsolutePath string    `db:"absolute_path"` // Absoluter Pfad
    Hash         string    `db:"hash"`          // SHA-256 des Inhalts
    Size         int64     `db:"size"`          // Dateigröße
    ModTime      time.Time `db:"mod_time"`      // Änderungszeit
    DocumentID   string    `db:"document_id"`   // Referenz zum Dokument in Hypatia
    Status       FileStatus `db:"status"`       // indexed, pending, error
    LastSynced   time.Time `db:"last_synced"`   // Letzte Synchronisation
    ErrorMessage string    `db:"error_message"` // Fehlermeldung bei Status=error
    CreatedAt    time.Time `db:"created_at"`
    UpdatedAt    time.Time `db:"updated_at"`
}

type FileStatus string

const (
    FileStatusPending  FileStatus = "pending"   // Noch nicht verarbeitet
    FileStatusIndexed  FileStatus = "indexed"   // Erfolgreich indiziert
    FileStatusError    FileStatus = "error"     // Fehler bei Verarbeitung
    FileStatusDeleted  FileStatus = "deleted"   // Datei wurde gelöscht
)
```

---

## Synchronisations-Logik

### Initiales Scannen

```
┌────────────────────────────────────────────────────────────────────────┐
│                        Initial Scan Workflow                            │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   1. Verzeichnis hinzufügen                                            │
│          │                                                              │
│          ▼                                                              │
│   2. Rekursives Scannen aller Dateien                                  │
│          │                                                              │
│          ▼                                                              │
│   3. Filter anwenden (Extensions, Exclude, MaxSize)                    │
│          │                                                              │
│          ▼                                                              │
│   4. Hash berechnen für jede Datei                                     │
│          │                                                              │
│          ▼                                                              │
│   5. FileState in DB speichern (Status: pending)                       │
│          │                                                              │
│          ▼                                                              │
│   6. Dateien zur Ingest-Queue hinzufügen                               │
│          │                                                              │
│          ▼                                                              │
│   7. Async Processing: Parse → Chunk → Embed → Store                   │
│          │                                                              │
│          ▼                                                              │
│   8. FileState aktualisieren (Status: indexed, DocumentID setzen)      │
│                                                                         │
└────────────────────────────────────────────────────────────────────────┘
```

### Änderungserkennung

```go
// SyncResult enthält das Ergebnis einer Synchronisation
type SyncResult struct {
    DataSourceID string
    StartTime    time.Time
    EndTime      time.Time

    // Statistiken
    FilesScanned   int
    FilesAdded     int
    FilesModified  int
    FilesDeleted   int
    FilesUnchanged int
    FilesErrored   int

    // Details
    Changes []ChangeDetail
    Errors  []SyncError
}

type ChangeDetail struct {
    Path       string
    ChangeType ChangeType
    OldHash    string
    NewHash    string
    DocumentID string
}
```

### Synchronisations-Algorithmus

```
Für jede Datei im Verzeichnis:
  1. Existiert FileState in DB?
     │
     ├─ NEIN → Neue Datei
     │         → FileState erstellen
     │         → Zur Ingest-Queue hinzufügen
     │
     └─ JA → Prüfe Änderung
             │
             ├─ ModTime oder Size geändert?
             │   │
             │   ├─ JA → Hash berechnen
             │   │       │
             │   │       ├─ Hash unterschiedlich?
             │   │       │   │
             │   │       │   ├─ JA → Datei geändert
             │   │       │   │       → Altes Dokument löschen
             │   │       │   │       → Neu ingestieren
             │   │       │   │
             │   │       │   └─ NEIN → Nur Metadaten geändert
             │   │       │             → FileState aktualisieren
             │   │       │
             │   │       └─ [Optimierung: Bei kleinen Dateien]
             │   │
             │   └─ NEIN → Unverändert
             │
             └─ [weiter]

Für jeden FileState in DB ohne entsprechende Datei:
  → Datei wurde gelöscht
  → Dokument aus Hypatia löschen
  → FileState als deleted markieren oder löschen
```

---

## Cross-Platform-Unterstützung

### File Watching

| Feature | Windows | macOS | Linux |
|---------|---------|-------|-------|
| **Library** | `fsnotify` | `fsnotify` | `fsnotify` |
| **Backend** | ReadDirectoryChangesW | FSEvents / kqueue | inotify |
| **Rekursiv** | ✓ Nativ | ✓ FSEvents | ⚠️ Manuell |
| **Max Watches** | ~10.000 | Unbegrenzt | ~8.192 (konfigurierbar) |

### Pfad-Handling

```go
// PathNormalizer normalisiert Pfade für alle Plattformen
type PathNormalizer struct{}

func (n *PathNormalizer) Normalize(path string) string {
    // Immer Forward-Slashes verwenden (intern)
    path = filepath.ToSlash(path)

    // Trailing Slash entfernen
    path = strings.TrimSuffix(path, "/")

    return path
}

func (n *PathNormalizer) ToNative(path string) string {
    return filepath.FromSlash(path)
}

func (n *PathNormalizer) IsAbsolute(path string) bool {
    // Windows: C:\, D:\, etc.
    if runtime.GOOS == "windows" {
        if len(path) >= 2 && path[1] == ':' {
            return true
        }
    }
    return filepath.IsAbs(path)
}
```

### Bekannte Plattform-Unterschiede

| Aspekt | Windows | macOS | Linux |
|--------|---------|-------|-------|
| **Pfad-Separator** | `\` | `/` | `/` |
| **Case-Sensitivity** | Nein | Standard Nein (HFS+) | Ja |
| **Symlinks** | NTFS ab Vista | ✓ | ✓ |
| **Max Pfadlänge** | 260 (erweiterbar) | 1024 | 4096 |
| **Versteckte Dateien** | Attribut | `.`-Prefix | `.`-Prefix |
| **Sperrung** | Exklusiv beim Schreiben | Advisory | Advisory |

### Symlink-Handling

```go
type SymlinkPolicy string

const (
    SymlinkPolicyFollow SymlinkPolicy = "follow"  // Symlinks folgen
    SymlinkPolicyIgnore SymlinkPolicy = "ignore"  // Symlinks ignorieren
    SymlinkPolicyError  SymlinkPolicy = "error"   // Fehler bei Symlinks
)
```

---

## Unterstützte Dateiformate

### Phase 1 (Initial)

| Format | Extension | Parser | Bibliothek |
|--------|-----------|--------|------------|
| Plain Text | `.txt` | Native | Go stdlib |
| Markdown | `.md` | Native | goldmark |
| HTML | `.html`, `.htm` | Native | x/net/html |
| JSON | `.json` | Native | Go stdlib |
| YAML | `.yaml`, `.yml` | Native | gopkg.in/yaml.v3 |
| XML | `.xml` | Native | encoding/xml |

### Phase 2 (Erweitert)

| Format | Extension | Parser | Bibliothek |
|--------|-----------|--------|------------|
| PDF | `.pdf` | Extern | pdfcpu / poppler |
| DOCX | `.docx` | Extern | unioffice |
| XLSX | `.xlsx` | Extern | excelize |
| PPTX | `.pptx` | Extern | unioffice |
| ODT | `.odt` | Extern | Custom ZIP parser |
| EPUB | `.epub` | Extern | go-epub |
| RTF | `.rtf` | Extern | Custom parser |

### Phase 3 (Code & Speziell)

| Format | Extension | Parser | Beschreibung |
|--------|-----------|--------|--------------|
| Source Code | `.go`, `.py`, `.js`, etc. | Native | Mit Syntax-Metadata |
| Jupyter | `.ipynb` | Native | JSON-basiert |
| CSV | `.csv` | Native | encoding/csv |
| Log Files | `.log` | Native | Zeilenbasiert |

### Parser Interface

```go
// DocumentParser parsed Dateien zu Text
type DocumentParser interface {
    // Extensions gibt die unterstützten Dateiendungen zurück
    Extensions() []string

    // Parse extrahiert Text und Metadaten aus einer Datei
    Parse(ctx context.Context, reader io.Reader, filename string) (*ParseResult, error)

    // CanParse prüft, ob die Datei verarbeitet werden kann
    CanParse(filename string, mimeType string) bool
}

type ParseResult struct {
    Content   string            // Extrahierter Text
    Title     string            // Dokumenttitel (falls vorhanden)
    Author    string            // Autor (falls vorhanden)
    Created   time.Time         // Erstellungsdatum (falls vorhanden)
    Metadata  map[string]string // Zusätzliche Metadaten
    Sections  []Section         // Strukturierte Abschnitte (optional)
    Language  string            // Erkannte Sprache
    PageCount int               // Seitenzahl (für PDFs etc.)
}

type Section struct {
    Title   string
    Content string
    Level   int // Hierarchieebene (1 = H1, 2 = H2, etc.)
}
```

---

## API-Design

### gRPC Service (Hypatia)

```protobuf
service HypatiaService {
    // Bestehende RPCs...

    // === DataSource Management ===

    // Datenquelle hinzufügen
    rpc AddDataSource(AddDataSourceRequest) returns (DataSource);

    // Datenquelle aktualisieren
    rpc UpdateDataSource(UpdateDataSourceRequest) returns (DataSource);

    // Datenquelle entfernen
    rpc RemoveDataSource(RemoveDataSourceRequest) returns (google.protobuf.Empty);

    // Datenquellen auflisten
    rpc ListDataSources(ListDataSourcesRequest) returns (ListDataSourcesResponse);

    // Datenquelle abrufen
    rpc GetDataSource(GetDataSourceRequest) returns (DataSource);

    // === Sync Operations ===

    // Manuellen Scan starten
    rpc TriggerSync(TriggerSyncRequest) returns (SyncStatus);

    // Sync-Status abrufen
    rpc GetSyncStatus(GetSyncStatusRequest) returns (SyncStatus);

    // Sync-Historie abrufen
    rpc GetSyncHistory(GetSyncHistoryRequest) returns (GetSyncHistoryResponse);

    // === File Operations ===

    // Dateistatus abrufen
    rpc GetFileStatus(GetFileStatusRequest) returns (FileStatus);

    // Dateien einer Datenquelle auflisten
    rpc ListDataSourceFiles(ListDataSourceFilesRequest) returns (ListDataSourceFilesResponse);
}
```

### REST API (Kant)

```yaml
paths:
  # === Datenquellen-Management ===
  /api/v1/datasources:
    get:
      summary: Liste aller Datenquellen
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  datasources:
                    type: array
                    items:
                      $ref: '#/components/schemas/DataSource'
                  total:
                    type: integer

    post:
      summary: Neue Datenquelle hinzufügen
      description: |
        Erstellt eine neue Datenquelle mit automatischer Collection.
        Die Collection wird aus dem Namen abgeleitet (slugified).
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateDataSourceRequest'
      responses:
        201:
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DataSource'

  /api/v1/datasources/{id}:
    get:
      summary: Datenquelle abrufen
    put:
      summary: Datenquelle aktualisieren
    delete:
      summary: Datenquelle entfernen (inkl. Collection)

  /api/v1/datasources/{id}/paths:
    post:
      summary: Verzeichnis zur Datenquelle hinzufügen
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                path:
                  type: string
                recursive:
                  type: boolean
                  default: true
    delete:
      summary: Verzeichnis aus Datenquelle entfernen

  /api/v1/datasources/{id}/sync:
    post:
      summary: Synchronisation starten
      responses:
        202:
          description: Sync gestartet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SyncStatus'

    get:
      summary: Sync-Status abrufen

  /api/v1/datasources/{id}/files:
    get:
      summary: Dateien der Datenquelle auflisten
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [pending, indexed, error, deleted]
        - name: limit
          in: query
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer

  # === Erweiterte Such-API ===
  /api/v1/search:
    post:
      summary: Semantische Suche (erweitert)
      description: |
        Sucht in einer oder mehreren Datenquellen/Collections.
        Ohne `collections` Parameter wird in allen gesucht.
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - query
              properties:
                query:
                  type: string
                  description: Suchanfrage
                collections:
                  type: array
                  items:
                    type: string
                  description: |
                    Liste der Collection-Namen oder Datenquellen-IDs.
                    Leer = alle Datenquellen durchsuchen.
                top_k:
                  type: integer
                  default: 10
                min_score:
                  type: number
                  default: 0.0
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  query:
                    type: string
                  results:
                    type: array
                    items:
                      $ref: '#/components/schemas/SearchResult'
                  collections_searched:
                    type: array
                    items:
                      type: string
                  total:
                    type: integer

components:
  schemas:
    DataSource:
      type: object
      properties:
        id:
          type: string
          description: Eindeutige ID der Datenquelle
        type:
          type: string
          enum: [filesystem, s3, webdav]
        name:
          type: string
          description: Benutzerfreundlicher Name
        collection_name:
          type: string
          description: |
            Name der zugehörigen Collection (automatisch aus name generiert).
            Beispiel: "Projekt Alpha" → "projekt-alpha"
        status:
          type: string
          enum: [active, paused, scanning, error]
        paths:
          type: array
          items:
            type: string
          description: Liste der überwachten Verzeichnisse
        config:
          type: object
          # Je nach Typ unterschiedlich
        statistics:
          $ref: '#/components/schemas/DataSourceStats'
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    DataSourceStats:
      type: object
      properties:
        total_files:
          type: integer
        indexed_files:
          type: integer
        pending_files:
          type: integer
        error_files:
          type: integer
        total_size:
          type: integer
          format: int64
        last_sync:
          type: string
          format: date-time

    CreateDataSourceRequest:
      type: object
      required:
        - type
        - name
        - config
      properties:
        type:
          type: string
          enum: [filesystem]
        name:
          type: string
        config:
          $ref: '#/components/schemas/FilesystemConfig'

    FilesystemConfig:
      type: object
      required:
        - paths
      properties:
        paths:
          type: array
          items:
            type: string
          description: Zu überwachende Verzeichnisse
        recursive:
          type: boolean
          default: true
        extensions:
          type: array
          items:
            type: string
          description: Erlaubte Dateiendungen (leer = alle)
        exclude:
          type: array
          items:
            type: string
          description: Auszuschließende Muster (glob)
        max_file_size:
          type: integer
          format: int64
          default: 104857600
          description: Max. Dateigröße in Bytes (default 100MB)
        collection:
          type: string
          description: Ziel-Collection
        sync_mode:
          type: string
          enum: [watch, poll, manual]
          default: watch
        poll_interval:
          type: string
          description: Polling-Intervall (z.B. "5m", "1h")
```

---

## Konfiguration

### config.toml

```toml
[hypatia.datasources]
# Aktiviert die DataSource-Funktionalität
enabled = true

# Datenbank für FileState Tracking
# Pfad relativ zu ~/.mdw/ oder absolut
state_db_path = "~/.mdw/hypatia/datasources.db"

# Standard-Einstellungen für Filesystem-Quellen
[hypatia.datasources.filesystem]
# Standard-Sync-Modus
default_sync_mode = "watch"

# Standard-Poll-Intervall (für poll-Modus)
default_poll_interval = "5m"

# Maximale Dateigröße (100 MB)
max_file_size = 104857600

# Standard-Ausschlüsse
default_exclude = [
    ".*",           # Versteckte Dateien/Ordner
    "node_modules",
    "__pycache__",
    "*.tmp",
    "*.swp",
    "~$*",          # Office Temp-Dateien
]

# Unterstützte Dateiendungen (leer = alle)
supported_extensions = [
    # Text
    ".txt", ".md", ".markdown",
    # Dokumente
    ".pdf", ".docx", ".doc", ".odt", ".rtf",
    # Daten
    ".json", ".yaml", ".yml", ".xml", ".csv",
    # Web
    ".html", ".htm",
    # Code
    ".go", ".py", ".js", ".ts", ".java", ".rs", ".c", ".cpp",
]

# Parallelität beim Ingestieren
ingest_workers = 4

# Batch-Größe beim Ingestieren
ingest_batch_size = 10

# Retry-Konfiguration
[hypatia.datasources.retry]
max_attempts = 3
initial_backoff = "1s"
max_backoff = "1m"
```

---

## Benutzeroberfläche (TUI)

### Verzeichnis hinzufügen

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Datenquelle hinzufügen                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Name:        [Projektdokumentation_________________]               │
│                                                                      │
│  Verzeichnisse:                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ /Users/mikes/Documents/Projekt                                 │ │
│  │ /Users/mikes/Notes                                             │ │
│  │                                                          [+ Add]│ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  Optionen:                                                           │
│  [x] Unterverzeichnisse einschließen                                │
│  [ ] Nur bestimmte Dateitypen: [.md, .txt, .pdf_______________]     │
│  [ ] Muster ausschließen:      [node_modules, .git__________]       │
│                                                                      │
│  Sync-Modus:                                                         │
│  (•) Echtzeit (Watch)                                               │
│  ( ) Polling alle [5] Minuten                                       │
│  ( ) Nur manuell                                                    │
│                                                                      │
│  Ziel-Collection: [projekt-docs________________] [Neu erstellen]   │
│                                                                      │
│                        [Abbrechen]  [Hinzufügen]                    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Datenquellen-Übersicht

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Datenquellen                         [+ Neu] │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ 📁 Projektdokumentation                              [Active]  │ │
│  │    /Users/mikes/Documents/Projekt                              │ │
│  │    /Users/mikes/Notes                                          │ │
│  │    ─────────────────────────────────────────────────────────── │ │
│  │    📊 245 Dateien (198 indexiert, 47 ausstehend)              │ │
│  │    🕐 Letzte Sync: vor 2 Minuten                              │ │
│  │                              [Sync] [Pausieren] [Bearbeiten]   │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ 📁 Technische Dokumentation                          [Paused]  │ │
│  │    /Users/mikes/Docs/Technical                                 │ │
│  │    ─────────────────────────────────────────────────────────── │ │
│  │    📊 1.203 Dateien (1.203 indexiert)                         │ │
│  │    🕐 Letzte Sync: vor 1 Tag                                  │ │
│  │                            [Sync] [Fortsetzen] [Bearbeiten]    │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ 📁 Persönliche Notizen                           [Scanning...] │ │
│  │    /Users/mikes/Personal/Notes                                 │ │
│  │    ─────────────────────────────────────────────────────────── │ │
│  │    ████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  67% (402/600)│ │
│  │    ⏱️  Geschätzte Restzeit: 2 Minuten                         │ │
│  │                                                   [Abbrechen]  │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Fehlerbehandlung

### Retry-Strategie

```go
type RetryConfig struct {
    MaxAttempts    int           // Maximale Versuche
    InitialBackoff time.Duration // Initiale Wartezeit
    MaxBackoff     time.Duration // Maximale Wartezeit
    Multiplier     float64       // Backoff-Multiplikator
}

// Exponential Backoff mit Jitter
func (r *RetryConfig) NextBackoff(attempt int) time.Duration {
    backoff := float64(r.InitialBackoff) * math.Pow(r.Multiplier, float64(attempt))
    if backoff > float64(r.MaxBackoff) {
        backoff = float64(r.MaxBackoff)
    }
    // Jitter: ±25%
    jitter := backoff * (0.75 + 0.5*rand.Float64())
    return time.Duration(jitter)
}
```

### Fehlertypen

| Fehler | Behandlung | Retry |
|--------|------------|-------|
| Datei nicht lesbar | In Error-Queue, User benachrichtigen | Nein |
| Datei gesperrt | Warten, später erneut versuchen | Ja (3x) |
| Parser-Fehler | In Error-Queue, ggf. als Text fallback | Nein |
| Embedding-Fehler | Retry mit Backoff | Ja (5x) |
| Speicher-Fehler | Retry mit Backoff, ggf. Admin-Alert | Ja (10x) |
| Verzeichnis nicht erreichbar | Datenquelle pausieren, User benachrichtigen | Nein |

---

## Sicherheitsüberlegungen

### Berechtigungen

1. **Lese-Berechtigungen**: Nur Verzeichnisse mit Leseberechtigung können hinzugefügt werden
2. **Symlink-Prüfung**: Symlinks werden aufgelöst und auf Zyklus geprüft
3. **Path Traversal**: Pfade werden normalisiert und validiert

### Sensible Daten

```go
// SensitivePatterns sind Muster für sensible Dateien
var SensitivePatterns = []string{
    "*.pem",
    "*.key",
    "*.p12",
    "*.pfx",
    "*password*",
    "*secret*",
    "*credential*",
    ".env",
    ".env.*",
    "*.kdbx",      // KeePass
    "id_rsa*",
    "id_ed25519*",
}

// WarnIfSensitive prüft auf sensible Dateien
func WarnIfSensitive(path string) bool {
    for _, pattern := range SensitivePatterns {
        if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
            return true
        }
    }
    return false
}
```

---

## Implementierungsplan

### Phase 1: Grundgerüst (1-2 Wochen)
- [ ] DataSource Interface definieren
- [ ] FilesystemSource implementieren (ohne Watch)
- [ ] FileState Datenbank (SQLite)
- [ ] Basic Scanner mit Filter
- [ ] Integration mit bestehendem Ingest

### Phase 2: Synchronisation (1 Woche)
- [ ] fsnotify Integration
- [ ] Änderungserkennung
- [ ] Polling-Modus als Fallback
- [ ] Sync-Status Tracking

### Phase 3: Parser (1-2 Wochen)
- [ ] Text/Markdown Parser
- [ ] HTML Parser
- [ ] PDF Parser
- [ ] DOCX Parser
- [ ] Parser-Registry

### Phase 4: API & UI (1 Woche)
- [ ] gRPC Service erweitern
- [ ] REST Endpoints in Kant
- [ ] TUI für Datenquellen-Management

### Phase 5: Hardening (1 Woche)
- [ ] Cross-Platform Tests
- [ ] Error Handling & Retry
- [ ] Performance-Optimierung
- [ ] Dokumentation

---

## Design-Entscheidungen

Die folgenden Entscheidungen wurden getroffen:

### 1. Große Dateien
| Aspekt | Entscheidung |
|--------|--------------|
| **Verhalten** | Konfigurierbar pro Datenquelle |
| **Standard-Limit** | 100 MB |
| **Überschreitung** | Datei wird übersprungen mit Warnung im Log |
| **Konfiguration** | `max_file_size` in Datenquellen-Config |

```go
// Beispiel-Konfiguration
FilesystemConfig{
    MaxFileSize: 100 * 1024 * 1024, // 100 MB (Default)
}
```

### 2. Duplikate innerhalb einer Datenquelle
| Aspekt | Entscheidung |
|--------|--------------|
| **Verhalten** | Hash-basierte Deduplizierung |
| **Speicherung** | Ein Dokument, mehrere Pfade in Metadaten |
| **Erkennung** | SHA-256 Hash des Dateiinhalts |

```go
// FileState mit mehreren Pfaden
type FileState struct {
    Hash      string   // SHA-256
    Paths     []string // Alle Pfade zur gleichen Datei
    DocumentID string  // Ein Dokument in Hypatia
}
```

### 3. Duplikate über Datenquellen hinweg (Cross-Source)
| Aspekt | Entscheidung |
|--------|--------------|
| **Verhalten** | Separate Dokumente in separaten Collections |
| **Deduplizierung** | Keine automatische Deduplizierung |
| **Begründung** | Unterschiedlicher Kontext, unterschiedliche Suchbereiche |

### 4. Versionierung
| Aspekt | Entscheidung |
|--------|--------------|
| **v1.0** | Keine Versionierung - immer überschreiben |
| **Später** | Optionale Versionierung als Feature geplant |
| **Bei Änderung** | Altes Dokument löschen, neues erstellen |

### 5. Binärdateien (Bilder, Videos, etc.)
| Aspekt | Entscheidung |
|--------|--------------|
| **v1.0** | Ignorieren (nicht indizieren) |
| **Später geplant** | Metadaten-Extraktion (EXIF, ID3, etc.) |
| **Später geplant** | OCR für Bilder mit Text |
| **Später geplant** | Audio-Transkription |

```go
// Binärdatei-Erkennung
var BinaryExtensions = []string{
    // Bilder
    ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg",
    // Audio
    ".mp3", ".wav", ".flac", ".ogg", ".m4a",
    // Video
    ".mp4", ".avi", ".mkv", ".mov", ".webm",
    // Archive
    ".zip", ".tar", ".gz", ".rar", ".7z",
    // Ausführbare
    ".exe", ".dll", ".so", ".dylib",
}
```

---

## Abhängigkeiten

| Paket | Version | Zweck |
|-------|---------|-------|
| `github.com/fsnotify/fsnotify` | v1.7+ | File Watching |
| `github.com/yuin/goldmark` | v1.6+ | Markdown Parsing |
| `github.com/pdfcpu/pdfcpu` | v0.6+ | PDF Parsing |
| `github.com/unidoc/unioffice` | v1.30+ | Office-Dokumente |
| `golang.org/x/net/html` | latest | HTML Parsing |
| `github.com/mattn/go-sqlite3` | v1.14+ | State Database |
