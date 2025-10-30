package config

import (
	"time"

	"github.com/saiset-co/sai-interx-manager/p2p"
)

type Option func(*NetworkConfig)

func WithNodeID(nodeID p2p.NodeID) Option {
	return func(c *NetworkConfig) {
		c.NodeID = nodeID
	}
}

func WithListenAddress(address string) Option {
	return func(c *NetworkConfig) {
		c.ListenAddress = address
	}
}

func WithMaxPeers(maxPeers int) Option {
	return func(c *NetworkConfig) {
		c.MaxPeers = maxPeers
	}
}

// WithHTTPPort sets the HTTP port
func WithHTTPPort(port int) Option {
	return func(c *NetworkConfig) {
		c.HTTPPort = port
	}
}

func WithMetricsWeights(weights p2p.Weights) Option {
	return func(c *NetworkConfig) {
		c.MetricsConfig.Weights = weights
	}
}

func WithMetricsWindowSize(windowSize time.Duration) Option {
	return func(c *NetworkConfig) {
		c.MetricsConfig.WindowSize = windowSize
	}
}

func WithLoadBalancerThreshold(threshold float64) Option {
	return func(c *NetworkConfig) {
		c.LoadBalancerConfig.Threshold = threshold
	}
}

func NewNetworkConfig(options ...Option) NetworkConfig {
	config := DefaultNetworkConfig()

	for _, option := range options {
		option(&config)
	}

	return config
}

func WithInitialPeers(peers []string) Option {
	return func(c *NetworkConfig) {
		c.InitialPeers = peers
	}
}
