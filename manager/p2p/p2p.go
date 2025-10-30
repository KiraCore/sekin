// Package p2p defines interfaces for the peer-to-peer network system
package p2p

import (
	"time"

	saiService "github.com/saiset-co/sai-service/service"
)

type NodeID string

type Peer interface {
	ID() NodeID
	Address() string
	Close() error
}

type PeerManager interface {
	Start() error
	Stop()
	GetPeerId() NodeID
	AddPeer(address string, remote bool) (Peer, error)
	RemovePeer(id NodeID)
}

type Message interface {
	Type() string
	Payload() interface{}
}

type MessageHandler interface {
	HandleMessage(msg Message, from Peer) error
}

type MetricsCollector interface {
	GetAllNodesMetrics() map[NodeID]NodeMetrics
	CollectLocalMetrics() NodeMetrics
	UpdateNodeMetrics(metrics NodeMetrics, latency float64)
	CalculateScore(nodeID NodeID) Score
	CreateMetricsMiddleware(method string) func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error)
}

type LoadBalancer interface {
	ShouldHandleRequest() (bool, NodeID)
	CreateLoadBalancerMiddleware(method string) func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error)
}

type NodeMetrics struct {
	NodeID         NodeID    `json:"node_id"`
	Address        string    `json:"address"`
	HttpPort       int       `json:"http_port"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	RequestsPerSec float64   `json:"requests_per_sec"`
	AverageLatency float64   `json:"average_latency"`
	ActiveRequests int       `json:"active_requests"`
	ErrorRate      float64   `json:"error_rate"`
	Timestamp      time.Time `json:"timestamp"`
}

type Score struct {
	CPUScore     float64
	MemoryScore  float64
	RPSScore     float64
	LatencyScore float64
	Total        float64
}

type Weights struct {
	CPU     float64
	Memory  float64
	RPS     float64
	Latency float64
}

type Network interface {
	Start() error
	Stop()
	PeerManager() PeerManager
	MetricsCollector() MetricsCollector
	LoadBalancer() LoadBalancer
}
