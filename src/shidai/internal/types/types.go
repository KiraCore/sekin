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
)

const (
	SEKAI_HOME  string = "/sekai"
	INTERX_HOME string = "/interx"

	// TODO: if plans change and interx manager will change default port to 11000 - switch back (for network parser)
	// DEFAULT_INTERX_PORT int    = 11000
	DEFAULT_INTERX_PORT int = 80

	DEFAULT_P2P_PORT  int = 26656
	DEFAULT_RPC_PORT  int = 26657
	DEFAULT_GRPC_PORT int = 9090
	DEFAULT_API_PORT  int = 1317

	SEKAI_CONFIG_FOLDER  string = SEKAI_HOME + "/config"
	INTERX_ADDRBOOK_PATH string = INTERX_HOME + "/addrbook.json"
	SEKAI_ADDRBOOK_PATH  string = SEKAI_CONFIG_FOLDER + "/addrbook.json"

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

	InvalidRequest = "invalid request"

	FilePermRO os.FileMode = 0444
	FilePermRW os.FileMode = 0644
	FilePermEX os.FileMode = 0755

	DirPermRO os.FileMode = 0555
	DirPermWR os.FileMode = 0755

	UPDATER_BIN_PATH         = "/updater"
	SEKIN_LATEST_COMPOSE_URL = "https://raw.githubusercontent.com/KiraCore/sekin/main/compose.yml"

	SIGKILL string = "SIGKILL" // 9 - interx
	SIGTERM string = "SIGTERM" // 15 - sekai
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

	ErrInvalidRequest = errors.New(InvalidRequest)

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

// interx endpoints
const (
	ENDPOINT_INTERX_STATUS string = "api/status"
	// ENDPOINT_INTERX_NET_INFO string = "api/net_info" // old
	ENDPOINT_INTERX_NET_INFO string = "api/tendermint/net_info"

	// ENDPOINT_INTERX_GENESIS  string = "api/genesis" // old
	ENDPOINT_INTERX_GENESIS  string = "api/tendermint/genesis"
	ENDPOINT_INTERX_VALOPERS string = "api/valopers"

	ENDPOINT_INTERX_DASHBOARD string = "dashboard"

	// ENDPOINT_INTERX_GENSUM   string = "api/gensum" // not working at all
)
