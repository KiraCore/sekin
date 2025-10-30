package metrics

import (
	"math"
	"sync"
	"time"

	saiService "github.com/saiset-co/sai-service/service"

	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/types"
	"github.com/saiset-co/sai-interx-manager/p2p/utils"
)

type CollectorImpl struct {
	nodeID         p2p.NodeID
	address        string
	httPort        int
	mutex          sync.RWMutex
	requests       map[string]Request
	metrics        map[p2p.NodeID]p2p.NodeMetrics
	latencies      map[p2p.NodeID]float64
	requestHistory []RequestStat
	weights        p2p.Weights
	startTime      time.Time
	windowSize     time.Duration
}

func NewCollector(nodeID p2p.NodeID, address string, httpPort int, weights p2p.Weights, windowSize time.Duration) *CollectorImpl {
	return &CollectorImpl{
		nodeID:         nodeID,
		address:        address,
		httPort:        httpPort,
		requests:       make(map[string]Request),
		metrics:        make(map[p2p.NodeID]p2p.NodeMetrics),
		latencies:      make(map[p2p.NodeID]float64),
		requestHistory: make([]RequestStat, 0),
		weights:        weights,
		startTime:      time.Now(),
		windowSize:     windowSize,
	}
}

func (c *CollectorImpl) CollectLocalMetrics() p2p.NodeMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	windowStart := time.Now().Add(-c.windowSize)
	var requests, errors int
	for _, stat := range c.requestHistory {
		if stat.Timestamp.After(windowStart) {
			requests++
			if stat.IsError {
				errors++
			}
		}
	}

	var totalLatency float64
	validStats := 0
	for _, stat := range c.requestHistory {
		if stat.Timestamp.After(windowStart) {
			totalLatency += stat.Duration
			validStats++
		}
	}

	rps := float64(requests) / c.windowSize.Seconds()
	errorRate := 0.0
	if requests > 0 {
		errorRate = float64(errors) / float64(requests) * 100
	}
	avgLatency := 0.0
	if validStats > 0 {
		avgLatency = totalLatency / float64(validStats)
	}

	c.metrics[c.nodeID] = p2p.NodeMetrics{
		NodeID:         c.nodeID,
		Address:        c.address,
		HttpPort:       c.httPort,
		CPUUsage:       utils.GetCPUUsage(),
		MemoryUsage:    utils.GetMemoryUsage(),
		RequestsPerSec: rps,
		AverageLatency: avgLatency,
		ActiveRequests: len(c.requests),
		ErrorRate:      errorRate,
		Timestamp:      time.Now(),
	}

	return c.metrics[c.nodeID]
}

func (c *CollectorImpl) UpdateNodeMetrics(metrics p2p.NodeMetrics, latency float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.metrics[metrics.NodeID] = metrics
	c.latencies[metrics.NodeID] = latency
	c.cleanup()
}

func (c *CollectorImpl) CalculateScore(nodeID p2p.NodeID) p2p.Score {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	metrics, exists := c.metrics[nodeID]
	if !exists {
		return p2p.Score{Total: 1.0}
	}

	cpuScore := metrics.CPUUsage / 100.0
	memScore := metrics.MemoryUsage / 100.0
	rpsScore := math.Min(metrics.RequestsPerSec/1000.0, 1.0)
	latencyScore := math.Min(c.latencies[nodeID]/1000.0, 1.0)

	total := cpuScore*c.weights.CPU +
		memScore*c.weights.Memory +
		rpsScore*c.weights.RPS +
		latencyScore*c.weights.Latency

	return p2p.Score{
		CPUScore:     cpuScore,
		MemoryScore:  memScore,
		RPSScore:     rpsScore,
		LatencyScore: latencyScore,
		Total:        total,
	}
}

func (c *CollectorImpl) StartRequest(req *Request) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.requests[req.ID] = *req
}

func (c *CollectorImpl) FinishRequest(reqID string, isError bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if req, exists := c.requests[reqID]; exists {
		endTime := time.Now()
		req.EndTime = &endTime
		duration := endTime.Sub(req.StartTime).Seconds()

		c.requestHistory = append(c.requestHistory, RequestStat{
			Data:      req.Data,
			Metadata:  req.Metadata,
			Duration:  duration,
			IsError:   isError,
			Timestamp: endTime,
		})

		delete(c.requests, reqID)
		c.cleanup()
	}
}

func (c *CollectorImpl) GetAllNodesMetrics() map[p2p.NodeID]p2p.NodeMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.metrics
}

func (c *CollectorImpl) GetAllNodes() map[p2p.NodeID]struct{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	nodes := make(map[p2p.NodeID]struct{})
	for nodeID := range c.metrics {
		nodes[nodeID] = struct{}{}
	}
	return nodes
}

func (c *CollectorImpl) GetNodeInfo(nodeID p2p.NodeID) (types.PeerInfo, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	metrics, exists := c.metrics[nodeID]
	if !exists {
		return types.PeerInfo{}, false
	}

	return types.PeerInfo{
		NodeID:    nodeID,
		Address:   metrics.Address,
		HttpPort:  metrics.HttpPort,
		Connected: true,
	}, true
}

func (c *CollectorImpl) CreateMetricsMiddleware(method string) func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error) {
	return func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error) {
		metadataMap, ok := metadata.(map[string]interface{})
		if ok && metadataMap["X-From-Peer"] == "true" {
			return next(data, metadata)
		}

		req := &Request{
			ID:        utils.GenerateID(),
			StartTime: time.Now(),
			Method:    method,
			Data:      data,
			Metadata:  metadata,
			FromPeer:  metadataMap["X-From-Peer"] == "true",
		}

		c.StartRequest(req)

		resp, code, err := next(data, metadata)

		c.FinishRequest(req.ID, code >= 400)

		return resp, code, err
	}
}

func (c *CollectorImpl) cleanup() {
	cutoff := time.Now().Add(-c.windowSize)

	newHistory := make([]RequestStat, 0)
	for _, stat := range c.requestHistory {
		if stat.Timestamp.After(cutoff) {
			newHistory = append(newHistory, stat)
		}
	}
	c.requestHistory = newHistory

	for nodeID, metrics := range c.metrics {
		if metrics.Timestamp.Before(cutoff) {
			delete(c.metrics, nodeID)
			delete(c.latencies, nodeID)
		}
	}
}
