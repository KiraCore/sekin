package config

import (
	"time"

	"github.com/saiset-co/sai-interx-manager/p2p"
)

type NetworkConfig struct {
	NodeID             p2p.NodeID
	ListenAddress      string
	MaxPeers           int
	HTTPPort           int
	MetricsConfig      MetricsConfig
	LoadBalancerConfig LoadBalancerConfig
	InitialPeers       []string
}

type MetricsConfig struct {
	Weights    p2p.Weights
	WindowSize time.Duration
}

type LoadBalancerConfig struct {
	Threshold float64
}

func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		MaxPeers:      10,
		ListenAddress: "127.0.0.1:8080",
		HTTPPort:      8081,
		MetricsConfig: MetricsConfig{
			Weights: p2p.Weights{
				CPU:     0.3,
				Memory:  0.3,
				RPS:     0.2,
				Latency: 0.2,
			},
			WindowSize: 60 * time.Second,
		},
		LoadBalancerConfig: LoadBalancerConfig{
			Threshold: 0.2,
		},
	}
}
