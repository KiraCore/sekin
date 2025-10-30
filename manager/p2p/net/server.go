package net

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/metrics"
	"github.com/saiset-co/sai-interx-manager/p2p/proto"
	"github.com/saiset-co/sai-interx-manager/p2p/types"
)

type PeerManager struct {
	nodeID           p2p.NodeID
	address          string
	p2pPort          int
	httpPort         int
	maxPeers         int
	peers            map[p2p.NodeID]*Peer
	metricsCollector metrics.Collector
	conn             *net.UDPConn
	mutex            sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	messageHandlers  map[string]p2p.MessageHandler
	addrMap          map[string]p2p.NodeID
	addrMapMutex     sync.RWMutex
	pendingJoins     map[string]struct{}
	pendingMutex     sync.RWMutex
	knownPeers       map[p2p.NodeID]*net.UDPAddr
	reconnecting     bool
	reconnectMutex   sync.RWMutex
	missingMetrics   map[p2p.NodeID]time.Time
	peeringMutex     sync.RWMutex
}

func NewPeerManager(
	ctx context.Context,
	nodeID p2p.NodeID,
	address string,
	httpPort int,
	maxPeers int,
	metricsCollector metrics.Collector,
) *PeerManager {
	peerCtx, cancel := context.WithCancel(ctx)

	_, portStr, err := net.SplitHostPort(address)
	var p2pPort int
	if err == nil {
		p2pPort, _ = strconv.Atoi(portStr)
	}

	return &PeerManager{
		nodeID:           nodeID,
		address:          address,
		p2pPort:          p2pPort,
		httpPort:         httpPort,
		maxPeers:         maxPeers,
		peers:            make(map[p2p.NodeID]*Peer),
		metricsCollector: metricsCollector,
		ctx:              peerCtx,
		cancel:           cancel,
		messageHandlers:  make(map[string]p2p.MessageHandler),
		addrMap:          make(map[string]p2p.NodeID),
		pendingJoins:     make(map[string]struct{}),
		knownPeers:       make(map[p2p.NodeID]*net.UDPAddr),
		missingMetrics:   make(map[p2p.NodeID]time.Time),
	}
}

func (pm *PeerManager) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", pm.address)
	if err != nil {
		logger.Logger.Error("Start", zap.Error(err))
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Logger.Error("Start", zap.Error(err))
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}
	pm.conn = conn

	go pm.handleIncomingMessages()
	go pm.startHealthCheck()

	return nil
}

func (pm *PeerManager) Stop() {
	pm.cancel()
	if pm.conn != nil {
		pm.conn.Close()
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for _, peer := range pm.peers {
		peer.Close()
	}
}

func (pm *PeerManager) AddPeer(address string, remote bool) (p2p.Peer, error) {
	return pm.addPeerWithVisited(address, make(map[string]bool), remote)
}

func (pm *PeerManager) addPeerWithVisited(address string, visitedNodes map[string]bool, remote bool) (p2p.Peer, error) {
	logger.Logger.Debug("addPeerWithVisited",
		zap.Any("Address", address),
		zap.Any("Current Node ID", pm.nodeID),
		zap.Bool("Remote", remote),
	)

	if len(pm.LocalPeers()) > 0 && !remote {
		logger.Logger.Debug("ALREADY CONNECTED",
			zap.Int("localPeerCount", len(pm.LocalPeers())),
		)
		return nil, fmt.Errorf("this peer already connected")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		logger.Logger.Error("UDP ADDRESS RESOLUTION ERROR",
			zap.Any("Address", address),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to resolve peer address: %w", err)
	}

	pm.addrMapMutex.RLock()
	nodeID, addrExists := pm.addrMap[udpAddr.String()]
	pm.addrMapMutex.RUnlock()

	if addrExists {
		pm.mutex.RLock()
		peer, peerExists := pm.peers[nodeID]
		pm.mutex.RUnlock()

		if peerExists {
			logger.Logger.Debug("Already connected to this address",
				zap.String("address", address),
				zap.String("nodeID", string(nodeID)))
			return peer, nil
		}
	}

	if visitedNodes != nil && visitedNodes[address] {
		err := fmt.Errorf("this adress already tried")
		logger.Logger.Debug("Skip already visited or failed connection",
			zap.String("address", address),
			zap.Error(err))
		return nil, err
	}

	pm.pendingMutex.Lock()
	pm.pendingJoins[udpAddr.String()] = struct{}{}
	logger.Logger.Debug("Added pending join for address", zap.String("address", udpAddr.String()))
	pm.pendingMutex.Unlock()

	visitedNodes[string(pm.nodeID)] = true

	joinReq := types.JoinRequest{
		NodeID:       pm.nodeID,
		Address:      pm.address,
		HttpPort:     pm.httpPort,
		VisitedNodes: visitedNodes,
		Remote:       remote,
	}

	msg := proto.NewMessage(proto.MessageTypeJoinRequest, joinReq)

	logger.Logger.Debug("SENDING JOIN REQUEST",
		zap.Any("Node ID", joinReq.NodeID),
		zap.Any("Address", joinReq.Address),
		zap.Int("HTTPPort", joinReq.HttpPort),
		zap.String("UDP Address", udpAddr.String()),
		zap.Bool("Remote", remote),
	)

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Logger.Error("JOIN REQUEST MARSHAL ERROR", zap.Error(err))

		pm.pendingMutex.Lock()
		delete(pm.pendingJoins, udpAddr.String())
		pm.pendingMutex.Unlock()

		return nil, fmt.Errorf("failed to marshal join request: %w", err)
	}

	responseCh := make(chan types.JoinResponse, 1)
	errCh := make(chan error, 1)
	timeoutCh := time.After(10 * time.Second)

	handler := &joinResponseHandler{
		responseCh: responseCh,
		errCh:      errCh,
	}

	pm.mutex.Lock()
	pm.messageHandlers[string(proto.MessageTypeJoinResponse)] = handler
	pm.mutex.Unlock()

	defer func() {
		pm.mutex.Lock()
		delete(pm.messageHandlers, string(proto.MessageTypeJoinResponse))
		pm.mutex.Unlock()

		pm.pendingMutex.Lock()
		delete(pm.pendingJoins, udpAddr.String())
		pm.pendingMutex.Unlock()
	}()

	_, sendErr := pm.conn.WriteToUDP(msgBytes, udpAddr)
	if sendErr != nil {
		logger.Logger.Error("JOIN REQUEST SEND ERROR", zap.Error(sendErr))
		return nil, fmt.Errorf("failed to send join request: %w", sendErr)
	}

	select {
	case joinResp := <-responseCh:
		return pm.processJoinResponse(joinResp, address, udpAddr, visitedNodes, remote)
	case err := <-errCh:
		return nil, err
	case <-timeoutCh:
		logger.Logger.Debug("JOIN REQUEST TIMEOUT")
		return nil, fmt.Errorf("join request timed out")
	}
}

func (pm *PeerManager) processJoinResponse(
	joinResp types.JoinResponse,
	address string,
	udpAddr *net.UDPAddr,
	visitedNodes map[string]bool,
	remote bool,
) (p2p.Peer, error) {
	logger.Logger.Debug("JOIN RESPONSE DETAILS",
		zap.Bool("Success", joinResp.Success),
		zap.Any("Node ID", joinResp.NodeID),
		zap.Int("Alternative Peers", len(joinResp.AlternativePeers)),
	)

	if !joinResp.Success {
		logger.Logger.Debug("JOIN REQUEST REJECTED",
			zap.String("Error", joinResp.Error),
		)

		visitedNodes[string(joinResp.NodeID)] = true

		if len(joinResp.AlternativePeers) > 0 {
			logger.Logger.Debug("Alternative Peers available",
				zap.Int("count", len(joinResp.AlternativePeers)))

			for nodeId, peerInfo := range joinResp.AlternativePeers {
				logger.Logger.Debug("Trying alternative peer",
					zap.String("address", peerInfo.Address))

				if visitedNodes[strconv.Itoa(nodeId)] {
					continue
				}

				alternativePeer, err := pm.addPeerWithVisited(peerInfo.Address, visitedNodes, false)
				if err == nil {
					return alternativePeer, nil
				}

				logger.Logger.Debug("Alternative peer connection failed", zap.Error(err))
			}

			return nil, fmt.Errorf("failed to connect to any alternative peers")
		}

		return nil, fmt.Errorf("peer rejected connection: %s", joinResp.Error)
	}

	if !remote && len(pm.LocalPeers()) > 0 {
		logger.Logger.Debug("ALREADY CONNECTED",
			zap.Int("localPeerCount", len(pm.LocalPeers())),
		)
		return nil, fmt.Errorf("this peer already connected")
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	remotePeerID := joinResp.NodeID

	if existingPeer, exists := pm.peers[remotePeerID]; exists {
		logger.Logger.Debug("PEER ALREADY EXISTS",
			zap.Any("Node ID", remotePeerID),
			zap.String("Address", existingPeer.Address()),
			zap.Bool("Remote", existingPeer.remotePeer))
		return existingPeer, nil
	}

	peer := NewPeer(remotePeerID, address, joinResp.HttpPort, udpAddr, remote)
	pm.peers[remotePeerID] = peer

	pm.reconnectMutex.RLock()
	pm.knownPeers[remotePeerID] = udpAddr
	pm.reconnectMutex.RUnlock()

	pm.addrMapMutex.Lock()
	pm.addrMap[udpAddr.String()] = remotePeerID
	pm.addrMapMutex.Unlock()

	logger.Logger.Debug("processJoinResponse",
		zap.Any("Node ID", remotePeerID),
		zap.Any("Address", address),
		zap.Any("HTTPPort", joinResp.HttpPort),
		zap.Any("UDP Address", udpAddr.String()),
		zap.Bool("Remote", remote),
		zap.Int("Alternative Peers", len(joinResp.AlternativePeers)),
	)

	if len(joinResp.AlternativePeers) > 0 {
		go pm.processAlternativePeers(joinResp.AlternativePeers, visitedNodes)
	}

	return peer, nil
}

func (pm *PeerManager) processAlternativePeers(alternativePeers []types.PeerInfo, visitedNodes map[string]bool) {
	pm.peeringMutex.RLock()
	defer pm.peeringMutex.RUnlock()

	time.Sleep(3 * time.Second)

	pm.mutex.RLock()
	existingPeers := make(map[p2p.NodeID]bool)
	for nodeID := range pm.peers {
		existingPeers[nodeID] = true
	}
	pm.mutex.RUnlock()

	var peersToConnect []types.PeerInfo
	for _, peerInfo := range alternativePeers {
		if !existingPeers[peerInfo.NodeID] {
			peersToConnect = append(peersToConnect, peerInfo)
		}
	}

	if len(peersToConnect) == 0 {
		logger.Logger.Debug("No new alternative peers to connect")
		return
	}

	logger.Logger.Debug("Processing alternative peers",
		zap.Int("peers to connect", len(peersToConnect)))

	for _, peerInfo := range peersToConnect {
		logger.Logger.Debug("CONNECTING TO ALTERNATIVE PEER",
			zap.Any("Node ID", peerInfo.NodeID),
			zap.String("Address", peerInfo.Address))

		_, err := pm.addPeerWithVisited(peerInfo.Address, visitedNodes, true)
		if err != nil {
			logger.Logger.Error("ALTERNATIVE PEER CONNECTION ERROR",
				zap.Any("Node ID", peerInfo.NodeID),
				zap.Error(err))
		}
	}
}

func (pm *PeerManager) GetPeerId() p2p.NodeID {
	return pm.nodeID
}

func (pm *PeerManager) RemovePeer(id p2p.NodeID) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if peer, exists := pm.peers[id]; exists {
		pm.addrMapMutex.Lock()
		for addr, nodeID := range pm.addrMap {
			if nodeID == id {
				delete(pm.addrMap, addr)
			}
		}
		pm.addrMapMutex.Unlock()

		peer.Close()
		delete(pm.peers, id)
	}
}

func (pm *PeerManager) handleIncomingMessages() {
	buffer := make([]byte, 65536)

	for {
		n, addr, err := pm.conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-pm.ctx.Done():
				return
			default:
				logger.Logger.Error("UDP READ ERROR", zap.Error(err))
				continue
			}
		}

		go pm.processUDPMessage(buffer[:n], addr)
	}
}

func (pm *PeerManager) processUDPMessage(msgBytes []byte, fromAddr *net.UDPAddr) {
	var msg proto.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		logger.Logger.Error("JSON DECODING ERROR",
			zap.Error(err),
			zap.String("Remote Address", fromAddr.String()),
		)
		return
	}

	logger.Logger.Debug("RECEIVED MESSAGE",
		zap.String("Type", msg.Type()),
		zap.String("From", fromAddr.String()),
	)

	addrStr := fromAddr.String()

	if msg.Type() == string(proto.MessageTypeJoinResponse) {
		pm.pendingMutex.RLock()
		isPending := false

		if _, ok := pm.pendingJoins[addrStr]; ok {
			isPending = true
		} else {
			if _, ok := pm.pendingJoins[addrStr]; ok {
				isPending = true
				logger.Logger.Debug("Found pending join via alternative address",
					zap.String("addrStr", addrStr))
			}
		}
		pm.pendingMutex.RUnlock()

		if isPending {
			handler, exists := pm.messageHandlers[string(proto.MessageTypeJoinResponse)]
			if exists {
				if err := handler.HandleMessage(&msg, nil); err != nil {
					logger.Logger.Error("ERROR HANDLING JOIN RESPONSE", zap.Error(err))
				}
				return
			}
		}
	}

	switch msg.Type() {
	case string(proto.MessageTypeJoinRequest):
		pm.handleJoinRequest(&msg, fromAddr)
	case string(proto.MessageTypeMetrics):
		pm.handleMetricsUpdate(&msg, fromAddr)
	default:
		pm.addrMapMutex.RLock()
		peerID, exists := pm.addrMap[fromAddr.String()]

		if !exists {
			if id, ok := pm.addrMap[fromAddr.String()]; ok {
				exists = true
				peerID = id
				logger.Logger.Debug("Found peer via alternative address",
					zap.String("addrStr", addrStr))
			}
		}
		pm.addrMapMutex.RUnlock()

		if !exists {
			logger.Logger.Debug("RECEIVED MESSAGE FROM UNKNOWN PEER",
				zap.String("address", fromAddr.String()))
			return
		}

		pm.mutex.RLock()
		peer, exists := pm.peers[peerID]
		pm.mutex.RUnlock()

		if !exists {
			logger.Logger.Debug("PEER NOT FOUND IN PEERS MAP",
				zap.String("peerID", string(peerID)))
			return
		}

		handler, ok := pm.messageHandlers[msg.Type()]
		if ok {
			if err := handler.HandleMessage(&msg, peer); err != nil {
				logger.Logger.Error("ERROR HANDLING MESSAGE",
					zap.String("Type", msg.Type()),
					zap.Error(err),
				)
			}
		} else {
			logger.Logger.Debug("NO HANDLER FOR MESSAGE TYPE",
				zap.String("messageType", msg.Type()))
		}
	}
}

func (pm *PeerManager) handleJoinRequest(msg *proto.Message, fromAddr *net.UDPAddr) {
	var joinReq types.JoinRequest
	if err := proto.UnmarshalPayload(msg.Payload(), &joinReq); err != nil {
		logger.Logger.Error("JOIN REQUEST UNMARSHAL ERROR", zap.Error(err))
		return
	}

	logger.Logger.Debug("JOIN REQUEST RECEIVED",
		zap.Any("Node ID", joinReq.NodeID),
		zap.String("Address", joinReq.Address),
		zap.String("From UDP", fromAddr.String()),
		zap.Any("Visited Nodes", joinReq.VisitedNodes),
	)

	pm.mutex.RLock()
	existingPeer, peerExists := pm.peers[joinReq.NodeID]
	pm.mutex.RUnlock()

	if peerExists {
		logger.Logger.Debug("PEER ALREADY EXISTS, UPDATING ADDRESS",
			zap.Any("Node ID", joinReq.NodeID),
			zap.String("Old Address", existingPeer.Address()),
			zap.String("New Address", joinReq.Address))

		pm.addrMapMutex.Lock()
		pm.addrMap[fromAddr.String()] = joinReq.NodeID
		pm.addrMapMutex.Unlock()

		udpAddr, err := net.ResolveUDPAddr("udp", fromAddr.String())
		if err != nil {
			logger.Logger.Error("UDP ADDRESS RESOLUTION ERROR",
				zap.Any("Address", fromAddr.String()),
				zap.Error(err),
			)
			return
		}
		existingPeer.SetUDPAddr(udpAddr)

		pm.mutex.RLock()
		pm.peers[joinReq.NodeID].address = fromAddr.String()
		alternativePeers := pm.getAllPeersInfo(joinReq.NodeID)
		pm.mutex.RUnlock()

		pm.reconnectMutex.RLock()
		pm.knownPeers[joinReq.NodeID] = fromAddr
		pm.reconnectMutex.RUnlock()

		response := types.JoinResponse{
			Success:          true,
			NodeID:           pm.nodeID,
			AlternativePeers: alternativePeers,
			HttpPort:         pm.httpPort,
		}

		pm.sendJoinResponse(response, fromAddr)
		return
	}

	canAccept := true

	if !joinReq.Remote {
		pm.mutex.RLock()
		localPeerCount := len(pm.LocalPeers())
		maxPeers := pm.maxPeers
		pm.mutex.RUnlock()

		canAccept = localPeerCount < maxPeers

		logger.Logger.Debug("LOCAL PEER CHECK",
			zap.Int("localPeerCount", localPeerCount),
			zap.Int("maxPeers", maxPeers),
			zap.Bool("canAccept", canAccept),
			zap.Bool("isRemoteRequest", joinReq.Remote))
	}

	pm.mutex.RLock()
	alternativePeers := pm.getAllPeersInfo(joinReq.NodeID)
	pm.mutex.RUnlock()

	if canAccept {
		pm.mutex.Lock()
		peer := NewPeer(joinReq.NodeID, joinReq.Address, joinReq.HttpPort, fromAddr, joinReq.Remote)
		pm.peers[joinReq.NodeID] = peer
		pm.mutex.Unlock()

		pm.addrMapMutex.Lock()
		pm.addrMap[fromAddr.String()] = joinReq.NodeID
		pm.addrMapMutex.Unlock()

		response := types.JoinResponse{
			Success:          true,
			NodeID:           pm.nodeID,
			AlternativePeers: alternativePeers,
			HttpPort:         pm.httpPort,
		}

		pm.sendJoinResponse(response, fromAddr)

		logger.Logger.Debug("PEER ADDED",
			zap.Any("Node ID", joinReq.NodeID),
			zap.String("Address", joinReq.Address),
			zap.String("UDP Address", fromAddr.String()),
			zap.Int("Alternative Peers", len(alternativePeers)),
			zap.Bool("Remote", joinReq.Remote))
	} else {
		response := types.JoinResponse{
			Success:          false,
			NodeID:           pm.nodeID,
			AlternativePeers: alternativePeers,
			Error:            "Maximum number of local peers reached",
		}

		pm.sendJoinResponse(response, fromAddr)

		logger.Logger.Debug("PEER JOIN REQUEST REJECTED",
			zap.Any("Node ID", joinReq.NodeID),
			zap.String("Address", joinReq.Address),
			zap.Int("Alternative Peers", len(alternativePeers)),
			zap.Bool("isRemoteRequest", joinReq.Remote))
	}
}

func (pm *PeerManager) getAllPeersInfo(excludeNodeID p2p.NodeID) []types.PeerInfo {
	alternativePeers := make([]types.PeerInfo, 0)
	for nodeID, peer := range pm.peers {
		if nodeID != pm.nodeID && nodeID != excludeNodeID {
			peerAddress := peer.Address()
			alternativePeers = append(alternativePeers, types.PeerInfo{
				NodeID:    nodeID,
				Address:   peerAddress,
				HttpPort:  peer.httpPort,
				Connected: true,
			})
		}
	}
	return alternativePeers
}

func (pm *PeerManager) remotePeerExists(nodeID p2p.NodeID) bool {
	for _nodeID, peer := range pm.peers {
		if nodeID == _nodeID && peer.remotePeer {
			return true
		}
	}
	return false
}

func (pm *PeerManager) sendJoinResponse(response types.JoinResponse, toAddr *net.UDPAddr) {
	respMsg := proto.NewMessage(proto.MessageTypeJoinResponse, response)
	respBytes, err := json.Marshal(respMsg)
	if err != nil {
		logger.Logger.Error("JOIN RESPONSE MARSHAL ERROR", zap.Error(err))
		return
	}

	if _, err = pm.conn.WriteToUDP(respBytes, toAddr); err != nil {
		logger.Logger.Error("JOIN RESPONSE SEND ERROR", zap.Error(err))
	}
}

func (pm *PeerManager) handleMetricsUpdate(msg *proto.Message, fromAddr *net.UDPAddr) {
	pm.addrMapMutex.RLock()
	_, exists := pm.addrMap[fromAddr.String()]
	pm.addrMapMutex.RUnlock()

	if !exists {
		logger.Logger.Debug("METRICS FROM UNKNOWN PEER",
			zap.String("address", fromAddr.String()))
		return
	}

	var nodeMetrics p2p.NodeMetrics
	if err := proto.UnmarshalPayload(msg.Payload(), &nodeMetrics); err != nil {
		logger.Logger.Error("METRICS UNMARSHAL ERROR", zap.Error(err))
		return
	}

	logger.Logger.Debug("RECEIVED METRICS FROM PEER",
		zap.Any("Node ID", nodeMetrics.NodeID),
		zap.String("Address", nodeMetrics.Address),
		zap.Int("HTTP", nodeMetrics.HttpPort),
		zap.Float64("CPU Usage", nodeMetrics.CPUUsage),
		zap.Float64("Memory Usage", nodeMetrics.MemoryUsage),
		zap.Float64("Requests/sec", nodeMetrics.RequestsPerSec),
		zap.Float64("Average Latency", nodeMetrics.AverageLatency),
	)

	latency := time.Since(nodeMetrics.Timestamp).Seconds() * 1000
	pm.metricsCollector.UpdateNodeMetrics(nodeMetrics, latency)
}

func (pm *PeerManager) startHealthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	reconnectTicker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()
	defer reconnectTicker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.sendMetricsToPeers()
			pm.updateLastConnCheck()
		case <-reconnectTicker.C:
			pm.checkAndReconnect()
		}
	}
}

func (pm *PeerManager) reconnectToPeers() {
	pm.peeringMutex.RLock()
	defer pm.peeringMutex.RUnlock()

	defer func() {
		pm.reconnectMutex.Lock()
		pm.reconnecting = false
		pm.reconnectMutex.Unlock()
	}()

	pm.reconnectMutex.RLock()
	initialPeers := pm.knownPeers
	pm.reconnectMutex.RUnlock()

	logger.Logger.Debug("ATTEMPTING TO RECONNECT TO NETWORK",
		zap.Any("Initial peers", initialPeers),
	)

	for _, address := range initialPeers {
		select {
		case <-pm.ctx.Done():
			return
		default:
		}

		if address.String() == "" {
			continue
		}

		logger.Logger.Debug("RECONNECTING: Trying peer", zap.String("address", address.String()))

		peer, err := pm.AddPeer(address.String(), false)
		if err != nil {
			logger.Logger.Error("RECONNECTION FAILED", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}

		logger.Logger.Debug("RECONNECTION SUCCESSFUL",
			zap.String("peerID", string(peer.ID())),
			zap.String("address", peer.Address()))

		break
	}

	pm.mutex.RLock()
	stillNoPeers := len(pm.peers) == 0
	pm.mutex.RUnlock()

	if stillNoPeers {
		logger.Logger.Error("RECONNECTION FAILED: Could not connect to any peer")
	}
}

func (pm *PeerManager) checkAndReconnect() {
	pm.mutex.RLock()
	activePeers := len(pm.peers)
	pm.mutex.RUnlock()

	pm.reconnectMutex.RLock()
	reconnecting := pm.reconnecting
	initialPeers := pm.knownPeers
	pm.reconnectMutex.RUnlock()

	if activePeers < len(initialPeers) && !reconnecting && len(initialPeers) > 0 {
		pm.reconnectMutex.Lock()
		pm.reconnecting = true
		pm.reconnectMutex.Unlock()

		go pm.reconnectToPeers()
	}
}

func (pm *PeerManager) updateLastConnCheck() {
	pm.mutex.RLock()
	_peers := pm.peers
	pm.mutex.RUnlock()

	if len(_peers) > 0 {
		allMetrics := pm.metricsCollector.GetAllNodesMetrics()

		for nodeID, _ := range _peers {
			nodeMetrics, ok := allMetrics[nodeID]
			pm.reconnectMutex.Lock()
			missingMetrics, isMissing := pm.missingMetrics[nodeID]
			pm.reconnectMutex.Unlock()

			if ok {
				if nodeMetrics.Timestamp.Add(30 * time.Second).Before(time.Now()) {
					pm.RemovePeer(nodeID)

					pm.reconnectMutex.Lock()
					if _, exists := pm.missingMetrics[nodeID]; exists {
						delete(pm.missingMetrics, nodeID)
					}
					pm.reconnectMutex.Unlock()
				}
			} else if isMissing && missingMetrics.Add(30*time.Second).Before(time.Now()) {
				pm.RemovePeer(nodeID)

				pm.reconnectMutex.Lock()
				if _, exists := pm.missingMetrics[nodeID]; exists {
					delete(pm.missingMetrics, nodeID)
				}
				pm.reconnectMutex.Unlock()
			} else {
				pm.reconnectMutex.Lock()
				pm.missingMetrics[nodeID] = time.Now()
				pm.reconnectMutex.Unlock()
			}
		}
	}
}

func (pm *PeerManager) sendMetricsToPeers() {
	localMetrics := pm.metricsCollector.CollectLocalMetrics()

	localMetrics.Address = pm.address
	localMetrics.HttpPort = pm.httpPort

	metricsMsg := proto.NewMessage(proto.MessageTypeMetrics, localMetrics)

	//logger.Logger.Debug("PREPARING TO SEND METRICS",
	//	zap.Any("Local Node ID", localMetrics.NodeID),
	//	zap.Float64("Memory Usage", localMetrics.MemoryUsage),
	//	zap.Time("Timestamp", localMetrics.Timestamp),
	//	zap.String("Local Address", localMetrics.Address),
	//)

	msgBytes, err := json.Marshal(metricsMsg)
	if err != nil {
		logger.Logger.Error("METRICS SERIALIZATION ERROR", zap.Error(err))
		return
	}

	if len(msgBytes) > 1400 {
		logger.Logger.Warn("Metrics message size may exceed UDP MTU",
			zap.Int("size", len(msgBytes)))
	}

	msgBytes = append(msgBytes, '\n')

	pm.mutex.RLock()
	activePeers := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		activePeers = append(activePeers, peer)
	}
	pm.mutex.RUnlock()

	//logger.Logger.Debug("ACTIVE PEERS FOR METRICS", zap.Int("count", len(activePeers)))

	if len(activePeers) == 0 {
		return
	}

	var wg sync.WaitGroup

	for _, peer := range activePeers {
		wg.Add(1)
		go func(peer *Peer) {
			defer wg.Done()

			udpAddr := peer.GetUDPAddr()
			if udpAddr == nil {
				logger.Logger.Error("METRICS SEND FAILED: No address for peer",
					zap.String("peerID", string(peer.ID())))
				return
			}

			logger.Logger.Debug("ATTEMPTING TO SEND METRICS",
				zap.String("To Peer", string(peer.ID())),
				zap.Any("UDP Address", udpAddr),
			)

			_, err := pm.conn.WriteToUDP(msgBytes, udpAddr)
			if err == nil {
				return
			}
		}(peer)
	}

	wg.Wait()
}

func (pm *PeerManager) LocalPeers() map[p2p.NodeID]*Peer {
	var localPeers = map[p2p.NodeID]*Peer{}

	pm.mutex.RLock()
	for nodeId, peer := range pm.peers {
		if !peer.remotePeer {
			localPeers[nodeId] = peer
		}
	}
	pm.mutex.RUnlock()

	return localPeers
}
