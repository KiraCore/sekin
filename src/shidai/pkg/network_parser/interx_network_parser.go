package networkparser

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	// interxendpoint "github.com/KiraCore/kensho/types/interxEndpoint"

	interxhelper "github.com/kiracore/sekin/src/shidai/internal/interx_handler/interx_helper"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	interxv2 "github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx_V2"
)

type InterxNetworkParser struct {
	mu sync.Mutex
}

func NewInterxNetworkParser() *InterxNetworkParser {
	return &InterxNetworkParser{}

}

// get nodes that are available by 11000 port
func (np *InterxNetworkParser) Scan(ctx context.Context, firstNode string, port, depth int, ignoreDepth bool) (map[string]Node, map[string]BlacklistedNode, error) {
	nodesPool := make(map[string]Node)
	blacklist := make(map[string]BlacklistedNode)
	processed := make(map[string]string)
	client := http.DefaultClient
	node, err := interxhelper.GetNetInfoV2(ctx, firstNode, port)
	if err != nil {
		return nil, nil, err
	}

	var wg sync.WaitGroup
	for _, n := range node.Peers {
		wg.Add(1)
		go np.loopFunc(ctx, &wg, client, nodesPool, blacklist, processed, n.RemoteIP, 0, depth, ignoreDepth)
	}

	wg.Wait()
	fmt.Println()
	log.Printf("\nTotal saved peers:%v\nOriginal node peer count: %v\nBlacklisted nodes(not reachable): %v\n", len(nodesPool), len(node.Peers), len(blacklist))
	// log.Printf("BlackListed: %+v ", blacklist)

	return nodesPool, blacklist, nil
}

func (np *InterxNetworkParser) loopFunc(ctx context.Context, wg *sync.WaitGroup, client *http.Client, pool map[string]Node, blacklist map[string]BlacklistedNode, processed map[string]string, ip string, currentDepth, totalDepth int, ignoreDepth bool) {

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
	// interxv2
	var nodeInfo *interxv2.NetInfo
	var status *interxv2.Status
	var errNetInfo error
	var errStatus error

	//local wait group
	var localWaitGroup sync.WaitGroup
	localWaitGroup.Add(2)
	go func() {
		defer localWaitGroup.Done()
		nodeInfo, errNetInfo = interxhelper.GetNetInfoV2(ctx, ip, types.DEFAULT_INTERX_PORT)
	}()
	go func() {
		defer localWaitGroup.Done()
		status, errStatus = interxhelper.GetInterxStatusV2(ctx, ip, types.DEFAULT_INTERX_PORT)
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
	log.Printf("adding <%v> to the pool, nPeers: %v", ip, nodeInfo.NPeers)
	npeers, err := strconv.Atoi(nodeInfo.NPeers)
	if err != nil {
		return
	}
	node := Node{
		IP:      ip,
		NCPeers: npeers,
		ID:      status.NodeInfo.ID,
	}

	for _, nn := range nodeInfo.Peers {
		ip, port, err := extractIP(nn.NodeInfo.ListenAddr)
		if err != nil {
			continue
		}
		node.Peers = append(node.Peers, Node{IP: fmt.Sprintf("%v:%v", ip, port), ID: nn.NodeInfo.ID})
	}

	pool[ip] = node
	cleanValue(processed, ip)
	np.mu.Unlock()

	for _, p := range nodeInfo.Peers {
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
