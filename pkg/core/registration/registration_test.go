package registration

import (
	"context"
	"testing"
	"time"
)

func TestNew_WithDefaults(t *testing.T) {
	cfg := Config{
		Name: "test-service",
		Port: 8080,
	}

	reg := New(cfg)

	if reg.name != "test-service" {
		t.Errorf("name = %q, want %q", reg.name, "test-service")
	}
	if reg.port != 8080 {
		t.Errorf("port = %d, want %d", reg.port, 8080)
	}
	if reg.russellAddr != "localhost:9100" {
		t.Errorf("russellAddr = %q, want %q", reg.russellAddr, "localhost:9100")
	}
	if reg.address != "localhost" {
		t.Errorf("address = %q, want %q", reg.address, "localhost")
	}
	if reg.version != "0.0.0" {
		t.Errorf("version = %q, want %q", reg.version, "0.0.0")
	}
	if reg.heartbeatInt != 10*time.Second {
		t.Errorf("heartbeatInt = %v, want %v", reg.heartbeatInt, 10*time.Second)
	}
}

func TestNew_WithAllFields(t *testing.T) {
	cfg := Config{
		Name:        "my-service",
		Version:     "1.2.3",
		Address:     "192.168.1.100",
		Port:        9999,
		RussellAddr: "russell.local:9100",
		Tags:        []string{"tag1", "tag2"},
		Metadata:    map[string]string{"key": "value"},
	}

	reg := New(cfg)

	if reg.name != "my-service" {
		t.Errorf("name = %q, want %q", reg.name, "my-service")
	}
	if reg.version != "1.2.3" {
		t.Errorf("version = %q, want %q", reg.version, "1.2.3")
	}
	if reg.address != "192.168.1.100" {
		t.Errorf("address = %q, want %q", reg.address, "192.168.1.100")
	}
	if reg.port != 9999 {
		t.Errorf("port = %d, want %d", reg.port, 9999)
	}
	if reg.russellAddr != "russell.local:9100" {
		t.Errorf("russellAddr = %q, want %q", reg.russellAddr, "russell.local:9100")
	}
	if len(reg.tags) != 2 || reg.tags[0] != "tag1" || reg.tags[1] != "tag2" {
		t.Errorf("tags = %v, want [tag1 tag2]", reg.tags)
	}
	if reg.metadata["key"] != "value" {
		t.Errorf("metadata[key] = %q, want %q", reg.metadata["key"], "value")
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantVersion string
		wantAddr    string
		wantRussell string
	}{
		{
			name:        "empty config uses defaults",
			cfg:         Config{},
			wantVersion: "0.0.0",
			wantAddr:    "localhost",
			wantRussell: "localhost:9100",
		},
		{
			name: "explicit empty version uses default",
			cfg: Config{
				Name:    "test",
				Version: "",
			},
			wantVersion: "0.0.0",
			wantAddr:    "localhost",
			wantRussell: "localhost:9100",
		},
		{
			name: "explicit version is preserved",
			cfg: Config{
				Name:    "test",
				Version: "2.0.0",
			},
			wantVersion: "2.0.0",
			wantAddr:    "localhost",
			wantRussell: "localhost:9100",
		},
		{
			name: "custom russell address",
			cfg: Config{
				Name:        "test",
				RussellAddr: "custom:1234",
			},
			wantVersion: "0.0.0",
			wantAddr:    "localhost",
			wantRussell: "custom:1234",
		},
		{
			name: "custom address",
			cfg: Config{
				Name:    "test",
				Address: "10.0.0.1",
			},
			wantVersion: "0.0.0",
			wantAddr:    "10.0.0.1",
			wantRussell: "localhost:9100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := New(tt.cfg)

			if reg.version != tt.wantVersion {
				t.Errorf("version = %q, want %q", reg.version, tt.wantVersion)
			}
			if reg.address != tt.wantAddr {
				t.Errorf("address = %q, want %q", reg.address, tt.wantAddr)
			}
			if reg.russellAddr != tt.wantRussell {
				t.Errorf("russellAddr = %q, want %q", reg.russellAddr, tt.wantRussell)
			}
		})
	}
}

func TestServiceRegistration_StopChannel(t *testing.T) {
	cfg := Config{
		Name: "test-service",
		Port: 8080,
	}

	reg := New(cfg)

	// Stop channel should be open
	select {
	case <-reg.stopCh:
		t.Error("stopCh should not be closed initially")
	default:
		// Expected
	}

	// StopHeartbeat should close the channel
	reg.StopHeartbeat()

	// Now channel should be closed
	select {
	case <-reg.stopCh:
		// Expected
	default:
		t.Error("stopCh should be closed after StopHeartbeat")
	}
}

func TestDeregister_WithEmptyServiceID(t *testing.T) {
	cfg := Config{
		Name: "test-service",
		Port: 8080,
	}

	reg := New(cfg)

	// Deregister without registration should return nil
	err := reg.Deregister(context.Background())
	if err != nil {
		t.Errorf("Deregister with empty serviceID returned error: %v", err)
	}
}

func TestRegister_ConnectionFailure(t *testing.T) {
	cfg := Config{
		Name:        "test-service",
		Port:        8080,
		RussellAddr: "invalid:99999", // Non-existent address
	}

	reg := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := reg.Register(ctx)
	if err == nil {
		t.Error("Register to invalid address should return error")
	}
}

func TestSendHeartbeat_WithEmptyServiceID(t *testing.T) {
	cfg := Config{
		Name: "test-service",
		Port: 8080,
	}

	reg := New(cfg)

	// Should not panic with empty serviceID
	reg.sendHeartbeat(context.Background())
}

func TestRegisterService_ConnectionFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := RegisterService(ctx, "test", "1.0.0", 8080, "invalid:99999")
	if err == nil {
		t.Error("RegisterService to invalid address should return error")
	}
}
