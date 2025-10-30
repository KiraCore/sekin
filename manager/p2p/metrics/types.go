package metrics

import (
	"time"

	saiService "github.com/saiset-co/sai-service/service"

	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/types"
)

type RequestStat struct {
	Data      interface{}
	Metadata  interface{}
	Duration  float64
	IsError   bool
	Timestamp time.Time
}

type Request struct {
	ID        string
	StartTime time.Time
	EndTime   *time.Time
	Method    string
	Data      interface{}
	Metadata  interface{}
	FromPeer  bool
}

type Collector interface {
	GetAllNodesMetrics() map[p2p.NodeID]p2p.NodeMetrics
	CollectLocalMetrics() p2p.NodeMetrics
	UpdateNodeMetrics(metrics p2p.NodeMetrics, latency float64)
	CalculateScore(nodeID p2p.NodeID) p2p.Score
	StartRequest(req *Request)
	FinishRequest(reqID string, isError bool)
	GetAllNodes() map[p2p.NodeID]struct{}
	GetNodeInfo(nodeID p2p.NodeID) (types.PeerInfo, bool)
	CreateMetricsMiddleware(method string) func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error)
}
