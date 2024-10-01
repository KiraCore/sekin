package main

import (
	"context"
	"net/http"

	"github.com/kiracore/sekin/src/exporter/exporter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Create a context that can be cancelled to stop the Prometheus exporter service gracefully

	// Register the metrics with Prometheus
	registry := exporter.RegisterMetrics()

	// Create an HTTP handler that serves metrics registered with the custom registry
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Start the Prometheus exporter service in a separate goroutine
	go func() {
		exporter.RunPrometheusExporterService(context.Background())
	}()

	// Start the HTTP server to expose the /metrics endpoint
	server := &http.Server{
		Addr: ":9333", // Change this to your preferred port if needed
	}

	// Start the HTTP server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		// Handle error (other than graceful shutdown)
		panic(err)
	}

}
