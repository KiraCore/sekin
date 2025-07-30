package networkparser

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	// interxendpoint "github.com/KiraCore/kensho/types/interxEndpoint"
	sekaihelper "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/sekai_helper"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	sekaiendpoint "github.com/kiracore/sekin/src/shidai/internal/types/endpoints/sekai"
)

// crawl network over sekaid's endpoint
type SekaiNetworkParser struct {
	mu sync.Mutex
}

func NewSekaiNetworkParser() *SekaiNetworkParser {
	return &SekaiNetworkParser{}

}

type Node struct {
	IP      string
	ID      string
	Peers   []Node
	NCPeers int
}

type BlacklistedNode struct {
	IP    string
	Error []error
}

func (np *SekaiNetworkParser) Scan(ctx context.Context, firstNodeIP string, port, depth int, ignoreDepth bool) (map[string]Node, map[string]BlacklistedNode, error) {
	nodePool := make(map[string]Node)
	blacklist := make(map[string]BlacklistedNode)
	processed := make(map[string]string)
	client := http.DefaultClient
	node, err := sekaihelper.GetNetInfo(ctx, firstNodeIP, strconv.Itoa(port))
	if err != nil {
		return nil, nil, err
	}

	var wg sync.WaitGroup
	for _, n := range node.Result.Peers {
		wg.Add(1)
		go np.loopFunc(ctx, &wg, client, nodePool, blacklist, processed, n.RemoteIP, 0, depth, ignoreDepth)
	}

	wg.Wait()
	fmt.Println()
	log.Printf("\nTotal saved peers:%v\nOriginal node peer count: %v\nBlacklisted nodes(not reachable): %v\n", len(nodePool), len(node.Result.Peers), len(blacklist))
	// log.Printf("BlackListed: %+v ", blacklist)
	return nodePool, blacklist, nil
}

func (np *SekaiNetworkParser) loopFunc(ctx context.Context, wg *sync.WaitGroup, client *http.Client, pool map[string]Node, blacklist map[string]BlacklistedNode, processed map[string]string, ip string, currentDepth, totalDepth int, ignoreDepth bool) {

	defer wg.Done()
	if !ignoreDepth {
		if currentDepth >= totalDepth {
			// log.Printf("DEPTH LIMIT REACHED")
			return
		}
	}

	// log.Printf("Current depth: %v, IP: %v", currentDepth, ip)

	np.mu.Lock()
	if _, exist := blacklist[ip]; exist {
		np.mu.Unlock()
		// log.Printf("BLACKLISTED: %v", ip)
		return
	}
	if _, exist := pool[ip]; exist {
		np.mu.Unlock()
		// log.Printf("ALREADY EXIST: %v", ip)
		return
	}
	if _, exist := processed[ip]; exist {
		np.mu.Unlock()
		// log.Printf("ALREADY PROCESSED: %v", ip)
		return
	} else {
		processed[ip] = ip
	}
	np.mu.Unlock()

	currentDepth++

	// var nodeInfo *interxendpoint.NetInfo
	var nodeInfo *sekaiendpoint.NetInfo
	// var status *interxendpoint.Status
	var status *sekaiendpoint.Status
	var errNetInfo error
	var errStatus error

	//local wait group
	var localWaitGroup sync.WaitGroup
	localWaitGroup.Add(2)
	go func() {
		defer localWaitGroup.Done()
		nodeInfo, errNetInfo = sekaihelper.GetNetInfo(ctx, ip, strconv.Itoa(types.DEFAULT_RPC_PORT))
		// nodeInfo, errNetInfo = GetNetInfoFromInterx(ctx, client, ip)
	}()
	go func() {
		defer localWaitGroup.Done()
		// status, errStatus = GetStatusFromInterx(ctx, client, ip)
		status, errStatus = sekaihelper.GetSekaidStatus(ctx, ip, strconv.Itoa(types.DEFAULT_RPC_PORT))
	}()
	localWaitGroup.Wait()
	// nodeInfo, errNetInfo = GetNetInfoFromInterx(ctx, client, ip)
	// status, errStatus = GetStatusFromInterx(ctx, client, ip)

	if errNetInfo != nil || errStatus != nil {
		// log.Printf("%v", err.Error())
		np.mu.Lock()
		log.Printf("adding <%v> to blacklist", ip)
		blacklist[ip] = BlacklistedNode{IP: ip, Error: []error{errNetInfo, errStatus}}
		cleanValue(processed, ip)
		np.mu.Unlock()
		// defer localWaitGroup.Done()
		return
	}

	np.mu.Lock()
	log.Printf("adding <%v> to the pool, nPeers: %v", ip, nodeInfo.Result.NPeers)

	nPeers, err := strconv.Atoi(nodeInfo.Result.NPeers)
	if err != nil {
		log.Printf("unable to parse %v value %v", nodeInfo.Result.NPeers, err)
		return
	}
	node := Node{
		IP:      ip,
		NCPeers: nPeers,
		ID:      status.Result.NodeInfo.ID,
	}

	for _, nn := range nodeInfo.Result.Peers {
		ip, port, err := extractIP(nn.NodeInfo.ListenAddr)
		if err != nil {
			continue
		}
		node.Peers = append(node.Peers, Node{IP: fmt.Sprintf("%v:%v", ip, port), ID: nn.NodeInfo.ID})
	}

	pool[ip] = node
	cleanValue(processed, ip)
	np.mu.Unlock()

	for _, p := range nodeInfo.Result.Peers {
		wg.Add(1)
		go np.loopFunc(ctx, wg, client, pool, blacklist, processed, p.RemoteIP, currentDepth, totalDepth, ignoreDepth)

		listenAddr, _, err := extractIP(p.NodeInfo.ListenAddr)
		if err != nil {
			continue
		} else {
			if listenAddr != p.RemoteIP {
				log.Printf("listen addr (%v) and remoteIp (%v) are not the same, creating new goroutine for listen addr", listenAddr, p.RemoteIP)
				wg.Add(1)
				go np.loopFunc(ctx, wg, client, pool, blacklist, processed, listenAddr, currentDepth, totalDepth, ignoreDepth)
			}
		}

	}

}

func cleanValue(toClean map[string]string, key string) {
	delete(toClean, key)
}

func extractIP(input string) (ip string, port string, err error) {
	// Regular expression to match IP addresses
	re := regexp.MustCompile(`tcp://([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+):([0-9]+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("no IP address or port found in input")
	}
	return matches[1], matches[2], nil
}
