// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     registration
// Description: Service registration with Russell discovery service
// Author:      Mike Stoffels with Claude
// Created:     2025-12-06
// License:     MIT
// ============================================================================

package registration

import (
	"context"
	"fmt"
	"sync"
	"time"

	russellpb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/pkg/core/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceRegistration handles registration with Russell
type ServiceRegistration struct {
	name         string
	version      string
	address      string
	port         int
	russellAddr  string
	serviceID    string
	tags         []string
	metadata     map[string]string
	logger       *logging.Logger
	mu           sync.Mutex
	stopCh       chan struct{}
	heartbeatInt time.Duration
}

// Config holds registration configuration
type Config struct {
	Name        string
	Version     string
	Address     string
	Port        int
	RussellAddr string
	Tags        []string
	Metadata    map[string]string
}

// New creates a new service registration
func New(cfg Config) *ServiceRegistration {
	if cfg.RussellAddr == "" {
		cfg.RussellAddr = "localhost:9100"
	}
	if cfg.Address == "" {
		cfg.Address = "localhost"
	}
	if cfg.Version == "" {
		cfg.Version = "0.0.0"
	}

	return &ServiceRegistration{
		name:         cfg.Name,
		version:      cfg.Version,
		address:      cfg.Address,
		port:         cfg.Port,
		russellAddr:  cfg.RussellAddr,
		tags:         cfg.Tags,
		metadata:     cfg.Metadata,
		logger:       logging.New("registration"),
		stopCh:       make(chan struct{}),
		heartbeatInt: 10 * time.Second,
	}
}

// Register registers the service with Russell
func (sr *ServiceRegistration) Register(ctx context.Context) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	conn, err := sr.dialRussell(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := russellpb.NewRussellServiceClient(conn)

	resp, err := client.Register(ctx, &russellpb.RegisterRequest{
		Name:     sr.name,
		Address:  sr.address,
		Port:     int32(sr.port),
		Tags:     sr.tags,
		Metadata: sr.metadata,
		Version:  sr.version,
	})
	if err != nil {
		return fmt.Errorf("failed to register with Russell: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	sr.serviceID = resp.Id
	sr.logger.Info("Service registered with Russell",
		"service", sr.name,
		"id", sr.serviceID,
	)

	return nil
}

// Deregister removes the service from Russell
func (sr *ServiceRegistration) Deregister(ctx context.Context) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.serviceID == "" {
		return nil
	}

	conn, err := sr.dialRussell(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := russellpb.NewRussellServiceClient(conn)

	_, err = client.Deregister(ctx, &russellpb.DeregisterRequest{
		Id: sr.serviceID,
	})
	if err != nil {
		return fmt.Errorf("failed to deregister: %w", err)
	}

	sr.logger.Info("Service deregistered from Russell", "id", sr.serviceID)
	sr.serviceID = ""

	return nil
}

// StartHeartbeat starts the heartbeat goroutine
func (sr *ServiceRegistration) StartHeartbeat(ctx context.Context) {
	go sr.heartbeatLoop(ctx)
}

// StopHeartbeat stops the heartbeat goroutine
func (sr *ServiceRegistration) StopHeartbeat() {
	close(sr.stopCh)
}

// heartbeatLoop sends periodic heartbeats
func (sr *ServiceRegistration) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(sr.heartbeatInt)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sr.stopCh:
			return
		case <-ticker.C:
			sr.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat sends a heartbeat to Russell
func (sr *ServiceRegistration) sendHeartbeat(ctx context.Context) {
	sr.mu.Lock()
	id := sr.serviceID
	sr.mu.Unlock()

	if id == "" {
		return
	}

	conn, err := sr.dialRussell(ctx)
	if err != nil {
		sr.logger.Warn("Failed to connect for heartbeat", "error", err)
		return
	}
	defer conn.Close()

	client := russellpb.NewRussellServiceClient(conn)

	resp, err := client.Heartbeat(ctx, &russellpb.HeartbeatRequest{
		Id: id,
	})
	if err != nil {
		sr.logger.Warn("Heartbeat failed", "error", err)
		return
	}

	if !resp.Acknowledged {
		sr.logger.Warn("Heartbeat not acknowledged")
		// Try to re-register
		sr.mu.Lock()
		sr.serviceID = ""
		sr.mu.Unlock()
		sr.Register(ctx)
	}
}

// dialRussell creates a connection to Russell
func (sr *ServiceRegistration) dialRussell(ctx context.Context) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, sr.russellAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Russell at %s: %w", sr.russellAddr, err)
	}

	return conn, nil
}

// RegisterService is a helper function to register a service once
func RegisterService(ctx context.Context, name, version string, port int, russellAddr string) (*ServiceRegistration, error) {
	reg := New(Config{
		Name:        name,
		Version:     version,
		Port:        port,
		RussellAddr: russellAddr,
	})

	if err := reg.Register(ctx); err != nil {
		return nil, err
	}

	reg.StartHeartbeat(ctx)
	return reg, nil
}
