package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/kiracore/sekin/src/exporter/exporter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

var (
	ipRateLimit = make(map[string]*rate.Limiter)
	mu          sync.Mutex
)

func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := ipRateLimit[ip]; !exists {
		ipRateLimit[ip] = rate.NewLimiter(rate.Every(1*time.Second), 1)
	}
	return ipRateLimit[ip]
}

func dropConnection(w http.ResponseWriter) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Server doesn't support connection hijacking", http.StatusInternalServerError)
		return
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		fmt.Println("Failed to hijack connection:", err)
		return
	}

	conn.Close()
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Invalid IP", http.StatusBadRequest)
			return
		}

		limiter := getLimiter(ip)

		if !limiter.Allow() {
			dropConnection(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	registry := exporter.RegisterMetrics()

	http.Handle("/metrics", limit(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))

	go func() {
		exporter.RunPrometheusExporterService(context.Background())
	}()

	server := &http.Server{
		Addr: ":9333",
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
