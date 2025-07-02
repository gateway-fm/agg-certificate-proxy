package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gateway-fm/agg-certificate-proxy/internal/certificate"
	proxyhealth "github.com/gateway-fm/agg-certificate-proxy/internal/health"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

func main() {
	// Configuration flags
	grpcAddr := flag.String("grpc", ":50051", "gRPC server address")
	httpAddr := flag.String("http", ":8080", "HTTP server address")
	dbPath := flag.String("db", "certificates.db", "SQLite database path")
	delayedChainsStr := flag.String("delayed-chains", "1,137", "Comma-separated list of chain IDs to delay")
	delayStr := flag.String("delay", "48h", "Delay duration for certificate processing (e.g., 48h, 30m, 2h15m)")
	aggsenderAddr := flag.String("aggsender-addr", "", "Address of the aggsender to forward certificates to")
	schedulerInterval := flag.String("scheduler-interval", "30s", "How often to check for pending certificates (e.g., 30s, 1m)")
	killSwitchAPIKey := flag.String("kill-switch-api-key", "", "API key for kill switch endpoint")
	killRestartAPIKey := flag.String("kill-restart-api-key", "", "API key for restart endpoint")
	flag.Parse()

	// Create root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse scheduler interval
	interval, err := time.ParseDuration(*schedulerInterval)
	if err != nil {
		log.Fatalf("Invalid scheduler interval: %v", err)
	}

	// Initialize database
	db, err := certificate.NewSqliteStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("failed to close database", "err", closeErr)
		}
	}()

	// Create certificate service
	service := certificate.NewService(db)

	// Create health service with root context
	healthService := proxyhealth.NewService(ctx)

	// Store API keys if provided
	if killSwitchAPIKey == nil || len(*killSwitchAPIKey) == 0 {
		slog.Error("no kill switch API key provided - cannot start")
		return
	}
	if err = hashAndStoreKey(db, "kill_switch_api_key", *killSwitchAPIKey); err != nil {
		slog.Error("failed to hash kill switch API key", "err", err)
		return
	}

	if killRestartAPIKey == nil || len(*killRestartAPIKey) == 0 {
		slog.Error("no kill restart API key provided - cannot start")
		return
	}
	if err = hashAndStoreKey(db, "kill_restart_api_key", *killRestartAPIKey); err != nil {
		slog.Error("failed to hash kill restart API key", "err", err)
		return
	}

	// Update aggsender address if provided
	if *aggsenderAddr != "" {
		if err := db.SetConfigValue("aggsender_address", *aggsenderAddr); err != nil {
			slog.Error("failed to set aggsender address", "err", err)
		} else {
			slog.Info("aggsender address set", "newVal", *aggsenderAddr)
		}
	}

	// Update delay if provided
	if *delayStr != "" {
		duration, err := time.ParseDuration(*delayStr)
		if err != nil {
			slog.Error("invalid delay duration", "val", *delayStr, "err", err)
			return
		}

		// Store as seconds in the database
		seconds := int(duration.Seconds())
		if err := db.SetConfigValue("delay_seconds", strconv.Itoa(seconds)); err != nil {
			slog.Error("failed to set delay", "err", err)
		} else {
			slog.Info("updated delay", "duration", duration, "seconds", seconds)
		}
	}

	// Update delayed chains if provided
	if *delayedChainsStr != "" {
		chains := parseChainIDs(*delayedChainsStr)
		if len(chains) > 0 {
			if err := service.SetDelayedChains(chains); err != nil {
				slog.Error("failed to set delayed chains", "err", err)
			} else {
				slog.Info("updated delayed chains", "val", chains)
			}
		}
	}

	// Log current configuration
	currentChains, err := service.GetDelayedChains()
	if err == nil {
		slog.Info("configured delayed chains", "val", currentChains)
	}

	// Get delay and display in human-readable format
	delaySeconds, err := service.GetConfigValue("delay_seconds")
	if err != nil {
		slog.Error("failed to get delay_seconds from config", "err", err)
		return
	} else {
		if seconds, err := strconv.Atoi(delaySeconds); err == nil {
			duration := time.Duration(seconds) * time.Second
			slog.Info("found delay", "val", duration)
		}
	}

	// Start scheduler for processing delayed certificates
	scheduler, err := certificate.NewScheduler(ctx, service, interval)
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}
	go scheduler.Start()

	// Create and register gRPC server
	grpcServer := grpc.NewServer()
	certGrpcServer := certificate.NewGRPCServer(service)
	certGrpcServer.Register(grpcServer)

	// Start gRPC server in goroutine
	go func() {
		slog.Info("starting gRPC server", "address", *grpcAddr)
		if err := startGRPCServer(grpcServer, *grpcAddr); err != nil {
			log.Fatalf("failed to start gRPC server: %v", err)
		}
	}()

	// Create and register HTTP handlers
	// First register certificate API handlers
	apiServer := certificate.NewAPIServer(service)
	apiServer.RegisterHandlers()

	// Then register health API handlers
	healthApi := proxyhealth.NewApi(healthService)
	healthApi.RegisterHandlers()

	// Start HTTP server with cancellation context
	httpServer := &http.Server{
		Addr:    *httpAddr,
		Handler: http.DefaultServeMux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		slog.Info("http server listening", "address", *httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either context cancellation or signal
	select {
	case <-sigCh:
		slog.Info("received shutdown signal")
	case <-ctx.Done():
		slog.Info("context cancelled")
	}

	slog.Info("shutting down...")

	// Cancel the root context to signal all components
	cancel()

	// Give components a moment to react to context cancellation
	time.Sleep(100 * time.Millisecond)

	// Create a shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Create a channel to coordinate shutdown steps
	shutdownComplete := make(chan struct{})

	go func() {
		// Wait for scheduler to finish (it will stop automatically via context)
		slog.Info("waiting for scheduler to complete shutdown...")
		// The scheduler's Start() method monitors context and calls stop() automatically

		// Give scheduler time to finish its tasks
		// In a production system, you might want to track scheduler completion
		time.Sleep(2 * time.Second)

		// Graceful shutdown of gRPC server
		slog.Info("shutting down gRPC server...")
		grpcServer.GracefulStop()
		slog.Info("gRPC server shut down")

		// Graceful shutdown of HTTP server
		// This will wait for all active HTTP requests to complete
		slog.Info("shutting down HTTP server...")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "err", err)
		}
		slog.Info("HTTP server shut down")

		close(shutdownComplete)
	}()

	// Wait for shutdown to complete or timeout
	select {
	case <-shutdownComplete:
		slog.Info("graceful shutdown completed")
	case <-shutdownCtx.Done():
		slog.Warn("shutdown timeout exceeded, forcing shutdown")
		grpcServer.Stop() // Force stop gRPC if still running
	}

	slog.Info("shutdown complete")
}

func parseChainIDs(chainsStr string) []uint32 {
	parts := strings.Split(chainsStr, ",")
	chains := make([]uint32, 0, len(parts))
	for _, part := range parts {
		chainID, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			slog.Warn("invalid chain ID", "id", part)
			continue
		}
		chains = append(chains, uint32(chainID))
	}
	return chains
}

func startGRPCServer(server *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	slog.Info("gRPC proxy listening", "address", addr)
	return server.Serve(lis)
}

func hashAndStoreKey(db certificate.Db, dbKey string, key string) error {
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := db.SetCredential(dbKey, string(hashedKey)); err != nil {
		return err
	}
	return nil
}
