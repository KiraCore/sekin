package internal

import (
	"time"

	"github.com/spf13/cast"

	"github.com/saiset-co/sai-interx-manager/gateway"
	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/config"
	"github.com/saiset-co/sai-interx-manager/p2p/net"
	"github.com/saiset-co/sai-interx-manager/types"
	"github.com/saiset-co/sai-service/service"
)

type InternalService struct {
	Context         *service.Context
	cosmosGateway   types.Gateway
	ethereumGateway types.Gateway
	storageGateway  types.Gateway
	storage         types.Storage
	p2pServer       p2p.Network
}

func (is *InternalService) Init() {
	var err error

	is.storage = types.NewStorage(
		cast.ToString(is.Context.GetConfig("storage.url", "")),
		cast.ToString(is.Context.GetConfig("storage.token", "")),
	)

	nodeID := cast.ToString(is.Context.GetConfig("p2p.id", ""))
	windowSize := cast.ToInt(is.Context.GetConfig("balancer.window_size", 60))
	threshold := cast.ToFloat64(is.Context.GetConfig("balancer.threshold", 0.2))

	networkConfig := config.NewNetworkConfig(
		config.WithNodeID(p2p.NodeID(nodeID)),
		config.WithListenAddress(cast.ToString(is.Context.GetConfig("p2p.address", "0.0.0.0:9000"))),
		config.WithMaxPeers(cast.ToInt(is.Context.GetConfig("p2p.max_peers", 3))),
		config.WithHTTPPort(cast.ToInt(is.Context.GetConfig("common.http.port", 8080))),
		config.WithMetricsWindowSize(time.Duration(windowSize)*time.Second),
		config.WithLoadBalancerThreshold(threshold),
		config.WithInitialPeers(cast.ToStringSlice(is.Context.GetConfig("p2p.peers", []string{}))),
	)

	is.p2pServer, err = net.NewNetwork(is.Context.Context, networkConfig)
	if err != nil {
		panic(err)
	}

	gatewayFactory := gateway.NewGatewayFactory(is.Context, is.storage)

	is.cosmosGateway, err = gatewayFactory.CreateGateway("cosmos")
	if err != nil {
		panic(err)
	}
	is.ethereumGateway, err = gatewayFactory.CreateGateway("ethereum")
	if err != nil {
		panic(err)
	}
	is.storageGateway, err = gatewayFactory.CreateGateway("storage")
	if err != nil {
		panic(err)
	}
}

func (is *InternalService) Process() {
	if err := is.p2pServer.Start(); err != nil {
		is.p2pServer.Stop()
		is.cosmosGateway.Close()
		is.ethereumGateway.Close()
		is.storageGateway.Close()

		panic(err)
	}
}
