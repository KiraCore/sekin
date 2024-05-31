package configconstructor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KiraCore/kensho/helper/networkparser"
	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/sekai"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

const (
	endpointPubP2PList string = "api/pub_p2p_list?peers_only=true"
	endpointStatus     string = "status"
)

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
	trustHeightBlock int
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

func FormSekaiJoinerConfigs(tc *TargetSeedKiraConfig) error {
	ctx := context.Background()

	info, err := retrieveNetworkInformation(ctx, tc)
	if err != nil {
		return err
	}

	configToml, err := getConfigsBasedOnSeed(ctx, info, tc, types.NewDefaultConfig())
	if err != nil {
		return err
	}
	pubIP, err := GetPublicIP()
	if err != nil {
		return err
	}
	configToml.P2P.ExternalAddress = fmt.Sprintf("tcp://%v:%v", pubIP, types.DEFAULT_P2P_PORT)
	log.Printf("%+v", configToml)

	configTomlSavePath := path.Join(types.SEKAI_HOME, "config", "config.toml")

	err = utils.SaveConfig(configTomlSavePath, *configToml)
	if err != nil {
		return err
	}

	appTomlSavePath := path.Join(types.SEKAI_HOME, "config", "app.toml")
	appToml := GetJoinerAppConfig(types.NewDefaultAppConfig())
	err = utils.SaveAppConfig(appTomlSavePath, *appToml)
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

	nodes, _, err := networkparser.GetAllNodesV3(ctx, tc.IpAddress, 3, false)
	if err != nil {
		return nil, fmt.Errorf("unable to parse peers: %w", err)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var listOfSeeds []string

	for _, node := range nodes {
		wg.Add(1)
		go func(n networkparser.Node) {
			defer wg.Done()
			pupP2PListResponse, err := getPubP2PList(ctx, n.IP, "11000")
			if err != nil {
				zap.L().Debug("getting sekaid public P2P list:", zap.Error(err))
				return
			}
			local, err := parsePubP2PListResponse(pupP2PListResponse)
			if err != nil {
				zap.L().Debug("parsing sekaid public P2P list", zap.Error(err))
				return
			}
			mu.Lock()
			for _, s := range local {
				exist := false
				for _, ss := range listOfSeeds {
					if s == ss {
						exist = true
						// zap.L().Debug("already exist", zap.String("IP", ss))
						log.Println("already exist", zap.String("IP", ss))
						break
					}
				}
				if !exist {
					zap.L().Debug("adding peer", zap.String("IP", s))
					listOfSeeds = append(listOfSeeds, s)
				}
			}
			mu.Unlock()
		}(node)

	}
	wg.Wait()

	if len(listOfSeeds) == 0 {
		zap.L().Debug("ERROR: List of seeds is empty, the trusted seed will be used")
		listOfSeeds = []string{fmt.Sprintf("tcp://%s@%s:%s", statusResponse.Result.NodeInfo.ID, tc.IpAddress, tc.SekaidP2PPort)}
	}
	zap.L().Debug("LIST OF seeds ==================================== ", zap.Strings("listOfSeeds", listOfSeeds))
	return &networkInfo{
		NetworkName: statusResponse.Result.NodeInfo.Network,
		NodeID:      statusResponse.Result.NodeInfo.ID,
		BlockHeight: statusResponse.Result.SyncInfo.LatestBlockHeight,
		Seeds:       listOfSeeds,
	}, nil
}

func getSekaidStatus(ctx context.Context, ipAddress, rpcPort string) (*sekai.Status, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, rpcPort, endpointStatus)
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}

	var response *sekai.Status
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Interx p2p list
func getPubP2PList(ctx context.Context, ipAddress, interxPort string) ([]byte, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, interxPort, endpointPubP2PList)
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}

	return body, nil
}

func parsePubP2PListResponse(seedsResponse []byte) ([]string, error) {
	if len(seedsResponse) == 0 {
		return nil, nil
	}

	linesOfPeers := strings.Split(string(seedsResponse), "\n")
	listOfSeeds := make([]string, 0)

	for _, line := range linesOfPeers {
		formattedSeed := fmt.Sprintf("tcp://%s", line)
		listOfSeeds = append(listOfSeeds, formattedSeed)
	}

	return listOfSeeds, nil
}

// getConfigsBasedOnSeed generates a slice of configuration values based on the provided network information
// and joins the seeds, RPC servers, and other relevant parameters into the configuration values.
func getConfigsBasedOnSeed(ctx context.Context, netInfo *networkInfo, tc *TargetSeedKiraConfig, cfgToUpdate *types.Config) (*types.Config, error) {

	// configValues = append(configValues, utilsTypes.TomlValue{Tag: "p2p", Name: "seeds", Value: strings.Join(netInfo.Seeds, ",")})
	cfgToUpdate.P2P.Seeds = strings.Join(netInfo.Seeds, ",")
	listOfRPC, err := parseRPCfromSeedsList(netInfo.Seeds, tc)
	if err != nil {
		return nil, fmt.Errorf("parsing RPCs from seeds list %w", err)
	}

	blockHeight, err := strconv.Atoi(netInfo.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %v to int %w", netInfo.BlockHeight, err)
	}

	syncInfo, err := getSyncInfo(ctx, listOfRPC, blockHeight)
	if err != nil {
		return nil, fmt.Errorf("getting sync information %w", err)
	}

	if syncInfo != nil {
		cfgToUpdate.StateSync.TrustHash = syncInfo.trustHashBlock
		cfgToUpdate.StateSync.TrustHeight = syncInfo.trustHeightBlock
		cfgToUpdate.StateSync.RPCServers = strings.Join(syncInfo.rpcServers, ",")
		cfgToUpdate.StateSync.TrustPeriod = "168h0m0s"
		cfgToUpdate.StateSync.Enable = true
		cfgToUpdate.StateSync.TempDir = "/tmp"

	}
	zap.L().Debug(" Config Values ", zap.Any("configValues", cfgToUpdate))
	// return nil, fmt.Errorf("TestError")
	return cfgToUpdate, nil
}

func GetJoinerAppConfig(config *types.AppConfig) *types.AppConfig {
	// return []utilsTypes.TomlValue{
	// 	{Tag: "state-sync", Name: "snapshot-interval", Value: "200"},
	// 	{Tag: "state-sync", Name: "snapshot-keep-recent", Value: "2"},
	// 	{Tag: "", Name: "pruning", Value: "custom"},
	// 	{Tag: "", Name: "pruning-keep-recent", Value: "2"},
	// 	{Tag: "", Name: "pruning-keep-every", Value: "100"},
	// 	{Tag: "", Name: "pruning-interval", Value: "10"},
	// 	{Tag: "grpc", Name: "address", Value: fmt.Sprintf("0.0.0.0:%v", grpcPort)},
	// }
	config.StateSync.SnapshotInterval = 200
	config.StateSync.SnapshotKeepRecent = 2
	config.Pruning = "custom"
	config.PruningKeepRecent = 2
	config.PruningKeepEvery = 100
	config.PruningInterval = 10
	config.GRPC.Address = fmt.Sprintf("0.0.0.0:%v", types.DEFAULT_GRPC_PORT)

	return config
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
		listOfRPCs = append(listOfRPCs, rpc)
	}

	return listOfRPCs, nil
}

// getSyncInfo retrieves synchronization information based on a list of RPC servers and a minimum block height.
// It queries each RPC server for block information at the specified height and checks if the retrieved data is consistent.
func getSyncInfo(ctx context.Context, listOfRPC []string, minHeight int) (*syncInfo, error) {
	resultSyncInfo := &syncInfo{
		rpcServers:       []string{},
		trustHeightBlock: 0,
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

	for _, r := range listOfRPC {
		go func(rpcServer string) {
			defer wg.Done()
			responseBlock, err := getBlockInfo(ctx, rpcServer, minHeight)
			if err != nil {
				zap.L().Debug("Can't get block information from RPC ", zap.String("rpcServer", rpcServer))
				return
			}
			if responseBlock.Result.Block.Header.Height != strconv.Itoa(minHeight) {
				zap.L().Debug("RPC (%s) height is '%s', but expected '%v'", zap.String("rpcServer", rpcServer), zap.String("responseBlock.Result.Block.Header.Height", responseBlock.Result.Block.Header.Height), zap.Int("minHeight", minHeight))
				return
			}

			if responseBlock.Result.BlockID.Hash != resultSyncInfo.trustHashBlock && resultSyncInfo.trustHashBlock != "" {
				zap.L().Debug("RPC (%s) hash is '%s', but expected '%s'", zap.String("rpcServer", rpcServer), zap.String("responseBlock.Result.BlockID.Hash", responseBlock.Result.BlockID.Hash), zap.String("resultSyncInfo.trustHashBlock", resultSyncInfo.trustHashBlock))
				return
			}

			resultSyncInfo.trustHashBlock = responseBlock.Result.BlockID.Hash
			resultSyncInfo.trustHeightBlock = minHeight

			zap.L().Debug("Adding RPC (%s) to RPC connection list", zap.String("rpcServer", rpcServer))
			rpcServersChannel <- rpcServer
		}(r)
	}
	wg.Wait()
	close(rpcServersChannel)

	if len(resultSyncInfo.rpcServers) < 2 {
		zap.L().Debug("Sync is NOT possible (not enough RPC servers)")
		return nil, nil
	}

	zap.L().Debug("Result sync info", zap.Any("resultSyncInfo", resultSyncInfo))
	zap.L().Debug("Amount of rpc servers", zap.Int("", len(resultSyncInfo.rpcServers)))
	return resultSyncInfo, nil
}

// getBlockInfo queries block information from a specified RPC server at a given block height using an HTTP GET request.
// It constructs the URL based on the provided RPC server URL and the endpointBlock with the specified minHeight parameter.
// The function then makes an HTTP GET request to retrieve the block information as a ResponseBlock struct.
func getBlockInfo(ctx context.Context, rpcServer string, blockHeight int) (*ResponseBlock, error) {
	endpointBlock := fmt.Sprintf("block?height=%v", blockHeight)
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	url := fmt.Sprintf("http://%s/%s", rpcServer, endpointBlock)
	client := &http.Client{}
	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
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

func GetPublicIP() (string, error) {
	services := []string{
		"http://ifconfig.me",
		"http://api.ipify.org",
		"http://checkip.amazonaws.com",
	}

	var ip string
	var err error
	for _, service := range services {
		ip, err = fetchIP(service)
		if err == nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get public IP from all services: %v", err)
}

// fetchIP retrieves the public IP address from a single service
func fetchIP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get IP address from %s, status code: %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
