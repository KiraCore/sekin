package net

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/balancer"
	"github.com/saiset-co/sai-interx-manager/p2p/config"
	"github.com/saiset-co/sai-interx-manager/p2p/metrics"
)

type Network struct {
	config           config.NetworkConfig
	peerManager      *PeerManager
	metricsCollector metrics.Collector
	loadBalancer     p2p.LoadBalancer
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewNetwork(ctx context.Context, config config.NetworkConfig) (p2p.Network, error) {
	networkCtx, cancel := context.WithCancel(ctx)

	metricsCollector := metrics.NewCollector(
		config.NodeID,
		config.ListenAddress,
		config.HTTPPort,
		config.MetricsConfig.Weights,
		config.MetricsConfig.WindowSize,
	)

	peerManager := NewPeerManager(
		networkCtx,
		config.NodeID,
		config.ListenAddress,
		config.HTTPPort,
		config.MaxPeers,
		metricsCollector,
	)

	loadBalancer := balancer.NewLoadBalancer(
		config.NodeID,
		metricsCollector,
		config.LoadBalancerConfig.Threshold,
	)

	return &Network{
		config:           config,
		peerManager:      peerManager,
		metricsCollector: metricsCollector,
		loadBalancer:     loadBalancer,
		ctx:              networkCtx,
		cancel:           cancel,
	}, nil
}

func (n *Network) Start() error {
	logger.Logger.Info("Starting UDP P2P Network...")

	if err := n.peerManager.Start(); err != nil {
		logger.Logger.Error("Start", zap.Error(err))
		return err
	}

	if len(n.config.InitialPeers) > 0 {
		go n.connectToInitialPeers()
	}

	logger.Logger.Info("UDP P2P Network started successfully")
	return nil
}

func (n *Network) connectToInitialPeers() {
	for _, address := range n.config.InitialPeers {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}

		logger.Logger.Info("Connecting to initial peer", zap.Any("Address", address))
		peer, err := n.peerManager.AddPeer(address, false)
		if err != nil {
			logger.Logger.Error("NewNetwork", zap.Error(err))
		} else {
			logger.Logger.Info("Successfully connected to peer",
				zap.Any("Peer", peer.ID()),
				zap.Any("Address", peer.Address()),
			)
		}
	}
}

func (n *Network) Stop() {
	logger.Logger.Info("Stopping UDP P2P Network..")
	n.peerManager.Stop()
	n.cancel()
	logger.Logger.Info("UDP P2P Network stopped")
}

func (n *Network) PeerManager() p2p.PeerManager {
	return n.peerManager
}

func (n *Network) MetricsCollector() p2p.MetricsCollector {
	return n.metricsCollector
}

func (n *Network) LoadBalancer() p2p.LoadBalancer {
	return n.loadBalancer
}
