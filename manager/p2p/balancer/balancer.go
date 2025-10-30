package balancer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	saiService "github.com/saiset-co/sai-service/service"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/metrics"
	"github.com/saiset-co/sai-interx-manager/p2p/utils"
	"github.com/saiset-co/sai-interx-manager/types"
)

type LoadBalancer struct {
	nodeID    p2p.NodeID
	metrics   metrics.Collector
	threshold float64
}

func NewLoadBalancer(nodeID p2p.NodeID, metrics metrics.Collector, threshold float64) *LoadBalancer {
	return &LoadBalancer{
		nodeID:    nodeID,
		metrics:   metrics,
		threshold: threshold,
	}
}

func (lb *LoadBalancer) CreateLoadBalancerMiddleware(method string) func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error) {
	return func(next saiService.HandlerFunc, data interface{}, metadata interface{}) (interface{}, int, error) {
		metadataMap, ok := metadata.(map[string]interface{})
		if ok && metadataMap["X-From-Peer"] == true {
			return next(data, metadata)
		}

		shouldHandle, targetNodeID := lb.ShouldHandleRequest()
		if !shouldHandle {
			metadataMap["X-From-Peer"] = true
			metadataMap["X-Original-Node"] = lb.nodeID

			request := types.SaiRequest{
				Method:   method,
				Data:     data,
				Metadata: metadataMap,
			}

			jsonData, err := json.Marshal(request)
			if err != nil {
				return nil, http.StatusInternalServerError, errors.New("failed to marshal request data")
			}

			response, err := lb.ProxyRequest(jsonData, targetNodeID)
			if err != nil {
				logger.Logger.Error("loadBalancerMiddleware: error proxying request", zap.Error(err))
				return nil, http.StatusInternalServerError, errors.New("failed to delegate request")
			}

			defer response.Body.Close()

			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				logger.Logger.Error("failed to read proxied response", zap.Error(err))
				return nil, http.StatusInternalServerError, errors.New("failed to read proxied response")
			}

			var result interface{}
			err = json.Unmarshal(body, &result)
			if err != nil {
				logger.Logger.Error("failed to proxied proxied response", zap.Error(err))
				return nil, http.StatusInternalServerError, errors.New("failed to parse proxied response")
			}

			return result, response.StatusCode, nil
		}

		return next(data, metadata)
	}
}

func (lb *LoadBalancer) ShouldHandleRequest() (bool, p2p.NodeID) {
	localScore := lb.metrics.CalculateScore(lb.nodeID)
	var scores = map[string]float64{}

	for nodeID := range lb.metrics.GetAllNodes() {
		scores[string(nodeID)] = lb.metrics.CalculateScore(nodeID).Total
	}

	bestNodeID, bestScore, err := utils.MapFloatMinValue(scores)
	if err != nil || localScore.Total+lb.threshold <= bestScore {
		bestNodeID = string(lb.nodeID)
	}

	return p2p.NodeID(bestNodeID) == lb.nodeID, p2p.NodeID(bestNodeID)
}

func (lb *LoadBalancer) ProxyRequest(jsonData []byte, targetNodeID p2p.NodeID) (*http.Response, error) {
	nodeInfo, exists := lb.metrics.GetNodeInfo(targetNodeID)
	if !exists {
		err := fmt.Errorf("node %s not found", targetNodeID)
		logger.Logger.Error("ProxyRequest", zap.Error(err))
		return nil, err
	}

	address, _, err := net.SplitHostPort(nodeInfo.Address)
	if err != nil {
		logger.Logger.Error("ProxyRequest", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d", address, nodeInfo.HttpPort), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	response, err := client.Do(req)
	if err != nil {
		logger.Logger.Error("ProxyRequest", zap.Error(err))
		return nil, err
	}

	return response, nil
}
