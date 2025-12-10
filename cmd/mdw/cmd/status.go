package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	commonpb "github.com/msto63/mDW/api/gen/common"
	russellpb "github.com/msto63/mDW/api/gen/russell"
	"github.com/msto63/mDW/internal/turing/ollama"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Zeigt den Status aller Services",
	Long: `Zeigt den Status aller meinDENKWERK Services an.

Prüft die Erreichbarkeit und den Gesundheitszustand jedes Services.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("meinDENKWERK Status")
	fmt.Println("===================")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Define services with correct ports from CLAUDE.md
	services := []struct {
		name     string
		port     int
		protocol string
		check    func(context.Context) (string, error)
	}{
		{"Kant (API Gateway)", 8080, "HTTP", checkHTTP(8080)},
		{"Russell (Discovery)", 9100, "gRPC", checkGRPC(9100)},
		{"Bayes (Logging)", 9120, "gRPC", checkGRPC(9120)},
		{"Leibniz (Agent)", 9140, "gRPC", checkGRPC(9140)},
		{"Babbage (NLP)", 9150, "gRPC", checkGRPC(9150)},
		{"Turing (LLM)", 9200, "gRPC", checkGRPC(9200)},
		{"Hypatia (RAG)", 9220, "gRPC", checkGRPC(9220)},
	}

	fmt.Println("Services:")
	fmt.Println("---------")

	allHealthy := true
	for _, svc := range services {
		status, err := svc.check(ctx)
		statusIcon := "[+]"
		statusText := "running"

		if err != nil {
			statusIcon = "[-]"
			statusText = "stopped"
			allHealthy = false
		} else if status != "" {
			statusText = status
		}

		fmt.Printf("  %s %-25s :%d (%s) - %s\n",
			statusIcon, svc.name, svc.port, svc.protocol, statusText)
	}

	// Try to get detailed status from Russell if available
	russellStatus, err := getRussellStatus(ctx)
	if err == nil && len(russellStatus) > 0 {
		fmt.Println()
		fmt.Println("Registrierte Services (via Russell):")
		fmt.Println("-------------------------------------")
		for name, status := range russellStatus {
			icon := "[+]"
			if status != "healthy" {
				icon = "[-]"
			}
			fmt.Printf("  %s %s: %s\n", icon, name, status)
		}
	}

	// Check external dependencies
	fmt.Println()
	fmt.Println("Externe Abhängigkeiten:")
	fmt.Println("-----------------------")

	// Ollama
	ollamaClient := ollama.NewClient(ollama.DefaultConfig())
	if err := ollamaClient.Ping(ctx); err != nil {
		fmt.Println("  [-] Ollama                      - nicht erreichbar")
		fmt.Println("      Start mit: ollama serve")
	} else {
		models, _ := ollamaClient.ListModels(ctx)
		fmt.Printf("  [+] Ollama                      - %d Modell(e) verfügbar\n", len(models.Models))
	}

	fmt.Println()

	if allHealthy {
		fmt.Println("Alle Services sind aktiv.")
	} else {
		fmt.Println("Einige Services sind nicht aktiv.")
		fmt.Println("Starte mit: mdw serve")
	}

	return nil
}

func checkHTTP(port int) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		url := fmt.Sprintf("http://localhost:%d/api/v1/health", port)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return "healthy", nil
		}
		return fmt.Sprintf("status %d", resp.StatusCode), nil
	}
}

func checkTCP(port int) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		addr := fmt.Sprintf("localhost:%d", port)

		dialer := &net.Dialer{Timeout: 2 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return "", err
		}
		conn.Close()

		return "listening", nil
	}
}

func checkGRPC(port int) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		addr := fmt.Sprintf("localhost:%d", port)

		// Quick TCP check first
		dialer := &net.Dialer{Timeout: 1 * time.Second}
		tcpConn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return "", err
		}
		tcpConn.Close()

		// Try to establish gRPC connection
		conn, err := grpc.DialContext(ctx, addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return "listening", nil // TCP works but gRPC handshake failed
		}
		defer conn.Close()

		return "healthy", nil
	}
}

func getRussellStatus(ctx context.Context) (map[string]string, error) {
	addr := fmt.Sprintf("localhost:%d", ServicePorts.Russell)

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := russellpb.NewRussellServiceClient(conn)

	resp, err := client.ListServices(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}

	status := make(map[string]string)
	for _, svc := range resp.Services {
		status[svc.Name] = svc.Status.String()
	}

	return status, nil
}
