package types

import "github.com/saiset-co/sai-interx-manager/p2p"

type JoinRequest struct {
	NodeID       p2p.NodeID      `json:"node_id"`
	Address      string          `json:"address"`
	HttpPort     int             `json:"http_port"`
	VisitedNodes map[string]bool `json:"visited_nodes,omitempty"`
	Remote       bool            `json:"remote"`
}

type JoinResponse struct {
	Success          bool       `json:"success"`
	NodeID           p2p.NodeID `json:"node_id"`
	Error            string     `json:"error,omitempty"`
	AlternativePeers []PeerInfo `json:"alternative_peers,omitempty"`
	NATPort          int        `json:"nat_port,omitempty"`
	HttpPort         int        `json:"http_port"`
}

type PeerInfo struct {
	NodeID    p2p.NodeID
	Address   string
	HttpPort  int
	Connected bool
}
