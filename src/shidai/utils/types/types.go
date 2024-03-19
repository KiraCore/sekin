package config

import "time"

type (
	// TomlValue represents a configuration value to be updated in the '*.toml' file of the 'sekaid' application.
	TomlValue struct {
		Tag   string
		Name  string
		Value string
	}

	// JsonValue represents a configuration value to be updated in the '*.json' file of the 'interx' application
	JsonValue struct {
		Key   string // Dot-separated keys by nesting
		Value any
	}

	// KiraConfig is a configuration for sekaid or interx manager.
	ShidaiConfig struct {
		NetworkName            string        // Name of a blockchain name (chain-ID)
		SekaidHome             string        // Home folder for sekai bin
		InterxHome             string        // Home folder for interx bin
		RpcPort                string        // Sekaid's rpc port
		GrpcPort               string        // Sekaid's grpc port
		P2PPort                string        // Sekaid's p2p port
		PrometheusPort         string        // Prometheus port
		InterxPort             string        // Interx endpoint port
		Moniker                string        // Moniker
		TimeBetweenBlocks      time.Duration // Awaiting time between blocks
		SekaiContainerAddress  string
		InterxContainerAddress string
		SecretsFolder          string
	}
)

func DefaultShidaiConfig() *ShidaiConfig {
	return &ShidaiConfig{
		NetworkName:       "shidaiNet-1",
		RpcPort:           "26657",
		P2PPort:           "26656",
		GrpcPort:          "9090",
		PrometheusPort:    "26660",
		InterxPort:        "11000",
		Moniker:           "VALIDATOR",
		SekaidHome:        "/sekai/sekaid",
		InterxHome:        "/interx/interxd",
		TimeBetweenBlocks: time.Second * 10,
	}
}
