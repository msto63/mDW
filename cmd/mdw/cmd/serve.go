package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	turingpb "github.com/msto63/mDW/api/gen/turing"
	aristotelesServer "github.com/msto63/mDW/internal/aristoteles/server"
	babbageServer "github.com/msto63/mDW/internal/babbage/server"
	bayesServer "github.com/msto63/mDW/internal/bayes/server"
	hypatiaServer "github.com/msto63/mDW/internal/hypatia/server"
	hypatiaService "github.com/msto63/mDW/internal/hypatia/service"
	kantServer "github.com/msto63/mDW/internal/kant/server"
	leibnizAgent "github.com/msto63/mDW/internal/leibniz/agent"
	"github.com/msto63/mDW/internal/leibniz/agentloader"
	leibnizServer "github.com/msto63/mDW/internal/leibniz/server"
	"github.com/msto63/mDW/internal/leibniz/servicetools"
	platonServer "github.com/msto63/mDW/internal/platon/server"
	russellServer "github.com/msto63/mDW/internal/russell/server"
	turingServer "github.com/msto63/mDW/internal/turing/server"
	"github.com/msto63/mDW/pkg/core/bayeslog"
	"github.com/msto63/mDW/pkg/core/config"
	"github.com/msto63/mDW/pkg/core/registration"
	"github.com/msto63/mDW/pkg/core/version"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var serveDaemon bool
var appConfig *config.Config

var serveCmd = &cobra.Command{
	Use:   "serve [service]",
	Short: "Startet einen oder alle Services",
	Long: `Startet meinDENKWERK Services.

Ohne Argument werden alle Services gestartet.
Mit Argument wird nur der angegebene Service gestartet.

Services:
  kant        - API Gateway (HTTP :8080)
  russell     - Orchestrierung (gRPC :9100)
  turing      - LLM Service (gRPC :9200)
  hypatia     - RAG Service (gRPC :9220)
  babbage     - NLP Service (gRPC :9150)
  leibniz     - Agentic AI (gRPC :9140)
  platon      - Pipeline Service (gRPC :9130)
  aristoteles - Agentic Pipeline (gRPC :9160)
  bayes       - Logging (gRPC :9120)

Beispiele:
  mdw serve            # Alle Services starten
  mdw serve kant       # Nur API Gateway starten
  mdw serve aristoteles # Nur Agentic Pipeline starten`,
	ValidArgs: []string{"kant", "russell", "turing", "hypatia", "leibniz", "babbage", "bayes", "platon", "aristoteles"},
	Args:      cobra.MaximumNArgs(1),
	RunE:      runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolVarP(&serveDaemon, "daemon", "d", false, "Als Daemon im Hintergrund starten")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load central configuration
	var err error
	appConfig, err = config.LoadFromEnv()
	if err != nil {
		fmt.Printf("Warnung: Config nicht geladen (%v), nutze Defaults\n", err)
		// Create minimal default config
		appConfig = &config.Config{}
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if len(args) == 0 {
		return startAllServices(ctx, sigCh)
	}
	return startSingleService(ctx, sigCh, args[0])
}

func startAllServices(ctx context.Context, sigCh chan os.Signal) error {
	fmt.Println("meinDENKWERK")
	fmt.Println("============")
	fmt.Println("Starte alle Services...")
	fmt.Println()

	var wg sync.WaitGroup
	errCh := make(chan error, 9)

	// Start services in order
	services := []struct {
		name  string
		start func(context.Context) error
	}{
		{"bayes", startBayes},
		{"russell", startRussell},
		{"turing", startTuring},
		{"hypatia", startHypatia},
		{"babbage", startBabbage},
		{"platon", startPlaton},
		{"leibniz", startLeibniz},
		{"aristoteles", startAristoteles},
		{"kant", startKant},
	}

	for _, svc := range services {
		wg.Add(1)
		go func(name string, startFn func(context.Context) error) {
			defer wg.Done()
			if err := startFn(ctx); err != nil {
				errCh <- fmt.Errorf("%s: %v", name, err)
			}
		}(svc.name, svc.start)

		// Small delay between services
		time.Sleep(100 * time.Millisecond)
	}

	// Initialize central logging after Bayes is up
	time.Sleep(500 * time.Millisecond)
	bayesPort := 9120
	if appConfig != nil && appConfig.Bayes.Port != 0 {
		bayesPort = appConfig.Bayes.Port
	}
	bayesAddr := fmt.Sprintf("localhost:%d", bayesPort)
	if err := bayeslog.Init(ctx, "mdw-serve", bayesAddr); err != nil {
		fmt.Printf("  [!] Bayes-Logging nicht verfügbar: %v\n", err)
	} else {
		fmt.Println("  [+] Zentrales Logging aktiviert")
	}

	fmt.Println()
	fmt.Println("Alle Services gestartet!")
	fmt.Println("Drücke Ctrl+C zum Beenden")
	fmt.Println()
	fmt.Println("API Gateway: http://localhost:8080")
	fmt.Println("Health Check: http://localhost:8080/api/v1/health")

	// Wait for signal or error
	select {
	case <-sigCh:
		fmt.Println("\nStoppe Services...")
	case err := <-errCh:
		fmt.Printf("Service-Fehler: %v\n", err)
	}

	return nil
}

func startSingleService(ctx context.Context, sigCh chan os.Signal, name string) error {
	fmt.Printf("Starte Service: %s\n", name)

	var startFn func(context.Context) error

	switch name {
	case "kant":
		startFn = startKant
	case "russell":
		startFn = startRussell
	case "turing":
		startFn = startTuring
	case "hypatia":
		startFn = startHypatia
	case "babbage":
		startFn = startBabbage
	case "leibniz":
		startFn = startLeibniz
	case "platon":
		startFn = startPlaton
	case "aristoteles":
		startFn = startAristoteles
	case "bayes":
		startFn = startBayes
	default:
		return fmt.Errorf("unbekannter Service: %s", name)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- startFn(ctx)
	}()

	select {
	case <-sigCh:
		fmt.Println("\nStoppe Service...")
		return nil
	case err := <-errCh:
		return err
	}
}

func startKant(ctx context.Context) error {
	cfg := kantServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Kant.Port != 0 {
		cfg.HTTPPort = appConfig.Kant.Port
		cfg.Host = appConfig.Kant.Host
		cfg.ReadTimeout = appConfig.Kant.ReadTimeout.Duration
		cfg.WriteTimeout = appConfig.Kant.WriteTimeout.Duration
	}
	srv, err := kantServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Kant (API Gateway) auf :%d\n", cfg.HTTPPort)
	return srv.Start()
}

func startRussell(ctx context.Context) error {
	cfg := russellServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Russell.Port != 0 {
		cfg.Port = appConfig.Russell.Port
		cfg.Host = appConfig.Russell.Host
	}
	srv, err := russellServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Russell (Orchestrierung) auf :%d\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}
	<-ctx.Done()
	srv.Stop(context.Background())
	return nil
}

func startTuring(ctx context.Context) error {
	cfg := turingServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Turing.Port != 0 {
		cfg.Port = appConfig.Turing.Port
		cfg.Host = appConfig.Turing.Host
		if appConfig.Turing.Providers.Ollama.BaseURL != "" {
			cfg.OllamaURL = appConfig.Turing.Providers.Ollama.BaseURL
		}
		if appConfig.Turing.Timeout.Duration > 0 {
			cfg.OllamaTimeout = appConfig.Turing.Timeout.Duration
		}
		if appConfig.Turing.DefaultModel != "" {
			cfg.DefaultModel = appConfig.Turing.DefaultModel
		}
	}
	srv, err := turingServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Turing (LLM) auf :%d\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "turing", version.Turing, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Turing: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}

func startHypatia(ctx context.Context) error {
	cfg := hypatiaServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Hypatia.Port != 0 {
		cfg.Port = appConfig.Hypatia.Port
		cfg.Host = appConfig.Hypatia.Host
		cfg.ChunkSize = appConfig.Hypatia.Chunking.DefaultSize
		cfg.ChunkOverlap = appConfig.Hypatia.Chunking.DefaultOverlap
		cfg.DefaultTopK = appConfig.Hypatia.DefaultTopK
		cfg.MinRelevance = float64(appConfig.Hypatia.MinRelevanceScore)
		if appConfig.Hypatia.VectorStore.Type != "" {
			cfg.VectorStoreType = appConfig.Hypatia.VectorStore.Type
		}
		if appConfig.Hypatia.VectorStore.Path != "" {
			cfg.VectorStorePath = appConfig.Hypatia.VectorStore.Path
		}
	}
	srv, err := hypatiaServer.New(cfg)
	if err != nil {
		return err
	}

	// Connect to Turing for embedding functionality
	turingPort := 9200
	if appConfig != nil && appConfig.Turing.Port != 0 {
		turingPort = appConfig.Turing.Port
	}
	turingAddr := fmt.Sprintf("localhost:%d", turingPort)

	// Create embedding function that calls Turing
	var embeddingFunc hypatiaService.EmbeddingFunc = func(ctx context.Context, texts []string) ([][]float64, error) {
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(dialCtx, turingAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Turing: %w", err)
		}
		defer conn.Close()

		client := turingpb.NewTuringServiceClient(conn)

		// Use BatchEmbed for multiple texts
		resp, err := client.BatchEmbed(ctx, &turingpb.BatchEmbedRequest{
			Model:  "nomic-embed-text",
			Inputs: texts,
		})
		if err != nil {
			return nil, fmt.Errorf("embedding failed: %w", err)
		}

		// Convert response to [][]float64
		embeddings := make([][]float64, len(resp.Embeddings))
		for i, emb := range resp.Embeddings {
			embeddings[i] = make([]float64, len(emb.Embedding))
			for j, v := range emb.Embedding {
				embeddings[i][j] = float64(v)
			}
		}
		return embeddings, nil
	}

	srv.SetEmbeddingFunc(embeddingFunc)

	fmt.Printf("  [+] Hypatia (RAG) auf :%d (→ Turing Embed)\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "hypatia", version.Hypatia, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Hypatia: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}

func startBabbage(ctx context.Context) error {
	cfg := babbageServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Babbage.Port != 0 {
		cfg.Port = appConfig.Babbage.Port
		cfg.Host = appConfig.Babbage.Host
	}
	srv, err := babbageServer.New(cfg)
	if err != nil {
		return err
	}

	// Connect to Turing for LLM functionality (needed for Translate)
	turingPort := 9200
	if appConfig != nil && appConfig.Turing.Port != 0 {
		turingPort = appConfig.Turing.Port
	}
	turingAddr := fmt.Sprintf("localhost:%d", turingPort)

	// Get default model
	defaultModel := "mistral:7b"
	if appConfig != nil && appConfig.Turing.DefaultModel != "" {
		defaultModel = appConfig.Turing.DefaultModel
	}

	// Create LLM function that calls Turing
	llmFunc := func(ctx context.Context, prompt string) (string, error) {
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(dialCtx, turingAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return "", fmt.Errorf("failed to connect to Turing: %w", err)
		}
		defer conn.Close()

		client := turingpb.NewTuringServiceClient(conn)

		resp, err := client.Chat(ctx, &turingpb.ChatRequest{
			Model: defaultModel,
			Messages: []*turingpb.Message{
				{Role: "user", Content: prompt},
			},
		})
		if err != nil {
			return "", fmt.Errorf("chat failed: %w", err)
		}

		return resp.Content, nil
	}

	srv.SetLLMFunc(llmFunc)

	fmt.Printf("  [+] Babbage (NLP) auf :%d\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "babbage", version.Babbage, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Babbage: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}

func startLeibniz(ctx context.Context) error {
	cfg := leibnizServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Leibniz.Port != 0 {
		cfg.Port = appConfig.Leibniz.Port
		cfg.Host = appConfig.Leibniz.Host
		cfg.MaxSteps = appConfig.Leibniz.MaxIterations
		cfg.Timeout = appConfig.Leibniz.DefaultTimeout.Duration
	}
	srv, err := leibnizServer.New(cfg)
	if err != nil {
		return err
	}

	// Connect to Turing for LLM functionality
	turingPort := 9200
	if appConfig != nil && appConfig.Turing.Port != 0 {
		turingPort = appConfig.Turing.Port
	}
	turingAddr := fmt.Sprintf("localhost:%d", turingPort)

	// Get default model
	defaultModel := "mistral:7b"
	if appConfig != nil && appConfig.Turing.DefaultModel != "" {
		defaultModel = appConfig.Turing.DefaultModel
	}

	// Create model-aware LLM function that calls Turing
	// This allows agents to use different models based on their specialization
	modelAwareLLMFunc := func(ctx context.Context, model string, messages []leibnizAgent.Message) (string, error) {
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(dialCtx, turingAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return "", fmt.Errorf("failed to connect to Turing: %w", err)
		}
		defer conn.Close()

		client := turingpb.NewTuringServiceClient(conn)

		// Convert messages to proto format
		protoMessages := make([]*turingpb.Message, len(messages))
		for i, msg := range messages {
			protoMessages[i] = &turingpb.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		// Use agent-specific model if provided, otherwise fall back to default
		requestModel := defaultModel
		if model != "" {
			requestModel = model
		}

		resp, err := client.Chat(ctx, &turingpb.ChatRequest{
			Model:    requestModel,
			Messages: protoMessages,
		})
		if err != nil {
			return "", fmt.Errorf("chat failed: %w", err)
		}

		return resp.Content, nil
	}

	// Create a wrapper for backward compatibility with LLMFunc interface
	llmFunc := func(ctx context.Context, messages []leibnizAgent.Message) (string, error) {
		return modelAwareLLMFunc(ctx, "", messages)
	}

	srv.SetLLMFunc(llmFunc)
	srv.SetModelAwareLLMFunc(modelAwareLLMFunc)

	// Register service tools (RAG, NLP)
	hypatiaPort := 9220
	if appConfig != nil && appConfig.Hypatia.Port != 0 {
		hypatiaPort = appConfig.Hypatia.Port
	}
	babbagePort := 9150
	if appConfig != nil && appConfig.Babbage.Port != 0 {
		babbagePort = appConfig.Babbage.Port
	}

	svcTools := servicetools.New(servicetools.Config{
		HypatiaAddr: fmt.Sprintf("localhost:%d", hypatiaPort),
		BabbageAddr: fmt.Sprintf("localhost:%d", babbagePort),
	})
	svcTools.RegisterAll(srv.Agent())

	// Create embedding function for agent matching (RAG-style agent selection)
	// This enables dynamic agent selection based on task similarity
	embeddingFunc := func(ctx context.Context, texts []string) ([][]float64, error) {
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		conn, err := grpc.DialContext(dialCtx, turingAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Turing for embeddings: %w", err)
		}
		defer conn.Close()

		client := turingpb.NewTuringServiceClient(conn)

		// Use BatchEmbed for multiple texts
		resp, err := client.BatchEmbed(ctx, &turingpb.BatchEmbedRequest{
			Model:  "nomic-embed-text",
			Inputs: texts,
		})
		if err != nil {
			return nil, fmt.Errorf("embedding failed: %w", err)
		}

		// Convert response to [][]float64
		embeddings := make([][]float64, len(resp.Embeddings))
		for i, emb := range resp.Embeddings {
			embeddings[i] = make([]float64, len(emb.Embedding))
			for j, v := range emb.Embedding {
				embeddings[i][j] = float64(v)
			}
		}
		return embeddings, nil
	}

	// Set embedding function on Leibniz service for intelligent agent matching
	srv.Service().SetEmbeddingFunc(agentloader.EmbeddingFunc(embeddingFunc))

	// Pipeline processing is now handled by Platon service
	// Leibniz connects to Platon via gRPC for pre-/post-processing

	fmt.Printf("  [+] Leibniz (Agentic AI) auf :%d (→ Turing, Hypatia, Babbage, Platon, Agent-Embeddings)\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "leibniz", version.Leibniz, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Leibniz: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}

func startBayes(ctx context.Context) error {
	cfg := bayesServer.DefaultConfig()
	// Apply central config
	if appConfig != nil && appConfig.Bayes.Port != 0 {
		cfg.Port = appConfig.Bayes.Port
		cfg.Host = appConfig.Bayes.Host
		if appConfig.Bayes.StoragePath != "" {
			cfg.Service.LogDir = appConfig.Bayes.StoragePath
		}
	}
	srv, err := bayesServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Bayes (Logging) auf :%d\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}
	<-ctx.Done()
	srv.Stop(context.Background())
	return nil
}

func startPlaton(ctx context.Context) error {
	cfg := platonServer.DefaultConfig()
	// Apply central config if available
	// Note: Platon config not yet in central config, using defaults

	srv, err := platonServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Platon (Pipeline) auf :%d\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "platon", version.Platon, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Platon: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}

func startAristoteles(ctx context.Context) error {
	cfg := aristotelesServer.DefaultConfig()
	// Apply central config if available
	// Service addresses from central config
	turingPort := 9200
	if appConfig != nil && appConfig.Turing.Port != 0 {
		turingPort = appConfig.Turing.Port
	}
	cfg.TuringAddr = fmt.Sprintf("localhost:%d", turingPort)

	leibnizPort := 9140
	if appConfig != nil && appConfig.Leibniz.Port != 0 {
		leibnizPort = appConfig.Leibniz.Port
	}
	cfg.LeibnizAddr = fmt.Sprintf("localhost:%d", leibnizPort)

	hypatiaPort := 9220
	if appConfig != nil && appConfig.Hypatia.Port != 0 {
		hypatiaPort = appConfig.Hypatia.Port
	}
	cfg.HypatiaAddr = fmt.Sprintf("localhost:%d", hypatiaPort)

	babbagePort := 9150
	if appConfig != nil && appConfig.Babbage.Port != 0 {
		babbagePort = appConfig.Babbage.Port
	}
	cfg.BabbageAddr = fmt.Sprintf("localhost:%d", babbagePort)

	platonPort := 9130
	// Platon port not in central config yet, using default
	cfg.PlatonAddr = fmt.Sprintf("localhost:%d", platonPort)

	srv, err := aristotelesServer.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("  [+] Aristoteles (Agentic Pipeline) auf :%d (→ Turing, Leibniz, Hypatia, Babbage, Platon)\n", cfg.Port)
	if err := srv.StartAsync(); err != nil {
		return err
	}

	// Register with Russell
	russellPort := 9100
	if appConfig != nil && appConfig.Russell.Port != 0 {
		russellPort = appConfig.Russell.Port
	}
	reg, err := registration.RegisterService(ctx, "aristoteles", version.Aristoteles, cfg.Port, fmt.Sprintf("localhost:%d", russellPort))
	if err != nil {
		fmt.Printf("  [!] Aristoteles: Russell-Registrierung fehlgeschlagen: %v\n", err)
	}

	<-ctx.Done()
	if reg != nil {
		reg.StopHeartbeat()
		reg.Deregister(context.Background())
	}
	srv.Stop(context.Background())
	return nil
}
