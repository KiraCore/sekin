package types

import (
	"errors"
	"os"
)

type (
	SekinPackagesVersion struct {
		Sekai  string
		Interx string
		Shidai string
	}

	InfraFiles map[string]string

	AppInfo struct {
		Version string `json:"version"`
		Infra   bool   `json:"infra"`
	}

	StatusResponse struct {
		Sekai  AppInfo `json:"sekai"`
		Interx AppInfo `json:"interx"`
		Shidai AppInfo `json:"shidai"`
		Syslog AppInfo `json:"syslog-ng"`
	}

	Config struct {
		ProxyApp               string                `toml:"proxy_app"`
		Moniker                string                `toml:"moniker"`
		FastSync               bool                  `toml:"fast_sync"`
		DBBackend              string                `toml:"db_backend"`
		DBDir                  string                `toml:"db_dir"`
		LogLevel               string                `toml:"log_level"`
		LogFormat              string                `toml:"log_format"`
		GenesisFile            string                `toml:"genesis_file"`
		PrivValidatorKeyFile   string                `toml:"priv_validator_key_file"`
		PrivValidatorStateFile string                `toml:"priv_validator_state_file"`
		PrivValidatorLaddr     string                `toml:"priv_validator_laddr"`
		NodeKeyFile            string                `toml:"node_key_file"`
		ABCI                   string                `toml:"abci"`
		FilterPeers            bool                  `toml:"filter_peers"`
		RPC                    RPCConfig             `toml:"rpc"`
		P2P                    P2PConfig             `toml:"p2p"`
		Mempool                MempoolConfig         `toml:"mempool"`
		StateSync              StateSyncConfig       `toml:"statesync"`
		FastSyncConfig         FastSyncConfig        `toml:"fastsync"`
		Consensus              ConsensusConfig       `toml:"consensus"`
		Storage                StorageConfig         `toml:"storage"`
		TxIndexer              TxIndexerConfig       `toml:"tx_index"`
		Instrumentation        InstrumentationConfig `toml:"instrumentation"`
	}

	RPCConfig struct {
		Laddr                                string   `toml:"laddr"`
		CorsAllowedOrigins                   []string `toml:"cors_allowed_origins"`
		CorsAllowedMethods                   []string `toml:"cors_allowed_methods"`
		CorsAllowedHeaders                   []string `toml:"cors_allowed_headers"`
		GRPCListenAddr                       string   `toml:"grpc_laddr"`
		GRPCMaxOpenConnections               int      `toml:"grpc_max_open_connections"`
		Unsafe                               bool     `toml:"unsafe"`
		MaxOpenConnections                   int      `toml:"max_open_connections"`
		MaxSubscriptionClients               int      `toml:"max_subscription_clients"`
		MaxSubscriptionsPerClient            int      `toml:"max_subscriptions_per_client"`
		ExperimentalSubscriptionBufferSize   int      `toml:"experimental_subscription_buffer_size"`
		ExperimentalWebSocketWriteBufferSize int      `toml:"experimental_websocket_write_buffer_size"`
		ExperimentalCloseOnSlowClient        bool     `toml:"experimental_close_on_slow_client"`
		TimeoutBroadcastTxCommit             string   `toml:"timeout_broadcast_tx_commit"`
		MaxBodyBytes                         int      `toml:"max_body_bytes"`
		MaxHeaderBytes                       int      `toml:"max_header_bytes"`
		TLSCertFile                          string   `toml:"tls_cert_file"`
		TLSKeyFile                           string   `toml:"tls_key_file"`
		PprofLaddr                           string   `toml:"pprof_laddr"`
	}

	P2PConfig struct {
		Laddr                        string `toml:"laddr"`
		ExternalAddress              string `toml:"external_address"`
		Seeds                        string `toml:"seeds"`
		PersistentPeers              string `toml:"persistent_peers"`
		UPNP                         bool   `toml:"upnp"`
		AddrBookFile                 string `toml:"addr_book_file"`
		AddrBookStrict               bool   `toml:"addr_book_strict"`
		MaxNumInboundPeers           int    `toml:"max_num_inbound_peers"`
		MaxNumOutboundPeers          int    `toml:"max_num_outbound_peers"`
		UnconditionalPeerIDs         string `toml:"unconditional_peer_ids"`
		PersistentPeersMaxDialPeriod string `toml:"persistent_peers_max_dial_period"`
		FlushThrottleTimeout         string `toml:"flush_throttle_timeout"`
		MaxPacketMsgPayloadSize      int    `toml:"max_packet_msg_payload_size"`
		SendRate                     int    `toml:"send_rate"`
		RecvRate                     int    `toml:"recv_rate"`
		Pex                          bool   `toml:"pex"`
		SeedMode                     bool   `toml:"seed_mode"`
		PrivatePeerIDs               string `toml:"private_peer_ids"`
		AllowDuplicateIP             bool   `toml:"allow_duplicate_ip"`
		HandshakeTimeout             string `toml:"handshake_timeout"`
		DialTimeout                  string `toml:"dial_timeout"`
	}

	MempoolConfig struct {
		Version               string `toml:"version"`
		Recheck               bool   `toml:"recheck"`
		Broadcast             bool   `toml:"broadcast"`
		WALDir                string `toml:"wal_dir"`
		Size                  int    `toml:"size"`
		MaxTxsBytes           int    `toml:"max_txs_bytes"`
		CacheSize             int    `toml:"cache_size"`
		KeepInvalidTxsInCache bool   `toml:"keep-invalid-txs-in-cache"`
		MaxTxBytes            int    `toml:"max_tx_bytes"`
		MaxBatchBytes         int    `toml:"max_batch_bytes"`
		TTLDuration           string `toml:"ttl-duration"`
		TTLNumBlocks          int    `toml:"ttl-num-blocks"`
	}

	StateSyncConfig struct {
		Enable              bool   `toml:"enable"`
		RPCServers          string `toml:"rpc_servers"`
		TrustHeight         int    `toml:"trust_height"`
		TrustHash           string `toml:"trust_hash"`
		TrustPeriod         string `toml:"trust_period"`
		DiscoveryTime       string `toml:"discovery_time"`
		TempDir             string `toml:"temp_dir"`
		ChunkRequestTimeout string `toml:"chunk_request_timeout"`
		ChunkFetchers       int    `toml:"chunk_fetchers"`
	}

	FastSyncConfig struct {
		Version string `toml:"version"`
	}

	ConsensusConfig struct {
		WALFile                     string `toml:"wal_file"`
		TimeoutPropose              string `toml:"timeout_propose"`
		TimeoutProposeDelta         string `toml:"timeout_propose_delta"`
		TimeoutPrevote              string `toml:"timeout_prevote"`
		TimeoutPrevoteDelta         string `toml:"timeout_prevote_delta"`
		TimeoutPrecommit            string `toml:"timeout_precommit"`
		TimeoutPrecommitDelta       string `toml:"timeout_precommit_delta"`
		TimeoutCommit               string `toml:"timeout_commit"`
		DoubleSignCheckHeight       int    `toml:"double_sign_check_height"`
		SkipTimeoutCommit           bool   `toml:"skip_timeout_commit"`
		CreateEmptyBlocks           bool   `toml:"create_empty_blocks"`
		CreateEmptyBlocksInterval   string `toml:"create_empty_blocks_interval"`
		PeerGossipSleepDuration     string `toml:"peer_gossip_sleep_duration"`
		PeerQueryMaj23SleepDuration string `toml:"peer_query_maj23_sleep_duration"`
	}

	StorageConfig struct {
		DiscardABCIMResponses bool `toml:"discard_abci_responses"`
	}

	TxIndexerConfig struct {
		Indexer  string `toml:"indexer"`
		PSQLConn string `toml:"psql-conn"`
	}

	InstrumentationConfig struct {
		Prometheus           bool   `toml:"prometheus"`
		PrometheusListenAddr string `toml:"prometheus_listen_addr"`
		MaxOpenConnections   int    `toml:"max_open_connections"`
		Namespace            string `toml:"namespace"`
	}

	AppConfig struct {
		MinimumGasPrices    string             `toml:"minimum-gas-prices"`
		Pruning             string             `toml:"pruning"`
		PruningKeepRecent   int                `toml:"pruning-keep-recent"`
		PruningKeepEvery    int                `toml:"pruning-keep-every"`
		PruningInterval     int                `toml:"pruning-interval"`
		HaltHeight          int                `toml:"halt-height"`
		HaltTime            int                `toml:"halt-time"`
		MinRetainBlocks     int                `toml:"min-retain-blocks"`
		InterBlockCache     bool               `toml:"inter-block-cache"`
		IndexEvents         []string           `toml:"index-events"`
		IavlCacheSize       int                `toml:"iavl-cache-size"`
		IavlDisableFastNode bool               `toml:"iavl-disable-fastnode"`
		Telemetry           TelemetryConfig    `toml:"telemetry"`
		API                 APIConfig          `toml:"api"`
		Rosetta             RosettaConfig      `toml:"rosetta"`
		GRPC                GRPCConfig         `toml:"grpc"`
		GRPCWeb             GRPCWebConfig      `toml:"grpc-web"`
		StateSync           StateSyncAppConfig `toml:"state-sync"`
		Wasm                WasmConfig         `toml:"wasm"`
	}

	TelemetryConfig struct {
		ServiceName             string     `toml:"service-name"`
		Enabled                 bool       `toml:"enabled"`
		EnableHostname          bool       `toml:"enable-hostname"`
		EnableHostnameLabel     bool       `toml:"enable-hostname-label"`
		EnableServiceLabel      bool       `toml:"enable-service-label"`
		PrometheusRetentionTime int        `toml:"prometheus-retention-time"`
		GlobalLabels            [][]string `toml:"global-labels"`
	}

	APIConfig struct {
		Enable             bool   `toml:"enable"`
		Swagger            bool   `toml:"swagger"`
		Address            string `toml:"address"`
		MaxOpenConnections int    `toml:"max-open-connections"`
		RPCReadTimeout     int    `toml:"rpc-read-timeout"`
		RPCWriteTimeout    int    `toml:"rpc-write-timeout"`
		RPCMaxBodyBytes    int    `toml:"rpc-max-body-bytes"`
		EnableUnsafeCORS   bool   `toml:"enabled-unsafe-cors"`
	}

	RosettaConfig struct {
		Enable     bool   `toml:"enable"`
		Address    string `toml:"address"`
		Blockchain string `toml:"blockchain"`
		Network    string `toml:"network"`
		Retries    int    `toml:"retries"`
		Offline    bool   `toml:"offline"`
	}

	GRPCConfig struct {
		Enable  bool   `toml:"enable"`
		Address string `toml:"address"`
	}

	GRPCWebConfig struct {
		Enable           bool   `toml:"enable"`
		Address          string `toml:"address"`
		EnableUnsafeCORS bool   `toml:"enable-unsafe-cors"`
	}

	StateSyncAppConfig struct {
		SnapshotInterval   int `toml:"snapshot-interval"`
		SnapshotKeepRecent int `toml:"snapshot-keep-recent"`
	}

	WasmConfig struct {
		QueryGasLimit int `toml:"query_gas_limit"`
	}
)

const (
	SEKAI_HOME          string = "/sekai"
	INTERX_HOME         string = "/interx"
	DEFAULT_INTERX_PORT int    = 11000
	DEFAULT_P2P_PORT    int    = 26656
	DEFAULT_RPC_PORT    int    = 26657
	DEFAULT_GRPC_PORT   int    = 9090

	SEKAI_CONTAINER_ADDRESS  string = "sekai.local"
	INTERX_CONTAINER_ADDRESS string = "interx.local"

	SEKAI_CONTAINER_ID  = "sekin-sekai-1"
	INTERX_CONTAINER_ID = "sekin-interx-1"
	SHIDAI_CONTAINER_ID = "sekin-shidai-1"
	SYSLOG_CONTAINER_ID = "seking-syslog-ng-1"

	SEKAI_RPC_LADDR  = "tcp://sekai.local:26657"
	SEKAI_P2P_LADDR  = "tcp://0.0.0.0:26657"
	SEKAI_gRPC_LADDR = "0.0.0.0:9090"

	ShidaiLogPath = "/syslog-data/syslog-ng/logs/shidai.log"
	SekaiLogPath  = "/syslog-data/syslog-ng/logs/sekai.log"
	InterxLogPath = "/syslog-data/syslog-ng/logs/interx.log"

	DashboardPath = "/shidaid/dashboard_cache.json"
	DashboardUrl  = "http://127.0.0.1:8282/dashboard"

	InvalidOrMissingMnemonic  = "invalid or missing mnemonic"
	InvalidOrMissingIP        = "invalid or missing IP"
	NoPublicIPAddresses       = "no public IP addresses found"
	MultiplePublicIPAddresses = "multiple public IP address found"

	InvalidOrMissingP2PPort    = "invalid or missing P2P port"
	InvalidOrMissingRPCPort    = "invalid or missing RPC port"
	InvalidOrMissingInterxPort = "invalid or missing interx port"

	InvalidOrMissingTx = "invalid or missing tx"

	FilePermRO os.FileMode = 0444
	FilePermRW os.FileMode = 0644
	FilePermEX os.FileMode = 0755

	DirPermRO os.FileMode = 0555
	DirPermWR os.FileMode = 0755

	UPDATER_BIN_PATH = "/updater"
)

var (
	ErrInvalidOrMissingMnemonic = errors.New(InvalidOrMissingMnemonic)
	ErrInvalidOrMissingIP       = errors.New(InvalidOrMissingIP)

	ErrInvalidOrMissingTx = errors.New(InvalidOrMissingTx)

	ErrInvalidOrMissingP2PPort    = errors.New(InvalidOrMissingP2PPort)
	ErrInvalidOrMissingRPCPort    = errors.New(InvalidOrMissingRPCPort)
	ErrInvalidOrMissingInterxPort = errors.New(InvalidOrMissingInterxPort)

	ErrNoPublicIPAddresses       = errors.New(NoPublicIPAddresses)
	ErrMultiplePublicIPAddresses = errors.New(MultiplePublicIPAddresses)

	SekaiFiles = InfraFiles{
		"config.toml":        "/sekai/config/config.toml",
		"app.toml":           "/sekai/config/app.toml",
		"priv_validator_key": "/sekai/config/priv_validator_key.json",
		"genesis.json":       "/sekai/config/genesis.json",
		"client.toml":        "/sekai/config/client.toml",
		"node_key.json":      "/sekai/config/node_key.json",
	}

	InterxFiles = InfraFiles{
		"config.json": "/interx/config.json",
	}

	SyslogFiles = InfraFiles{
		"shidai.log": "/syslog-data/syslog-ng/logs/shidai.log",
		"sekai.log":  "/syslog-data/syslog-ng/logs/sekai.log",
		"interx.log": "/syslog-data/syslog-ng/logs/interx.log",
	}
)

func NewDefaultAppConfig() *AppConfig {
	return &AppConfig{
		MinimumGasPrices:    "0stake",
		Pruning:             "custom",
		PruningKeepRecent:   2,
		PruningKeepEvery:    100,
		PruningInterval:     10,
		HaltHeight:          0,
		HaltTime:            0,
		MinRetainBlocks:     0,
		InterBlockCache:     true,
		IndexEvents:         []string{},
		IavlCacheSize:       781250,
		IavlDisableFastNode: true,
		Telemetry: TelemetryConfig{
			ServiceName:             "",
			Enabled:                 false,
			EnableHostname:          false,
			EnableHostnameLabel:     false,
			EnableServiceLabel:      false,
			PrometheusRetentionTime: 0,
			GlobalLabels:            [][]string{},
		},
		API: APIConfig{
			Enable:             false,
			Swagger:            false,
			Address:            "tcp://0.0.0.0:1317",
			MaxOpenConnections: 1000,
			RPCReadTimeout:     10,
			RPCWriteTimeout:    0,
			RPCMaxBodyBytes:    1000000,
			EnableUnsafeCORS:   false,
		},
		Rosetta: RosettaConfig{
			Enable:     false,
			Address:    ":8080",
			Blockchain: "app",
			Network:    "network",
			Retries:    3,
			Offline:    false,
		},
		GRPC: GRPCConfig{
			Enable:  true,
			Address: "0.0.0.0:9090",
		},
		GRPCWeb: GRPCWebConfig{
			Enable:           true,
			Address:          "0.0.0.0:9091",
			EnableUnsafeCORS: false,
		},
		StateSync: StateSyncAppConfig{
			SnapshotInterval:   200,
			SnapshotKeepRecent: 2,
		},
		Wasm: WasmConfig{
			QueryGasLimit: 300000,
		},
	}
}

func NewDefaultConfig() *Config {
	return &Config{
		ProxyApp:               "tcp://127.0.0.1:26658",
		Moniker:                "KIRA VALIDATOR NODE",
		FastSync:               true,
		DBBackend:              "goleveldb",
		DBDir:                  "data",
		LogLevel:               "info",
		LogFormat:              "plain",
		GenesisFile:            "config/genesis.json",
		PrivValidatorKeyFile:   "config/priv_validator_key.json",
		PrivValidatorStateFile: "data/priv_validator_state.json",
		PrivValidatorLaddr:     "",
		NodeKeyFile:            "config/node_key.json",
		ABCI:                   "socket",
		FilterPeers:            false,
		RPC: RPCConfig{
			Laddr:                                "tcp://0.0.0.0:26657",
			CorsAllowedOrigins:                   []string{"*"},
			CorsAllowedMethods:                   []string{"HEAD", "GET", "POST"},
			CorsAllowedHeaders:                   []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "X-Server-Time"},
			GRPCListenAddr:                       "",
			GRPCMaxOpenConnections:               900,
			Unsafe:                               false,
			MaxOpenConnections:                   900,
			MaxSubscriptionClients:               100,
			MaxSubscriptionsPerClient:            5,
			ExperimentalSubscriptionBufferSize:   200,
			ExperimentalWebSocketWriteBufferSize: 200,
			ExperimentalCloseOnSlowClient:        false,
			TimeoutBroadcastTxCommit:             "10s",
			MaxBodyBytes:                         1000000,
			MaxHeaderBytes:                       1048576,
			TLSCertFile:                          "",
			TLSKeyFile:                           "",
			PprofLaddr:                           "localhost:6060",
		},
		P2P: P2PConfig{
			Laddr:                        "tcp://0.0.0.0:26656",
			ExternalAddress:              "", // tcp://IP:PORT
			Seeds:                        "", // tcp://NODE_ID@IP:PORT
			PersistentPeers:              "",
			UPNP:                         false,
			AddrBookFile:                 "config/addrbook.json",
			AddrBookStrict:               false,
			MaxNumInboundPeers:           128,
			MaxNumOutboundPeers:          32,
			UnconditionalPeerIDs:         "",
			PersistentPeersMaxDialPeriod: "0s",
			FlushThrottleTimeout:         "100ms",
			MaxPacketMsgPayloadSize:      131072,
			SendRate:                     65536000,
			RecvRate:                     65536000,
			Pex:                          true,
			SeedMode:                     false,
			PrivatePeerIDs:               "",
			AllowDuplicateIP:             true,
			HandshakeTimeout:             "60s",
			DialTimeout:                  "30s",
		},
		Mempool: MempoolConfig{
			Version:               "v0",
			Recheck:               true,
			Broadcast:             true,
			WALDir:                "",
			Size:                  5000,
			MaxTxsBytes:           131072000,
			CacheSize:             10000,
			KeepInvalidTxsInCache: false,
			MaxTxBytes:            131072,
			MaxBatchBytes:         0,
			TTLDuration:           "0s",
			TTLNumBlocks:          0,
		},
		StateSync: StateSyncConfig{
			Enable:              false,
			RPCServers:          "",
			TrustHeight:         0,
			TrustHash:           "",
			TrustPeriod:         "168h0m0s",
			DiscoveryTime:       "15s",
			TempDir:             "/tmp",
			ChunkRequestTimeout: "10s",
			ChunkFetchers:       4,
		},
		FastSyncConfig: FastSyncConfig{
			Version: "v1",
		},
		Consensus: ConsensusConfig{
			WALFile:                     "data/cs.wal/wal",
			TimeoutPropose:              "3s",
			TimeoutProposeDelta:         "500ms",
			TimeoutPrevote:              "1s",
			TimeoutPrevoteDelta:         "500ms",
			TimeoutPrecommit:            "1s",
			TimeoutPrecommitDelta:       "500ms",
			TimeoutCommit:               "3000ms",
			DoubleSignCheckHeight:       0,
			SkipTimeoutCommit:           false,
			CreateEmptyBlocks:           true,
			CreateEmptyBlocksInterval:   "20s",
			PeerGossipSleepDuration:     "100ms",
			PeerQueryMaj23SleepDuration: "2s",
		},
		Storage: StorageConfig{
			DiscardABCIMResponses: false,
		},
		TxIndexer: TxIndexerConfig{
			Indexer:  "kv",
			PSQLConn: "",
		},
		Instrumentation: InstrumentationConfig{
			Prometheus:           true,
			PrometheusListenAddr: ":26660",
			MaxOpenConnections:   3,
			Namespace:            "tendermint",
		},
	}
}
