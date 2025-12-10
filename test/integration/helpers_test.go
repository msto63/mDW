package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Test configuration from environment or defaults
type TestConfig struct {
	TuringAddr  string
	HypatiaAddr string
	LeibnizAddr string
	RussellAddr string
	BabbageAddr string
	BayesAddr   string
	KantAddr    string
	PlatonAddr  string
	OllamaAddr  string
}

func getTestConfig() TestConfig {
	return TestConfig{
		TuringAddr:  getEnv("TEST_TURING_ADDR", "localhost:9200"),
		HypatiaAddr: getEnv("TEST_HYPATIA_ADDR", "localhost:9220"),
		LeibnizAddr: getEnv("TEST_LEIBNIZ_ADDR", "localhost:9140"),
		RussellAddr: getEnv("TEST_RUSSELL_ADDR", "localhost:9100"),
		BabbageAddr: getEnv("TEST_BABBAGE_ADDR", "localhost:9150"),
		BayesAddr:   getEnv("TEST_BAYES_ADDR", "localhost:9120"),
		KantAddr:    getEnv("TEST_KANT_ADDR", "localhost:8080"),
		PlatonAddr:  getEnv("TEST_PLATON_ADDR", "localhost:9130"),
		OllamaAddr:  getEnv("TEST_OLLAMA_ADDR", "localhost:11434"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// skipIfServiceUnavailable skips the test if the service is not reachable
func skipIfServiceUnavailable(t *testing.T, addr string, serviceName string) {
	t.Helper()
	if !isServiceAvailable(addr) {
		t.Skipf("Skipping: %s service not available at %s", serviceName, addr)
	}
}

// isServiceAvailable checks if a TCP connection can be established
func isServiceAvailable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// dialGRPC creates a gRPC connection with timeout
func dialGRPC(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("Failed to connect to %s: %v", addr, err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

// testContext returns a context with timeout for tests
func testContext(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}

// requireNoError fails the test if err is not nil
func requireNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// requireTrue fails the test if condition is false
func requireTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("Expected true: %s", msg)
	}
}

// requireEqual fails the test if expected != actual
func requireEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// requireNotEmpty fails the test if value is empty
func requireNotEmpty(t *testing.T, value string, msg string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s: expected non-empty string", msg)
	}
}

// logTestStart logs the start of a test with service info
func logTestStart(t *testing.T, serviceName, testName string) {
	t.Helper()
	t.Logf("=== %s: %s ===", serviceName, testName)
}

// waitForService waits for a service to become available
func waitForService(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isServiceAvailable(addr) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("service at %s not available after %v", addr, timeout)
}
