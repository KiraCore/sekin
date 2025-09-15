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
	sekaihelper "github.com/kiracore/sekin/src/shidai/internal/sekai_handler/sekai_helper"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	interxv2 "github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx_V2"
	sekaiendpoint "github.com/kiracore/sekin/src/shidai/internal/types/endpoints/sekai"
	"golang.org/x/sync/errgroup"
)

type CombinedNetworkParser struct {
	mu sync.Mutex
}

func NewCombinedNetworkParser() *CombinedNetworkParser {
	return &CombinedNetworkParser{}

}

// get nodes that are available by 11000 port
func (np *CombinedNetworkParser) Scan(ctx context.Context, firstNode string, port, depth int, ignoreDepth bool) (map[string]Node, map[string]BlacklistedNode, error) {
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

func (np *CombinedNetworkParser) loopFunc(ctx context.Context, wg *sync.WaitGroup, client *http.Client, pool map[string]Node, blacklist map[string]BlacklistedNode, processed map[string]string, ip string, currentDepth, totalDepth int, ignoreDepth bool) {

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
	node, err := fetchNode(ctx, ip)
	if err != nil {
		np.mu.Lock()
		log.Printf("adding <%v> to blacklist", ip)
		blacklist[ip] = BlacklistedNode{IP: ip, Error: []error{err}}
		cleanValue(processed, ip)
		np.mu.Unlock()
		return
	}

	np.mu.Lock()
	log.Printf("adding <%v> to the pool, nPeers: %v", ip, node.NCPeers)
	// ... add `node` to your pool ...

	pool[ip] = *node
	cleanValue(processed, ip)
	np.mu.Unlock()

	for _, p := range node.Peers {
		wg.Add(1)
		go np.loopFunc(ctx, wg, client, pool, blacklist, processed, p.IP, currentDepth, totalDepth, ignoreDepth)

		listenAddr, _, err := extractIP(p.IP)
		if err != nil {
			continue
		} else {
			if listenAddr != p.IP {
				log.Printf("listen addr (%v) and remoteIp (%v) are not the same, creating new goroutine for listen addr", listenAddr, p.IP)
				wg.Add(1)
				go np.loopFunc(ctx, wg, client, pool, blacklist, processed, listenAddr, currentDepth, totalDepth, ignoreDepth)
			}
		}

	}

}

func fetchNode(ctx context.Context, ip string) (*Node, error) {
	// ---------- Try Interx (primary) ----------
	var (
		interxInfo   *interxv2.NetInfo
		interxStatus *interxv2.Status
	)

	if err := func() error {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			ni, err := interxhelper.GetNetInfoV2(ctx, ip, types.DEFAULT_INTERX_PORT)
			interxInfo = ni
			return err
		})
		g.Go(func() error {
			st, err := interxhelper.GetInterxStatusV2(ctx, ip, types.DEFAULT_INTERX_PORT)
			interxStatus = st
			return err
		})
		return g.Wait()
	}(); err == nil && interxInfo != nil && interxStatus != nil {
		npeers, perr := strconv.Atoi(interxInfo.NPeers)
		if perr != nil {
			return nil, fmt.Errorf("interx npeers parse: %w", perr)
		}
		node := &Node{
			IP:      ip,
			NCPeers: npeers,
			ID:      interxStatus.NodeInfo.ID,
		}
		node.Peers = extractPeersFromInterx(interxInfo)
		return node, nil
	}

	// ---------- Fallback: Sekai RPC ----------
	var (
		sekaiInfo   *sekaiendpoint.NetInfo
		sekaiStatus *sekaiendpoint.Status
	)
	if err := func() error {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			ni, err := sekaihelper.GetNetInfo(ctx, ip, strconv.Itoa(types.DEFAULT_RPC_PORT))
			sekaiInfo = ni
			return err
		})
		g.Go(func() error {
			st, err := sekaihelper.GetSekaidStatus(ctx, ip, strconv.Itoa(types.DEFAULT_RPC_PORT))
			sekaiStatus = st
			return err
		})
		return g.Wait()
	}(); err != nil || sekaiInfo == nil || sekaiStatus == nil {
		// keep the original combined-failure semantics
		return nil, fmt.Errorf("both interx and sekai failed")
	}

	npeers, perr := strconv.Atoi(sekaiInfo.Result.NPeers)
	if perr != nil {
		return nil, fmt.Errorf("sekai npeers parse: %w", perr)
	}
	node := &Node{
		IP:      ip,
		NCPeers: npeers,
		ID:      sekaiStatus.Result.NodeInfo.ID,
	}
	node.Peers = extractPeersFromSekai(sekaiInfo)
	return node, nil
}

// ----- helpers for peers -----

func extractPeersFromInterx(info *interxv2.NetInfo) []Node {
	if info == nil {
		return nil
	}
	out := make([]Node, 0, len(info.Peers))
	for _, nn := range info.Peers {
		ip, port, err := extractIP(nn.NodeInfo.ListenAddr)
		if err != nil {
			continue
		}
		out = append(out, Node{
			IP: fmt.Sprintf("%s:%s", ip, port),
			ID: nn.NodeInfo.ID,
		})
	}
	return out
}

func extractPeersFromSekai(info *sekaiendpoint.NetInfo) []Node {
	if info == nil {
		return nil
	}
	out := make([]Node, 0, len(info.Result.Peers))
	for _, nn := range info.Result.Peers {
		ip, port, err := extractIP(nn.NodeInfo.ListenAddr)
		if err != nil {
			continue
		}
		out = append(out, Node{
			IP: fmt.Sprintf("%s:%s", ip, port),
			ID: nn.NodeInfo.ID,
		})
	}
	return out
}
