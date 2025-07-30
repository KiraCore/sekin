package interxv2

type NetInfo struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	NPeers    string   `json:"n_peers"`
	Peers     []Peer   `json:"peers"`
}

type Peer struct {
	NodeInfo         NodeInfo         `json:"node_info"`
	IsOutbound       bool             `json:"is_outbound"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	RemoteIP         string           `json:"remote_ip"`
}

type ConnectionStatus struct {
	Duration    string    `json:"Duration"`
	SendMonitor Monitor   `json:"SendMonitor"`
	RecvMonitor Monitor   `json:"RecvMonitor"`
	Channels    []Channel `json:"Channels"`
}

type Channel struct {
	ID                int    `json:"ID"`
	Priority          string `json:"Priority"`
	RecentlySent      string `json:"RecentlySent"`
	SendQueueCapacity string `json:"SendQueueCapacity"`
	SendQueueSize     string `json:"SendQueueSize"`
}

type Monitor struct {
	Active   bool   `json:"Active"`
	AvgRate  string `json:"AvgRate"`
	Bytes    string `json:"Bytes"`
	BytesRem string `json:"BytesRem"`
	CurRate  string `json:"CurRate"`
	Duration string `json:"Duration"`
	Idle     string `json:"Idle"`
	InstRate string `json:"InstRate"`
	PeakRate string `json:"PeakRate"`
	Progress int    `json:"Progress"`
	Samples  string `json:"Samples"`
	Start    string `json:"Start"`
	TimeRem  string `json:"TimeRem"`
}

type NodeInfo struct {
	Channels        string          `json:"channels"`
	ID              string          `json:"id"`
	ListenAddr      string          `json:"listen_addr"`
	Moniker         string          `json:"moniker"`
	Network         string          `json:"network"`
	Other           NodeOther       `json:"other"`
	ProtocolVersion ProtocolVersion `json:"protocol_version"`
	Version         string          `json:"version"`
}

type NodeOther struct {
	RPCAddress string `json:"rpc_address"`
	TxIndex    string `json:"tx_index"`
}

type ProtocolVersion struct {
	App   string `json:"app"`
	Block string `json:"block"`
	P2P   string `json:"p2p"`
}
