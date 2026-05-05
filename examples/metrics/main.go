// Package main provides an example of using the Firefly metrics system.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zhangpeihaoks/firefly/internal/metrics"
	httptransport "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

func main() {
	// Initialize metrics system
	metrics.Initialize()

	// Create HTTP server
	server := httptransport.NewServer(
		httptransport.Address(":8080"),
	)

	// Register metrics endpoint
	server.RegisterMetrics("/metrics", metrics.Handler())

	// Register a simple test endpoint
	server.Route("GET", "/test", func(ctx context.Context, req any) (any, error) {
		return map[string]string{
			"message": "Hello, metrics!",
			"time":    time.Now().Format(time.RFC3339),
		}, nil
	})

	// Register a counter endpoint
	server.Route("GET", "/counter", func(ctx context.Context, req any) (any, error) {
		// Create and register a custom counter
		counter := metrics.NewCounter(prometheus.CounterOpts{
			Name: "custom_requests_total",
			Help: "Total number of custom requests",
		})
		counter.MustRegister()

		// Increment the counter
		if c, ok := counter.Collector().(prometheus.Counter); ok {
			c.Inc()
		}

		return map[string]string{
			"message": "Counter incremented!",
		}, nil
	})

	// Start server in background
	go func() {
		if err := server.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(2 * time.Second)

	// Make some test requests
	fmt.Println("Making test requests...")

	// Request to test endpoint
	resp, err := http.Get("http://localhost:8080/test")
	if err != nil {
		log.Printf("Error making test request: %v", err)
	} else {
		resp.Body.Close()
		fmt.Println("Test request completed")
	}

	// Request to counter endpoint
	resp, err = http.Get("http://localhost:8080/counter")
	if err != nil {
		log.Printf("Error making counter request: %v", err)
	} else {
		resp.Body.Close()
		fmt.Println("Counter request completed")
	}

	// Request to metrics endpoint
	resp, err = http.Get("http://localhost:8080/metrics")
	if err != nil {
		log.Printf("Error fetching metrics: %v", err)
	} else {
		fmt.Println("Metrics endpoint is working!")
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body[:min(len(body), 500)]))
		resp.Body.Close()
	}

	fmt.Println("\nExample completed. Server is running on http://localhost:8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET /test     - Test endpoint")
	fmt.Println("  GET /counter  - Increment custom counter")
	fmt.Println("  GET /metrics  - Prometheus metrics")

	// Keep server running
	select {}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
