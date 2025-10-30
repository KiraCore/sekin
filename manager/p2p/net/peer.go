package net

import (
	"net"
	"sync"

	"github.com/saiset-co/sai-interx-manager/p2p"
)

type PeerStatus struct {
	connected bool
}

type Peer struct {
	nodeID      p2p.NodeID
	address     string
	httpPort    int
	udpAddr     *net.UDPAddr
	status      PeerStatus
	remotePeer  bool
	statusMutex sync.RWMutex
}

func NewPeer(nodeID p2p.NodeID, address string, httpPort int, udpAddr *net.UDPAddr, remote bool) *Peer {
	return &Peer{
		nodeID:     nodeID,
		address:    address,
		httpPort:   httpPort,
		udpAddr:    udpAddr,
		remotePeer: remote,
		status: PeerStatus{
			connected: true,
		},
	}
}

func (p *Peer) ID() p2p.NodeID {
	return p.nodeID
}

func (p *Peer) Address() string {
	return p.address
}

func (p *Peer) Close() error {
	p.statusMutex.Lock()
	p.status.connected = false
	p.statusMutex.Unlock()
	return nil
}

func (p *Peer) GetUDPAddr() *net.UDPAddr {
	return p.udpAddr
}

func (p *Peer) SetUDPAddr(newAddr *net.UDPAddr) {
	p.udpAddr = newAddr
}
