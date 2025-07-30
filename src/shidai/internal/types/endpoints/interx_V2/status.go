package interxv2

import "time"

type Status struct {
	ID            string         `json:"id"`
	InterxInfo    InterxInfo     `json:"interx_info"`
	NodeInfo      StatusNodeInfo `json:"node_info"`
	SyncInfo      SyncInfo       `json:"sync_info"`
	ValidatorInfo ValidatorInfo  `json:"validator_info"`
}

type InterxInfo struct {
	PubKey            PubKey  `json:"pub_key"`
	Moniker           string  `json:"moniker"`
	KiraAddr          string  `json:"kira_addr"`
	KiraPubKey        string  `json:"kira_pub_key"`
	FaucetAddr        string  `json:"faucet_addr"`
	GenesisChecksum   string  `json:"genesis_checksum"`
	ChainID           string  `json:"chain_id"`
	LatestBlockHeight string  `json:"latest_block_height"`
	CatchingUp        bool    `json:"catching_up"`
	Node              NodeIDs `json:"node"`
}

type PubKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type NodeIDs struct {
	NodeType        string `json:"node_type"`
	SentryNodeID    string `json:"sentry_node_id"`
	SnapshotNodeID  string `json:"snapshot_node_id"`
	ValidatorNodeID string `json:"validator_node_id"`
	SeedNodeID      string `json:"seed_node_id"`
}

type StatusNodeInfo struct {
	ProtocolVersion StatusProtocolVersion `json:"protocol_version"`
	ID              string                `json:"id"`
	ListenAddr      string                `json:"listen_addr"`
	Network         string                `json:"network"`
	Version         string                `json:"version"`
	Channels        string                `json:"channels"`
	Moniker         string                `json:"moniker"`
	Other           StatusNodeOther       `json:"other"`
}

type StatusProtocolVersion struct {
	P2P   string `json:"p2p"`
	Block string `json:"block"`
	App   string `json:"app"`
}

type StatusNodeOther struct {
	TxIndex    string `json:"tx_index"`
	RPCAddress string `json:"rpc_address"`
}

type SyncInfo struct {
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestAppHash       string    `json:"latest_app_hash"`
	LatestBlockHeight   string    `json:"latest_block_height"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	EarliestBlockHash   string    `json:"earliest_block_hash"`
	EarliestAppHash     string    `json:"earliest_app_hash"`
	EarliestBlockHeight string    `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`
}

type ValidatorInfo struct {
	Address     string `json:"address"`
	PubKey      PubKey `json:"pub_key"`
	VotingPower string `json:"voting_power"`
}
