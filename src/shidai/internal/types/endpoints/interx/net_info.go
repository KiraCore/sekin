package interx

import "time"

type netInfoNodeInfo struct {
	ProtocolVersion struct {
		P2P   int `json:"p2p"`
		Block int `json:"block"`
		App   int `json:"app"`
	} `json:"protocol_version"`
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
	Network    string `json:"network"`
	Version    string `json:"version"`
	Channels   string `json:"channels"`
	Moniker    string `json:"moniker"`
	Other      struct {
		TxIndex    string `json:"tx_index"`
		RPCAddress string `json:"rpc_address"`
	} `json:"other"`
}

type monitor struct {
	Start    time.Time `json:"start"`
	Bytes    int       `json:"bytes"`
	Samples  int       `json:"samples"`
	InstRate int       `json:"inst_rate"`
	CurRate  int       `json:"cur_rate"`
	AvgRate  int       `json:"avg_rate"`
	PeakRate int       `json:"peak_rate"`
	BytesRem int       `json:"bytes_rem"`
	Duration int64     `json:"duration"`
	Idle     int64     `json:"idle"`
	TimeRem  int       `json:"time_rem"`
	Progress int       `json:"progress"`
	Active   bool      `json:"active"`
}

type channel struct {
	ID                int `json:"id"`
	SendQueueCapacity int `json:"send_queue_capacity"`
	SendQueueSize     int `json:"send_queue_size"`
	Priority          int `json:"priority"`
	RecentlySent      int `json:"recently_sent"`
}

type connectionStatus struct {
	Duration    int64     `json:"duration"`
	SendMonitor monitor   `json:"send_monitor"`
	RecvMonitor monitor   `json:"recv_monitor"`
	Channels    []channel `json:"channels"`
}

type peer struct {
	NodeInfo         netInfoNodeInfo  `json:"node_info"`
	IsOutbound       bool             `json:"is_outbound"`
	ConnectionStatus connectionStatus `json:"connection_status"`
	RemoteIP         string           `json:"remote_ip"`
}

type NetInfo struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	NPeers    int      `json:"n_peers"`
	Peers     []peer   `json:"peers"`
}
