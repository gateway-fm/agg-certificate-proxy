package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/gateway-fm/agg-certificate-proxy/internal/certificate"
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
	defer db.Close()

	// Create service
	service := certificate.NewService(db)

	// Store API keys if provided
	if *killSwitchAPIKey != "" {
		if err := db.SetCredential("kill_switch_api_key", *killSwitchAPIKey); err != nil {
			log.Printf("Failed to set kill switch API key: %v", err)
		} else {
			log.Printf("Kill switch API key configured")
		}
	}

	if *killRestartAPIKey != "" {
		if err := db.SetCredential("kill_restart_api_key", *killRestartAPIKey); err != nil {
			log.Printf("Failed to set kill restart API key: %v", err)
		} else {
			log.Printf("Kill restart API key configured")
		}
	}

	// Update aggsender address if provided
	if *aggsenderAddr != "" {
		if err := db.SetConfigValue("aggsender_address", *aggsenderAddr); err != nil {
			log.Printf("Failed to set aggsender address: %v", err)
		} else {
			log.Printf("Aggsender address set to: %s", *aggsenderAddr)
		}
	}

	// Update delay if provided
	if *delayStr != "" {
		duration, err := time.ParseDuration(*delayStr)
		if err != nil {
			log.Fatalf("Invalid delay duration '%s': %v", *delayStr, err)
		}

		// Store as seconds in the database
		seconds := int(duration.Seconds())
		if err := db.SetConfigValue("delay_seconds", strconv.Itoa(seconds)); err != nil {
			log.Printf("Failed to set delay: %v", err)
		} else {
			log.Printf("Updated delay to %s (%d seconds)", duration, seconds)
		}
	}

	// Update delayed chains if provided
	if *delayedChainsStr != "" {
		chains := parseChainIDs(*delayedChainsStr)
		if len(chains) > 0 {
			if err := service.SetDelayedChains(chains); err != nil {
				log.Printf("Failed to set delayed chains: %v", err)
			} else {
				log.Printf("Updated delayed chains to: %v", chains)
			}
		}
	}

	// Log current configuration
	currentChains, err := service.GetDelayedChains()
	if err == nil {
		log.Printf("Delayed chains: %v", currentChains)
	}

	// Get delay and display in human-readable format
	delaySeconds, err := service.GetConfigValue("delay_seconds")
	if err == nil && delaySeconds != "" {
		if seconds, err := strconv.Atoi(delaySeconds); err == nil {
			duration := time.Duration(seconds) * time.Second
			log.Printf("Delay: %s", duration)
		}
	} else {
		// Check old delay_hours for backward compatibility
		delayHours, err := service.GetConfigValue("delay_hours")
		if err == nil && delayHours != "" {
			if hours, err := strconv.Atoi(delayHours); err == nil {
				seconds := hours * 3600
				// Migrate to delay_seconds
				db.SetConfigValue("delay_seconds", strconv.Itoa(seconds))
				log.Printf("Migrated delay from %s hours to %d seconds", delayHours, seconds)
				log.Printf("Delay: %s", time.Duration(seconds)*time.Second)
			}
		}
	}

	// Start scheduler for processing delayed certificates
	scheduler, err := certificate.NewScheduler(service, interval)
	if err != nil {
		log.Fatalf("Failed to create scheduler: %v", err)
	}
	go scheduler.Start()
	defer scheduler.Stop()

	// Create and register gRPC server
	grpcServer := grpc.NewServer()
	certGrpcServer := certificate.NewGRPCServer(service)
	certGrpcServer.Register(grpcServer)

	// Start gRPC server in goroutine
	go func() {
		log.Printf("Starting gRPC server on %s", *grpcAddr)
		if err := startGRPCServer(grpcServer, *grpcAddr); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Create and start HTTP server
	apiServer := certificate.NewAPIServer(service)
	apiServer.RegisterHandlers()
	go apiServer.Start(*httpAddr)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Graceful shutdown
	stopCh := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopCh)
	}()

	select {
	case <-stopCh:
		log.Println("gRPC server shut down gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}

	log.Println("Shutdown complete")
}

func parseChainIDs(chainsStr string) []uint32 {
	parts := strings.Split(chainsStr, ",")
	chains := make([]uint32, 0, len(parts))
	for _, part := range parts {
		chainID, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			log.Printf("Invalid chain ID: %s", part)
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
	log.Printf("gRPC proxy listening on %s", addr)
	return server.Serve(lis)
}
