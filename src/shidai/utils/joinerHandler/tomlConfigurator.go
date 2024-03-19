package joinerhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	tomlEditor "shidai/utils/TomlEditor"
	httpexecutor "shidai/utils/httpExecutor"
	utilsTypes "shidai/utils/types"
	"strings"
	"sync"
	"time"
)

const endpointPubP2PList string = "api/pub_p2p_list?peers_only=true"
const endpointStatus string = "status"

type networkInfo struct {
	NetworkName string
	NodeID      string
	BlockHeight string
	Seeds       []string
}

type TargetSeedKiraConfig struct {
	IpAddress     string
	InterxPort    string
	SekaidRPCPort string
	SekaidP2PPort string
}

type syncInfo struct {
	rpcServers       []string
	trustHeightBlock string
	trustHashBlock   string
}
type ResponseBlock struct {
	Result struct {
		BlockID struct {
			Hash string `json:"hash"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}

type responseSekaidStatus struct {
	Result struct {
		NodeInfo struct {
			ID      string `json:"id"`
			Network string `json:"network"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHeight string    `json:"latest_block_height"`
			LatestBlockTime   time.Time `json:"latest_block_time"`
			CatchingUp        bool      `json:"catching_up"`
		} `json:"sync_info"`
	} `json:"result"`
}

func ApplyJoinerTomlSettings(sekaidHome string, tc *TargetSeedKiraConfig, cfg *utilsTypes.ShidaiConfig) error {
	ctx := context.Background()

	info, err := retrieveNetworkInformation(ctx, tc)
	if err != nil {
		return err
	}
	log.Printf("DEBUG: info: %+v", info)

	standardTomlValues := tomlEditor.GetStandardConfigPack(cfg)
	// Get config for config.toml
	configFromSeed, err := getConfigsBasedOnSeed(ctx, info, tc)
	if err != nil {
		return err
	}
	updates := append(standardTomlValues, configFromSeed...)
	// apply new config for toml.config
	err = tomlEditor.ApplyNewConfig(ctx, updates, fmt.Sprintf("%v/config/config.toml", sekaidHome))
	if err != nil {
		return err
	}

	err = tomlEditor.ApplyNewConfig(ctx, GetJoinerAppConfig(9090), fmt.Sprintf("%v/config/app.toml", sekaidHome))
	if err != nil {
		return err
	}

	return nil
}

func retrieveNetworkInformation(ctx context.Context, tc *TargetSeedKiraConfig) (*networkInfo, error) {

	statusResponse, err := getSekaidStatus(ctx, tc.IpAddress, tc.SekaidRPCPort)
	if err != nil {
		return nil, fmt.Errorf("getting sekaid status: %w", err)
	}

	// TODO: rewrite for 26657/netInfo instead of interx pub_p2p_list
	pupP2PListResponse, err := getPubP2PList(ctx, tc.IpAddress, tc.InterxPort)
	if err != nil {
		return nil, fmt.Errorf("getting sekaid public P2P list: %w", err)
	}

	listOfSeeds, err := parsePubP2PListResponse(pupP2PListResponse)
	if err != nil {
		return nil, fmt.Errorf("parsing sekaid public P2P list %w", err)
	}
	if len(listOfSeeds) == 0 {
		log.Printf("ERROR: List of seeds is empty, the trusted seed will be used")
		listOfSeeds = []string{fmt.Sprintf("tcp://%s@%s:%s", statusResponse.Result.NodeInfo.ID, tc.IpAddress, tc.SekaidP2PPort)}
	}

	return &networkInfo{
		NetworkName: statusResponse.Result.NodeInfo.Network,
		NodeID:      statusResponse.Result.NodeInfo.ID,
		BlockHeight: statusResponse.Result.SyncInfo.LatestBlockHeight,
		Seeds:       listOfSeeds,
	}, nil
}

func getSekaidStatus(ctx context.Context, ipAddress, rpcPort string) (*responseSekaidStatus, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, rpcPort, endpointStatus)
	client := &http.Client{}
	body, err := httpexecutor.DoGetHttpQuery(ctx, client, url)
	if err != nil {
		log.Printf("ERROR: Querying error: %s", err)
		return nil, err
	}

	var response *responseSekaidStatus
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Printf("ERROR: Can't parse JSON response: %s", err)
		return nil, err
	}

	return response, nil
}

// Interx p2p list
func getPubP2PList(ctx context.Context, ipAddress, interxPort string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, endpointPubP2PList)
	client := &http.Client{}
	body, err := httpexecutor.DoGetHttpQuery(ctx, client, url)
	if err != nil {
		log.Printf("ERROR: Querying error: %s", err)
		return nil, err
	}

	return body, nil
}

func parsePubP2PListResponse(seedsResponse []byte) ([]string, error) {
	if len(seedsResponse) == 0 {
		log.Printf("WARNING: The list of public seeds is not available")
		return nil, nil
	}

	linesOfPeers := strings.Split(string(seedsResponse), "\n")
	listOfSeeds := make([]string, 0)

	for _, line := range linesOfPeers {
		formattedSeed := fmt.Sprintf("tcp://%s", line)
		log.Printf("DEBUG: Got seed: %s", formattedSeed)
		listOfSeeds = append(listOfSeeds, formattedSeed)
	}

	return listOfSeeds, nil
}

// getConfigsBasedOnSeed generates a slice of configuration values based on the provided network information
// and joins the seeds, RPC servers, and other relevant parameters into the configuration values.
func getConfigsBasedOnSeed(ctx context.Context, netInfo *networkInfo, tc *TargetSeedKiraConfig) ([]utilsTypes.TomlValue, error) {
	configValues := make([]utilsTypes.TomlValue, 0)

	configValues = append(configValues, utilsTypes.TomlValue{Tag: "p2p", Name: "seeds", Value: strings.Join(netInfo.Seeds, ",")})

	listOfRPC, err := parseRPCfromSeedsList(netInfo.Seeds, tc)
	if err != nil {
		return nil, fmt.Errorf("parsing RPCs from seeds list %w", err)
	}

	syncInfo, err := getSyncInfo(ctx, listOfRPC, netInfo.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("getting sync information %w", err)
	}

	if syncInfo != nil {
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "trust_hash", Value: syncInfo.trustHashBlock})
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "trust_height", Value: syncInfo.trustHeightBlock})
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "rpc_servers", Value: strings.Join(syncInfo.rpcServers, ",")})
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "trust_period", Value: "168h0m0s"})
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "enable", Value: "true"})
		configValues = append(configValues, utilsTypes.TomlValue{Tag: "statesync", Name: "temp_dir", Value: "/tmp"})
	}
	log.Printf("DEBUG: configValues %+v", configValues)
	// return nil, fmt.Errorf("TestError")
	return configValues, nil
}

func GetJoinerAppConfig(grpcPort uint) []utilsTypes.TomlValue {
	return []utilsTypes.TomlValue{
		{Tag: "state-sync", Name: "snapshot-interval", Value: "200"},
		{Tag: "state-sync", Name: "snapshot-keep-recent", Value: "2"},
		{Tag: "", Name: "pruning", Value: "custom"},
		{Tag: "", Name: "pruning-keep-recent", Value: "2"},
		{Tag: "", Name: "pruning-keep-every", Value: "100"},
		{Tag: "", Name: "pruning-interval", Value: "10"},
		{Tag: "grpc", Name: "address", Value: fmt.Sprintf("0.0.0.0:%v", grpcPort)},
	}
}

func parseRPCfromSeedsList(seeds []string, tc *TargetSeedKiraConfig) ([]string, error) {

	listOfRPCs := make([]string, 0)

	for _, seed := range seeds {
		// tcp://23ca3770ae3874ac8f5a6f84a5cfaa1b39e49fc9@128.140.86.241:26656 -> 128.140.86.241:26657
		parts := strings.Split(seed, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid seed format")
		}

		ipAndPort := strings.Split(parts[1], ":")
		if len(ipAndPort) != 2 {
			return nil, fmt.Errorf("invalid port format")
		}

		rpc := fmt.Sprintf("%s:%s", ipAndPort[0], tc.SekaidRPCPort)
		log.Printf("Adding rpc to list: %s", rpc)
		listOfRPCs = append(listOfRPCs, rpc)
	}

	return listOfRPCs, nil
}

// getSyncInfo retrieves synchronization information based on a list of RPC servers and a minimum block height.
// It queries each RPC server for block information at the specified height and checks if the retrieved data is consistent.
func getSyncInfo(ctx context.Context, listOfRPC []string, minHeight string) (*syncInfo, error) {
	resultSyncInfo := &syncInfo{
		rpcServers:       []string{},
		trustHeightBlock: "",
		trustHashBlock:   "",
	}

	rpcServersChannel := make(chan string)
	var wg sync.WaitGroup
	wg.Add(len(listOfRPC))
	go func() {
		for rpcServer := range rpcServersChannel {
			resultSyncInfo.rpcServers = append(resultSyncInfo.rpcServers, rpcServer)
		}
	}()

	for _, rpcServer := range listOfRPC {
		go func() {
			defer wg.Done()
			responseBlock, err := getBlockInfo(ctx, rpcServer, minHeight)
			if err != nil {
				log.Printf("Can't get block information from RPC '%s'", rpcServer)
				return
			}
			if responseBlock.Result.Block.Header.Height != minHeight {
				log.Printf("RPC (%s) height is '%s', but expected '%s'", rpcServer, responseBlock.Result.Block.Header.Height, minHeight)
				return
			}

			if responseBlock.Result.BlockID.Hash != resultSyncInfo.trustHashBlock && resultSyncInfo.trustHashBlock != "" {
				log.Printf("RPC (%s) hash is '%s', but expected '%s'", rpcServer, responseBlock.Result.BlockID.Hash, resultSyncInfo.trustHashBlock)
				return
			}

			resultSyncInfo.trustHashBlock = responseBlock.Result.BlockID.Hash
			resultSyncInfo.trustHeightBlock = minHeight

			log.Printf("Adding RPC (%s) to RPC connection list", rpcServer)
			rpcServersChannel <- rpcServer
		}()
	}
	wg.Wait()
	close(rpcServersChannel)

	if len(resultSyncInfo.rpcServers) < 2 {
		log.Printf("Sync is NOT possible (not enough RPC servers)")
		return nil, nil
	}

	log.Printf("%+v", resultSyncInfo)
	log.Printf("DEBUG: amount of rpc servers: %v", len(resultSyncInfo.rpcServers))
	return resultSyncInfo, nil
}

// getBlockInfo queries block information from a specified RPC server at a given block height using an HTTP GET request.
// It constructs the URL based on the provided RPC server URL and the endpointBlock with the specified minHeight parameter.
// The function then makes an HTTP GET request to retrieve the block information as a ResponseBlock struct.
func getBlockInfo(ctx context.Context, rpcServer, blockHeight string) (*ResponseBlock, error) {
	endpointBlock := fmt.Sprintf("block?height=%s", blockHeight)
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	url := fmt.Sprintf("http://%s/%s", rpcServer, endpointBlock)
	client := &http.Client{}
	body, err := httpexecutor.DoGetHttpQuery(ctx, client, url)
	if err != nil {
		return nil, fmt.Errorf("can't reach block response %w", err)
	}

	var response *ResponseBlock
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("can't parse JSON response %w", err)
	}

	return response, nil
}
