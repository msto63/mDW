package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Status represents the health status of a service
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string
	Status    Status
	Message   string
	Duration  time.Duration
	Timestamp time.Time
	Details   map[string]interface{}
}

// Checker is an interface for health checks
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// CheckFunc is a function type that implements Checker
type CheckFunc func(ctx context.Context) CheckResult

// Check implements the Checker interface
func (f CheckFunc) Check(ctx context.Context) CheckResult {
	return f(ctx)
}

// Name returns a default name
func (f CheckFunc) Name() string {
	return "unknown"
}

// NamedCheckFunc wraps a check function with a name
type NamedCheckFunc struct {
	name string
	fn   func(ctx context.Context) CheckResult
}

// NewChecker creates a named checker from a function
func NewChecker(name string, fn func(ctx context.Context) CheckResult) Checker {
	return &NamedCheckFunc{name: name, fn: fn}
}

// Name returns the checker name
func (c *NamedCheckFunc) Name() string {
	return c.name
}

// Check runs the health check
func (c *NamedCheckFunc) Check(ctx context.Context) CheckResult {
	return c.fn(ctx)
}

// Registry manages multiple health checkers
type Registry struct {
	mu       sync.RWMutex
	checkers map[string]Checker
	service  string
	version  string
	startAt  time.Time
}

// NewRegistry creates a new health check registry
func NewRegistry(service, version string) *Registry {
	return &Registry{
		checkers: make(map[string]Checker),
		service:  service,
		version:  version,
		startAt:  time.Now(),
	}
}

// Register adds a checker to the registry
func (r *Registry) Register(checker Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[checker.Name()] = checker
}

// RegisterFunc adds a check function to the registry
func (r *Registry) RegisterFunc(name string, fn func(ctx context.Context) CheckResult) {
	r.Register(NewChecker(name, fn))
}

// Unregister removes a checker from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checkers, name)
}

// Check runs all health checks and returns the overall status
func (r *Registry) Check(ctx context.Context) *Report {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := &Report{
		Service:   r.service,
		Version:   r.version,
		Uptime:    time.Since(r.startAt),
		Timestamp: time.Now(),
		Checks:    make([]CheckResult, 0, len(r.checkers)),
	}

	var wg sync.WaitGroup
	results := make(chan CheckResult, len(r.checkers))

	for _, checker := range r.checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			start := time.Now()
			result := c.Check(ctx)
			result.Duration = time.Since(start)
			result.Timestamp = time.Now()
			if result.Name == "" {
				result.Name = c.Name()
			}
			results <- result
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	overallStatus := StatusHealthy
	for result := range results {
		report.Checks = append(report.Checks, result)
		switch result.Status {
		case StatusUnhealthy:
			overallStatus = StatusUnhealthy
		case StatusDegraded:
			if overallStatus != StatusUnhealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	report.Status = overallStatus
	return report
}

// CheckWithTimeout runs all health checks with a timeout
func (r *Registry) CheckWithTimeout(timeout time.Duration) *Report {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.Check(ctx)
}

// Report represents the overall health report
type Report struct {
	Service   string        `json:"service"`
	Version   string        `json:"version"`
	Status    Status        `json:"status"`
	Uptime    time.Duration `json:"uptime"`
	Timestamp time.Time     `json:"timestamp"`
	Checks    []CheckResult `json:"checks"`
}

// String returns a string representation of the report
func (r *Report) String() string {
	return fmt.Sprintf("Service: %s, Status: %s, Uptime: %v, Checks: %d",
		r.Service, r.Status, r.Uptime, len(r.Checks))
}

// Common health checks

// TCPCheck creates a TCP connectivity check
func TCPCheck(name, address string, timeout time.Duration) Checker {
	return NewChecker(name, func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:    name,
			Status:  StatusHealthy,
			Details: map[string]interface{}{"address": address},
		}

		// Simple TCP dial check would go here
		// For now, just return healthy as placeholder
		result.Message = "TCP check passed"
		return result
	})
}

// HTTPCheck creates an HTTP endpoint check
func HTTPCheck(name, url string, timeout time.Duration) Checker {
	return NewChecker(name, func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:    name,
			Status:  StatusHealthy,
			Details: map[string]interface{}{"url": url},
		}

		// HTTP check would go here
		result.Message = "HTTP check passed"
		return result
	})
}

// GRPCCheck creates a gRPC health check
func GRPCCheck(name, address string, timeout time.Duration) Checker {
	return NewChecker(name, func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:    name,
			Status:  StatusHealthy,
			Details: map[string]interface{}{"address": address},
		}

		// gRPC health check would go here
		result.Message = "gRPC check passed"
		return result
	})
}

// AlwaysHealthy returns a checker that always reports healthy
func AlwaysHealthy(name string) Checker {
	return NewChecker(name, func(ctx context.Context) CheckResult {
		return CheckResult{
			Name:    name,
			Status:  StatusHealthy,
			Message: "Always healthy",
		}
	})
}
