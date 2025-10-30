package types

type GRPCResponse struct {
	Code    float64 `json:"code"`
	Message string  `json:"message"`
	Details []byte  `json:"details"`
}

type EVMStatus struct {
	NodeInfo struct {
		Network    uint64 `json:"network"`
		RPCAddress string `json:"rpc_address"`
		Version    struct {
			Net      string `json:"net"`
			Web3     string `json:"web3"`
			Protocol string `json:"protocol"`
		} `json:"version"`
	} `json:"node_info"`
	SyncInfo struct {
		CatchingUp          bool   `json:"catching_up"`
		EarliestBlockHash   string `json:"earliest_block_hash"`
		EarliestBlockHeight uint64 `json:"earliest_block_height"`
		EarliestBlockTime   uint64 `json:"earliest_block_time"`
		LatestBlockHash     string `json:"latest_block_hash"`
		LatestBlockHeight   uint64 `json:"latest_block_height"`
		LatestBlockTime     uint64 `json:"latest_block_time"`
	} `json:"sync_info"`
	GasPrice string `json:"gas_price"`
}

type IdentityRecord struct {
	ID        uint64      `json:"id,string"`
	Key       string      `json:"key"`
	Value     string      `json:"value"`
	Date      interface{} `json:"date"`
	Verifiers []string    `json:"verifiers"`
}

type QueryValidator struct {
	Top int `json:"top,string"`

	Address             string           `json:"address"`
	Valkey              string           `json:"valkey"`
	Pubkey              string           `json:"pubkey"`
	Proposer            string           `json:"proposer"`
	Moniker             string           `json:"moniker"`
	Status              string           `json:"status"`
	Rank                int64            `json:"rank,string"`
	Streak              int64            `json:"streak,string"`
	Mischance           int64            `json:"mischance,string"`
	MischanceConfidence int64            `json:"mischance_confidence,string"`
	Identity            []IdentityRecord `json:"identity,omitempty"`

	StartHeight           int64  `json:"start_height,string"`
	InactiveUntil         string `json:"inactive_until"`
	LastPresentBlock      int64  `json:"last_present_block,string"`
	MissedBlocksCounter   int64  `json:"missed_blocks_counter,string"`
	ProducedBlocksCounter int64  `json:"produced_blocks_counter,string"`
	StakingPoolId         int64  `json:"staking_pool_id,string,omitempty"`
	StakingPoolStatus     string `json:"staking_pool_status,omitempty"`

	Description       string `json:"description,omitempty"`
	Website           string `json:"website,omitempty"`
	Logo              string `json:"logo,omitempty"`
	Social            string `json:"social,omitempty"`
	Contact           string `json:"contact,omitempty"`
	Validator_node_id string `json:"validator_node_id,omitempty"`
	Sentry_node_id    string `json:"sentry_node_id,omitempty"`
}

type QueryValidators []QueryValidator

func (s QueryValidators) Len() int {
	return len(s)
}
func (s QueryValidators) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s QueryValidators) Less(i, j int) bool {
	if s[i].Status != s[j].Status {
		if s[j].Status == "ACTIVE" {
			return false
		}
		if s[i].Status == "ACTIVE" {
			return true
		}
		return s[i].Status > s[j].Status
	}
	if s[i].Rank != s[j].Rank {
		return s[i].Rank > s[j].Rank
	}
	if s[i].Streak != s[j].Streak {
		return s[i].Streak > s[j].Streak
	}
	if s[i].MissedBlocksCounter != s[j].MissedBlocksCounter {
		return s[i].MissedBlocksCounter < s[j].MissedBlocksCounter
	}

	return false
}

type TokenRatesResponse struct {
	Data []TokenRate `json:"data"`
}

type ValidatorsResponse = struct {
	Validators []QueryValidator `json:"validators,omitempty"`
	Actors     []string         `json:"actors,omitempty"`
	Pagination struct {
		Total int `json:"total,string,omitempty"`
	} `json:"pagination,omitempty"`
}

type ValidatorInfoResponse = struct {
	ValValidatorInfos []ValidatorSigningInfo `json:"info,omitempty"`
	Validators        []QueryValidator       `json:"validators,omitempty"`
}

type AllValidators struct {
	Status struct {
		ActiveValidators   int `json:"active_validators"`
		PausedValidators   int `json:"paused_validators"`
		InactiveValidators int `json:"inactive_validators"`
		JailedValidators   int `json:"jailed_validators"`
		TotalValidators    int `json:"total_validators"`
		WaitingValidators  int `json:"waiting_validators"`
	} `json:"status"`
	PoolTokens      []string                 `json:"-"`
	AddrToValidator map[string]string        `json:"-"`
	PoolToValidator map[int64]QueryValidator `json:"-"`
	Waiting         []string                 `json:"waiting"`
	Validators      []QueryValidator         `json:"validators"`
}

type TokenRate struct {
	Denom       string `json:"denom"`
	FeePayments bool   `json:"feePayments"`
	FeeRate     string `json:"feeRate"`
	StakeCap    string `json:"stakeCap"`
	StakeMin    string `json:"stakeMin"`
	StakeToken  bool   `json:"stakeToken"`
}

type ValidatorSigningInfo struct {
	Address               string `json:"address"`
	StartHeight           int64  `json:"startHeight,string"`
	InactiveUntil         string `json:"inactiveUntil"`
	MischanceConfidence   int64  `json:"mischanceConfidence,string"`
	Mischance             int64  `json:"mischance,string"`
	LastPresentBlock      int64  `json:"lastPresentBlock,string"`
	MissedBlocksCounter   int64  `json:"missedBlocksCounter,string"`
	ProducedBlocksCounter int64  `json:"producedBlocksCounter,string"`
}

type ValidatorPool struct {
	ID                 int64    `json:"id,string"`
	Validator          string   `json:"validator,omitempty"`
	Enabled            bool     `json:"enabled,omitempty"`
	Slashed            string   `json:"slashed"`
	TotalStakingTokens []string `json:"totalStakingTokens"`
	TotalShareTokens   []string `json:"totalShareTokens"`
	TotalRewards       []string `json:"totalRewards"`
	Commission         string   `json:"commission"`
}

type AllPools struct {
	ValToPool map[string]ValidatorPool
	IdToPool  map[int64]ValidatorPool
}

type NodeConfig struct {
	NodeType        string `json:"node_type"`
	SentryNodeID    string `json:"sentry_node_id"`
	SnapshotNodeID  string `json:"snapshot_node_id"`
	ValidatorNodeID string `json:"validator_node_id"`
	SeedNodeID      string `json:"seed_node_id"`
}

type ProtocolVersion struct {
	P2P   string `json:"p2p,omitempty"`
	Block string `json:"block,omitempty"`
	App   string `json:"app,omitempty"`
}

type NodeOtherInfo struct {
	TxIndex    string `json:"tx_index,omitempty"`
	RpcAddress string `json:"rpc_address,omitempty"`
}

type NodeInfo struct {
	ProtocolVersion ProtocolVersion `json:"protocol_version,omitempty"`
	Id              string          `json:"id,omitempty"`
	ListenAddr      string          `json:"listen_addr,omitempty"`
	Network         string          `json:"network,omitempty"`
	Version         string          `json:"version,omitempty"`
	Channels        string          `json:"channels,omitempty"`
	Moniker         string          `json:"moniker,omitempty"`
	Other           NodeOtherInfo   `json:"other,omitempty"`
}

type SyncInfo struct {
	LatestBlockHash     string `json:"latest_block_hash,omitempty"`
	LatestAppHash       string `json:"latest_app_hash,omitempty"`
	LatestBlockHeight   string `json:"latest_block_height,omitempty"`
	LatestBlockTime     string `json:"latest_block_time,omitempty"`
	EarliestBlockHash   string `json:"earliest_block_hash,omitempty"`
	EarliestAppHash     string `json:"earliest_app_hash,omitempty"`
	EarliestBlockHeight string `json:"earliest_block_height,omitempty"`
	EarliestBlockTime   string `json:"earliest_block_time,omitempty"`
	CatchingUp          bool   `json:"catching_up,omitempty"`
}

type PubKey struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type ValidatorInfo struct {
	Address     string  `json:"address,omitempty"`
	PubKey      *PubKey `json:"pub_key,omitempty"`
	VotingPower string  `json:"voting_power,omitempty"`
}

type InterxStatus struct {
	ID         string `json:"id"`
	InterxInfo struct {
		PubKey struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"pub_key,omitempty"`
		Moniker           string     `json:"moniker"`
		KiraAddr          string     `json:"kira_addr"`
		KiraPubKey        string     `json:"kira_pub_key"`
		FaucetAddr        string     `json:"faucet_addr"`
		GenesisChecksum   string     `json:"genesis_checksum"`
		ChainID           string     `json:"chain_id"`
		InterxVersion     string     `json:"version,omitempty"`
		SekaiVersion      string     `json:"sekai_version,omitempty"`
		LatestBlockHeight string     `json:"latest_block_height"`
		CatchingUp        bool       `json:"catching_up"`
		Node              NodeConfig `json:"node"`
	} `json:"interx_info,omitempty"`
	NodeInfo      NodeInfo      `json:"node_info,omitempty"`
	SyncInfo      SyncInfo      `json:"sync_info,omitempty"`
	ValidatorInfo ValidatorInfo `json:"validator_info,omitempty"`
}
