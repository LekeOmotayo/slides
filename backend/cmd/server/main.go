package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"leke-sliding-window-limiter/backend/internal/api/handlers"
	"leke-sliding-window-limiter/backend/internal/api/repositories"
	"leke-sliding-window-limiter/backend/internal/api/services"
	"leke-sliding-window-limiter/backend/internal/domain/limiter"
)

func main() {
	serve := flag.Bool("serve", true, "Start interactive web server on port 8096")
	port := flag.String("port", "8096", "Port for web server")
	flag.Parse()

	if *serve {
		startServer(*port)
	} else {
		runCliDemo()
	}
}

func startServer(port string) {
	repo := repositories.NewInMemLimiterRepository()
	service := services.NewLimiterService(repo)
	handler := handlers.NewLimiterHandler(service)

	mux := http.NewServeMux()

	// Locate frontend directory
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	frontendDir := filepath.Join(workDir, "frontend")
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		// Fallback for execution from backend/cmd/server
		frontendDir = filepath.Join(workDir, "..", "..", "..", "frontend")
	}

	mux.HandleFunc("/api/allow", handler.HandleAllow)
	mux.HandleFunc("/api/stats", handler.HandleStats)
	mux.HandleFunc("/api/reset", handler.HandleReset)
	mux.HandleFunc("/api/health", handler.HandleHealth)

	// Serve frontend assets
	fs := http.FileServer(http.Dir(frontendDir))
	mux.Handle("/", fs)

	addr := ":" + port
	fmt.Printf("\n🚀 Leke Sliding Window Limiter Console\n")
	fmt.Printf("   URL: http://localhost%s\n", addr)
	fmt.Printf("   Mode: High-Precision Concurrency Guard\n\n")

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func runCliDemo() {
	fmt.Println("=== Leke Sliding Window Limiter CLI Demo ===")
	sw := limiter.NewSlidingWindowLimiter(2*time.Second, 5)

	fmt.Println("\nSimulating 7 rapid requests (limit = 5 reqs / 2 sec):")
	for i := 1; i <= 7; i++ {
		res := sw.AllowWithResult("cli-demo-client")
		status := "✅ ALLOWED"
		if !res.Allowed {
			status = "❌ BLOCKED (429 Too Many Requests)"
		}
		fmt.Printf("Req #%d: %s | Remaining: %d | ResetIn: %.2fs\n",
			i, status, res.Remaining, res.ResetAfterSec)
		time.Sleep(150 * time.Millisecond)
	}
}
