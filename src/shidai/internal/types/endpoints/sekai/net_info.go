package sekai

type NetInfo struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  Result `json:"result"`
}

type Result struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	NPeers    string   `json:"n_peers"`
	Peers     []Peer   `json:"peers"`
	RemoteIP  string   `json:"remote_ip"`
}

type Peer struct {
	NodeInfo         NodeInfo         `json:"node_info"`
	IsOutbound       bool             `json:"is_outbound"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	RemoteIP         string           `json:"remote_ip"`
}

type NodeInfo struct {
	ProtocolVersion ProtocolVersion `json:"protocol_version"`
	ID              string          `json:"id"`
	ListenAddr      string          `json:"listen_addr"`
	Network         string          `json:"network"`
	Version         string          `json:"version"`
	Channels        string          `json:"channels"`
	Moniker         string          `json:"moniker"`
	Other           OtherInfo       `json:"other"`
}

type ProtocolVersion struct {
	P2P   string `json:"p2p"`
	Block string `json:"block"`
	App   string `json:"app"`
}

type OtherInfo struct {
	TXIndex    string `json:"tx_index"`
	RPCAddress string `json:"rpc_address"`
}

type ConnectionStatus struct {
	Duration    string    `json:"Duration"`
	SendMonitor Monitor   `json:"SendMonitor"`
	RecvMonitor Monitor   `json:"RecvMonitor"`
	Channels    []Channel `json:"Channels"`
}

type Monitor struct {
	Start    string `json:"Start"`
	Bytes    string `json:"Bytes"`
	Samples  string `json:"Samples"`
	InstRate string `json:"InstRate"`
	CurRate  string `json:"CurRate"`
	AvgRate  string `json:"AvgRate"`
	PeakRate string `json:"PeakRate"`
	BytesRem string `json:"BytesRem"`
	Duration string `json:"Duration"`
	Idle     string `json:"Idle"`
	TimeRem  string `json:"TimeRem"`
	Progress int    `json:"Progress"`
	Active   bool   `json:"Active"`
}

type Channel struct {
	ID                int    `json:"ID"`
	SendQueueCapacity string `json:"SendQueueCapacity"`
	SendQueueSize     string `json:"SendQueueSize"`
	Priority          string `json:"Priority"`
	RecentlySent      string `json:"RecentlySent"`
}
