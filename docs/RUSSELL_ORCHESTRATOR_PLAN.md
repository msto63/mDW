# Russell Orchestrator - Implementierungsplan

## Übersicht

Russell wird vom reinen Service-Registry zum vollständigen Service-Orchestrator umgebaut.
ControlCenter wird vereinfacht und zeigt nur noch Russell's Sicht der Welt an.

---

## Phase 1: Russell Service-Konfiguration

### 1.1 Service-Definition erweitern

**Datei:** `internal/russell/orchestrator/config.go` (neu)

```go
type ServiceConfig struct {
    Name         string
    ShortName    string
    Description  string
    GRPCPort     int
    HTTPPort     int
    Command      []string        // z.B. ["./bin/mdw", "serve", "kant"]
    Dependencies []string        // z.B. ["turing", "ollama"]
    StartOrder   int             // Startreihenfolge (1 = zuerst)
    MaxRetries   int             // Max. Neustartversuche (default: 3)
    HealthCheck  HealthCheckConfig
}

type HealthCheckConfig struct {
    Type     string        // "grpc", "http", "tcp"
    Endpoint string        // z.B. "/api/v1/health"
    Interval time.Duration // z.B. 10s
    Timeout  time.Duration // z.B. 3s
}
```

### 1.2 Service-Konfiguration laden

**Datei:** `configs/services.toml` (neu)

```toml
[[services]]
name = "Kant"
short_name = "kant"
description = "API Gateway"
http_port = 8080
command = ["./bin/mdw", "serve", "kant"]
dependencies = ["turing"]
start_order = 10
max_retries = 3

[services.health_check]
type = "http"
endpoint = "/api/v1/health"
interval = "10s"
timeout = "3s"

[[services]]
name = "Turing"
short_name = "turing"
description = "LLM Management"
grpc_port = 9200
http_port = 9201
command = ["./bin/mdw", "serve", "turing"]
dependencies = []
start_order = 1
max_retries = 3

[services.health_check]
type = "grpc"
interval = "10s"
timeout = "3s"

# ... weitere Services
```

---

## Phase 2: Russell Orchestrator-Kern

### 2.1 Orchestrator-Struktur

**Datei:** `internal/russell/orchestrator/orchestrator.go` (neu)

```go
type Orchestrator struct {
    mu            sync.RWMutex
    services      map[string]*ManagedService
    config        []ServiceConfig
    logger        *logging.Logger
    binaryPath    string

    // Channels
    stopCh        chan struct{}
    eventCh       chan ServiceEvent
}

type ManagedService struct {
    Config        ServiceConfig
    Status        ServiceStatus
    Process       *os.Process
    PID           int
    StartedAt     time.Time
    RestartCount  int
    LastError     string
    LastHealthCheck time.Time
    Healthy       bool
}

type ServiceEvent struct {
    Type      EventType // Started, Stopped, Failed, HealthCheckFailed
    Service   string
    Message   string
    Timestamp time.Time
}
```

### 2.2 Startup-Logik

```go
func (o *Orchestrator) StartAll(ctx context.Context) error {
    // 1. Services nach StartOrder sortieren
    sorted := o.getSortedServices()

    for _, svc := range sorted {
        // 2. Dependencies prüfen
        if err := o.waitForDependencies(ctx, svc); err != nil {
            return fmt.Errorf("dependency check failed for %s: %w", svc.Name, err)
        }

        // 3. Port-Konflikt prüfen
        if conflict := o.checkPortConflict(svc); conflict != nil {
            // Versuchen, existierenden Service zu übernehmen
            if err := o.handlePortConflict(ctx, svc, conflict); err != nil {
                return err
            }
            continue
        }

        // 4. Service starten mit Retry
        if err := o.startServiceWithRetry(ctx, svc); err != nil {
            return fmt.Errorf("failed to start %s: %w", svc.Name, err)
        }
    }

    return nil
}

func (o *Orchestrator) startServiceWithRetry(ctx context.Context, svc ServiceConfig) error {
    var lastErr error

    for attempt := 1; attempt <= svc.MaxRetries; attempt++ {
        o.logger.Info("Starting service",
            "service", svc.Name,
            "attempt", attempt,
            "maxRetries", svc.MaxRetries)

        if err := o.startService(ctx, svc); err != nil {
            lastErr = err
            o.logger.Warn("Service start failed, retrying",
                "service", svc.Name,
                "attempt", attempt,
                "error", err)

            // Kurz warten vor Retry
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(2 * time.Second):
            }
            continue
        }

        // Warten bis Service healthy ist
        if err := o.waitForHealthy(ctx, svc, 30*time.Second); err != nil {
            lastErr = err
            o.stopService(svc.ShortName)
            continue
        }

        return nil
    }

    return fmt.Errorf("failed after %d attempts: %w", svc.MaxRetries, lastErr)
}
```

### 2.3 Port-Konflikt-Handling

```go
func (o *Orchestrator) checkPortConflict(svc ServiceConfig) *PortConflict {
    port := svc.GRPCPort
    if port == 0 {
        port = svc.HTTPPort
    }

    conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
    if err != nil {
        return nil // Port frei
    }
    conn.Close()

    // Port belegt - PID ermitteln
    pid := o.findProcessOnPort(port)

    return &PortConflict{
        Port:    port,
        PID:     pid,
        Service: svc.ShortName,
    }
}

func (o *Orchestrator) handlePortConflict(ctx context.Context, svc ServiceConfig, conflict *PortConflict) error {
    o.logger.Warn("Port conflict detected",
        "service", svc.Name,
        "port", conflict.Port,
        "existingPID", conflict.PID)

    // 1. Versuchen, den existierenden Prozess zu kontaktieren
    if o.tryAdoptService(ctx, svc, conflict.Port) {
        o.logger.Info("Adopted existing service",
            "service", svc.Name,
            "port", conflict.Port)
        return nil
    }

    // 2. Prozess beenden und neu starten
    o.logger.Info("Terminating conflicting process",
        "service", svc.Name,
        "pid", conflict.PID)

    if err := o.killProcess(conflict.PID); err != nil {
        return fmt.Errorf("failed to kill process %d: %w", conflict.PID, err)
    }

    // Warten bis Port frei
    time.Sleep(2 * time.Second)

    return o.startServiceWithRetry(ctx, svc)
}
```

### 2.4 Health-Monitoring

```go
func (o *Orchestrator) runHealthMonitor(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            o.checkAllServices(ctx)
        }
    }
}

func (o *Orchestrator) checkAllServices(ctx context.Context) {
    o.mu.Lock()
    defer o.mu.Unlock()

    for name, svc := range o.services {
        if svc.Status != ServiceRunning {
            continue
        }

        healthy := o.performHealthCheck(ctx, svc)
        svc.LastHealthCheck = time.Now()

        if !healthy {
            svc.Healthy = false
            o.logger.Warn("Service unhealthy",
                "service", name,
                "restartCount", svc.RestartCount)

            // Auto-Restart bei Fehlern
            if svc.RestartCount < svc.Config.MaxRetries {
                go o.restartService(ctx, name)
            }
        } else {
            svc.Healthy = true
        }
    }
}
```

---

## Phase 3: Russell gRPC API erweitern

### 3.1 Neue Proto-Definitionen

**Datei:** `api/proto/russell.proto` (erweitern)

```protobuf
// Neue Messages
message StartAllRequest {
    bool force = 1;  // Force restart auch wenn Services laufen
}

message StartAllResponse {
    bool success = 1;
    repeated ServiceStartResult results = 2;
}

message ServiceStartResult {
    string name = 1;
    bool success = 2;
    string error = 3;
    int32 attempts = 4;
}

message OrchestratorStatus {
    string status = 1;  // "starting", "running", "stopping", "stopped"
    int32 total_services = 2;
    int32 running_services = 3;
    int32 healthy_services = 4;
    int32 failed_services = 5;
    repeated ServiceStatus services = 6;
    google.protobuf.Timestamp started_at = 7;
}

// Neue Service-Methoden
service RussellService {
    // Existierende Methoden...

    // Neue Orchestrator-Methoden
    rpc StartAllServices(StartAllRequest) returns (StartAllResponse);
    rpc StopAllServices(common.Empty) returns (common.Empty);
    rpc RestartService(ServiceRequest) returns (common.Empty);
    rpc GetOrchestratorStatus(common.Empty) returns (OrchestratorStatus);
}
```

---

## Phase 4: ControlCenter Vereinfachung

### 4.1 Neues Layout

**Datei:** `internal/tui/controlcenter/model.go` (umbauen)

```
┌─────────────────────────────────────────────────────────┐
│              mDW Control Center v0.1.0                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  RUSSELL ORCHESTRATOR                                   │
│  ════════════════════                                   │
│  Status: ● RUNNING          Uptime: 2h 15m              │
│  Services: 5/7 running      Healthy: 5/5                │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  SERVICE         PORT    STATUS      HEALTH    UPTIME   │
│  ─────────────────────────────────────────────────────  │
│  → Kant          :8080   ● running   ✓ ok      1h 23m   │
│    Turing        :9200   ● running   ✓ ok      1h 23m   │
│    Hypatia       :9220   ○ stopped   — n/a     —        │
│    Babbage       :9150   ● running   ✓ ok      45m      │
│    Leibniz       :9140   ● running   ✓ ok      45m      │
│    Bayes         :9120   ● running   ✓ ok      45m      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│  Dependencies: Go ✓  Ollama ✓  Models ✓                 │
│                                                         │
│  [a] start all  [s] stop all  [Enter] toggle  [q] quit  │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Vereinfachter ServiceManager

```go
// ControlCenter fragt NUR Russell
type ServiceManager struct {
    russellClient russellpb.RussellServiceClient
    conn          *grpc.ClientConn

    orchestratorStatus *OrchestratorStatus
    lastRefresh        time.Time
}

func (sm *ServiceManager) RefreshStatus() error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    status, err := sm.russellClient.GetOrchestratorStatus(ctx, &commonpb.Empty{})
    if err != nil {
        return err
    }

    sm.orchestratorStatus = status
    sm.lastRefresh = time.Now()
    return nil
}

// Keine direkten Port-Checks mehr!
// Alle Aktionen gehen über Russell
func (sm *ServiceManager) StartAll() error {
    ctx := context.Background()
    _, err := sm.russellClient.StartAllServices(ctx, &russellpb.StartAllRequest{})
    return err
}
```

---

## Phase 5: Migrations-Schritte

### Schritt 1: Russell Orchestrator implementieren
- [ ] `internal/russell/orchestrator/` Package erstellen
- [ ] Service-Konfiguration laden
- [ ] Basis-Startup-Logik implementieren
- [ ] Unit-Tests schreiben

### Schritt 2: Startup mit Retry
- [ ] Retry-Logik implementieren
- [ ] Port-Konflikt-Erkennung
- [ ] Dependency-Reihenfolge

### Schritt 3: Health-Monitoring
- [ ] Periodische Health-Checks
- [ ] Auto-Restart bei Fehlern
- [ ] Event-Logging

### Schritt 4: gRPC API erweitern
- [ ] Proto-Definitionen erweitern
- [ ] Neue Endpoints implementieren
- [ ] Bestehende Tests anpassen

### Schritt 5: ControlCenter umbauen
- [ ] Russell-Status-Anzeige oben
- [ ] Direkte Port-Checks entfernen
- [ ] Nur noch Russell-API nutzen
- [ ] UI-Tests

### Schritt 6: Integration & Testing
- [ ] End-to-End Tests
- [ ] Failure-Szenarien testen
- [ ] Performance-Tests

---

## Zeitschätzung

| Phase | Beschreibung | Aufwand |
|-------|-------------|---------|
| 1 | Service-Konfiguration | Klein |
| 2 | Orchestrator-Kern | Mittel-Groß |
| 3 | gRPC API | Klein |
| 4 | ControlCenter | Mittel |
| 5 | Migration & Tests | Mittel |

---

## Risiken & Mitigation

| Risiko | Mitigation |
|--------|-----------|
| Prozess-Kill kann Daten verlieren | Graceful shutdown mit Timeout |
| Race Conditions bei Port-Checks | Mutex und atomare Operationen |
| Health-Check-Flapping | Debouncing, mehrere Checks vor Restart |
| Russell selbst crasht | Supervisor/systemd für Russell |

---

## Offene Fragen

1. **Soll Russell auch externe Dependencies prüfen?** (Ollama, PostgreSQL)
2. **Wie verhält sich Russell beim eigenen Crash?** (PID-File, Lock?)
3. **Sollen Service-Logs über Russell gesammelt werden?**
4. **Wie wird Russell selbst gestartet?** (systemd, ControlCenter, CLI?)
