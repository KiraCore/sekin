package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiracore/sekin/src/shidai/internal/docker"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

type (
	DashboardPointer struct {
		Data *Dashboard
		mu   sync.RWMutex
	}

	Dashboard struct {
		RoleIDs []string `json:"role_ids"`

		Date                string `json:"date"`
		ValidatorStatus     string `json:"val_status"`
		Blocks              string `json:"blocks"`
		Top                 string `json:"top"`
		Streak              string `json:"streak"`
		Mischance           string `json:"mischance"`
		MischanceConfidence string `json:"mischance_confidence"`
		StartHeight         string `json:"start_height"`
		LastProducedBlock   string `json:"last_present_block"`
		ProducedBlocks      string `json:"produced_blocks_counter"`
		Moniker             string `json:"moniker"`
		ValidatorAddress    string `json:"address"`
		ChainID             string `json:"chain_id"`
		NodeID              string `json:"node_id"`
		GenesisChecksum     string `json:"genesis_checksum"`

		ActiveValidators   int `json:"active_validators"`
		PausedValidators   int `json:"paused_validators"`
		InactiveValidators int `json:"inactive_validators"`
		JailedValidators   int `json:"jailed_validatore"`
		WaitingValidators  int `json:"waiting_validators"`

		SeatClaimAvailable bool `json:"seat_claim_available"`
		Waiting            bool `json:"seat_claim_pending"`
		CatchingUp         bool `json:"catching_up"`
	}
)

var dashboardPointer = NewDashboardPointer()

func NewDashboardPointer() *DashboardPointer {
	return &DashboardPointer{
		Data: &Dashboard{
			Date:                "Unknown",
			ValidatorStatus:     "Unknown",
			Blocks:              "Unknown",
			Top:                 "Unknown",
			Streak:              "Unknown",
			Mischance:           "Unknown",
			MischanceConfidence: "Unknown",
			StartHeight:         "Unknown",
			LastProducedBlock:   "Unknown",
			ProducedBlocks:      "Unknown",
			Moniker:             "Unknown",
			ValidatorAddress:    "Unknown",
			ChainID:             "Unknown",
			NodeID:              "Unknown",
			GenesisChecksum:     "Unknown",
			RoleIDs:             []string{"Unknown"},
			ActiveValidators:    0,
			PausedValidators:    0,
			InactiveValidators:  0,
			JailedValidators:    0,
			WaitingValidators:   0,
			SeatClaimAvailable:  false,
			Waiting:             false,
			CatchingUp:          false,
		},
	}
}

func getDashboardHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		dashboardPointer.mu.RLock()
		clone := dashboardPointer.Data.DeepCopy() // Use the read lock to allow concurrent reads
		defer dashboardPointer.mu.RUnlock()

		// Instead of reading from file, directly marshal the in-memory dashboard
		dashboardJSON, err := json.Marshal(clone)
		if err != nil {
			log.Error("Failed to marshal dashboard data", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to marshal dashboard data"})
			return
		}

		c.Data(http.StatusOK, "application/json", dashboardJSON)
	}
}

func backgroundUpdate() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Debug("Starting background update")

		if err := updateDashboard(); err != nil {
			log.Error("Failed to update dashboard", zap.Error(err))
			continue
		}

		if err := saveDashboardToFile(); err != nil {
			log.Error("Failed to save dashboard to file", zap.String("file", types.DashboardPath), zap.Error(err))
		}
	}
}

func saveDashboardToFile() error {
	dashboardPointer.mu.Lock()
	defer dashboardPointer.mu.Unlock()

	writer, err := os.OpenFile(types.DashboardPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, types.FilePermRW)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer writer.Close()

	bufWriter := bufio.NewWriter(writer)
	if err := json.NewEncoder(bufWriter).Encode(dashboardPointer.Data); err != nil {
		return fmt.Errorf("failed to marshal and write dashboard: %w", err)
	}
	if err := bufWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush write buffer: %w", err)
	}

	log.Info("Dashboard pointer updated and saved to file", zap.String("file", types.DashboardPath))
	return nil
}

func updateDashboard() error {
	ctx := context.Background()
	cm, err := docker.NewContainerManager()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	dashboardData := dashboardPointer.GetDashboardData()
	dashboardUpdates := make(chan *Dashboard, 10) // Buffer size based on expected concurrency
	done := make(chan error, 10)

	wg.Add(6) //Increase with qty of fetches

	go func() {
		defer wg.Done()
		fetchDateNow(dashboardUpdates, done)
	}()

	go func() {
		defer wg.Done()
		fetchAccAddressFromSekaidBin(ctx, cm, types.SEKAI_CONTAINER_ID, dashboardUpdates, done)
	}()

	go func() {
		defer wg.Done()
		fetchRoleIDsFromSekaidBin(ctx, cm, types.SEKAI_CONTAINER_ID, dashboardData.ValidatorAddress, dashboardUpdates, done)
	}()

	go func() {
		defer wg.Done()
		fetchValidatorDataFromValopersAPI(ctx, dashboardData.ValidatorAddress, dashboardUpdates, done)
	}()

	go func() {
		defer wg.Done()
		fetchValidatorsStatus(ctx, dashboardData.ValidatorAddress, dashboardUpdates, done)
	}()

	go func() {
		defer wg.Done()
		fetchNodeStatus(ctx, dashboardUpdates, done)
	}()

	var errors []error
	for i := 0; i < 6; i++ { // Adjust this based on the number of goroutines
		if err := <-done; err != nil {
			log.Warn("Failed to fetch some data", zap.Error(err))
			errors = append(errors, err)
		}
	}

	go func() {
		wg.Wait()
		close(dashboardUpdates)
		close(done)
	}()
	if len(errors) > 0 {
		log.Warn("Fetching ecnounted errors", zap.String("times", strconv.Itoa(len(errors))))
		for _, err := range errors {
			log.Error("Detail[FETCH]: ", zap.Error(err))
		}
	}

	var (
		catchingUpFinal         bool = false // Finale state if node is catching up or not after all updates
		waitingFinal            bool = false
		activeValidatorsFinal   int  = 0
		pausedValidatorsFinal   int  = 0
		inactiveValidatorsFinal int  = 0
		jailedValidatorsFinal   int  = 0
		waitingValidatorsFinal  int  = 0
	)
	for update := range dashboardUpdates {
		applyUpdate(dashboardPointer, update)
		if update.CatchingUp {
			catchingUpFinal = true
		}
		if update.Waiting {
			waitingFinal = true
		}
		if update.ActiveValidators != 0 {
			activeValidatorsFinal = update.ActiveValidators
		}
		if update.PausedValidators != 0 {
			pausedValidatorsFinal = update.PausedValidators
		}
		if update.InactiveValidators != 0 {
			inactiveValidatorsFinal = update.InactiveValidators
		}
		if update.JailedValidators != 0 {
			jailedValidatorsFinal = update.JailedValidators
		}
		if update.WaitingValidators != 0 {
			waitingValidatorsFinal = update.WaitingValidators
		}
	}

	dashboardPointer.mu.Lock()
	dashboardPointer.Data.CatchingUp = catchingUpFinal
	dashboardPointer.Data.Waiting = waitingFinal
	dashboardPointer.Data.ActiveValidators = activeValidatorsFinal
	dashboardPointer.Data.PausedValidators = pausedValidatorsFinal
	dashboardPointer.Data.InactiveValidators = inactiveValidatorsFinal
	dashboardPointer.Data.JailedValidators = jailedValidatorsFinal
	dashboardPointer.Data.WaitingValidators = waitingValidatorsFinal
	dashboardPointer.Data.SeatClaimAvailable = utils.ContainsValue(dashboardPointer.Data.RoleIDs, "2") && !dashboardPointer.Data.CatchingUp && dashboardPointer.Data.Waiting
	dashboardPointer.mu.Unlock()

	return nil
}

// applyUpdate safely applies updates to the shared dashboardPointer.
func applyUpdate(pointer *DashboardPointer, update *Dashboard) {
	// Lock the pointer for writing.
	pointer.mu.Lock()
	defer pointer.mu.Unlock()

	// Check and update each field, avoiding overwrite with "Unknown" unless intended
	if update.Date != "Unknown" && update.Date != "" {
		pointer.Data.Date = update.Date
	}
	if update.ValidatorStatus != "Unknown" && update.ValidatorStatus != "" {
		pointer.Data.ValidatorStatus = update.ValidatorStatus
	}
	if update.Blocks != "Unknown" && update.Blocks != "" {
		pointer.Data.Blocks = update.Blocks
	}
	if update.Top != "Unknown" && update.Top != "" {
		pointer.Data.Top = update.Top
	}
	if update.Streak != "Unknown" && update.Streak != "" {
		pointer.Data.Streak = update.Streak
	}
	if update.Mischance != "Unknown" && update.Mischance != "" {
		pointer.Data.Mischance = update.Mischance
	}
	if update.MischanceConfidence != "Unknown" && update.MischanceConfidence != "" {
		pointer.Data.MischanceConfidence = update.MischanceConfidence
	}
	if update.StartHeight != "Unknown" && update.StartHeight != "" {
		pointer.Data.StartHeight = update.StartHeight
	}
	if update.LastProducedBlock != "Unknown" && update.LastProducedBlock != "" {
		pointer.Data.LastProducedBlock = update.LastProducedBlock
	}
	if update.ProducedBlocks != "Unknown" && update.ProducedBlocks != "" {
		pointer.Data.ProducedBlocks = update.ProducedBlocks
	}
	if update.Moniker != "Unknown" && update.Moniker != "" {
		pointer.Data.Moniker = update.Moniker
	}
	if update.ValidatorAddress != "Unknown" && update.ValidatorAddress != "" {
		pointer.Data.ValidatorAddress = update.ValidatorAddress
	}
	if update.ChainID != "Unknown" && update.ChainID != "" {
		pointer.Data.ChainID = update.ChainID
	}
	if update.NodeID != "Unknown" && update.NodeID != "" {
		pointer.Data.NodeID = update.NodeID
	}
	if len(update.RoleIDs) > 0 && !utils.ContainsValue(update.RoleIDs, "Unknown") {
		pointer.Data.RoleIDs = update.RoleIDs
	}
	if update.GenesisChecksum != "Unknown" && update.GenesisChecksum != "" {
		pointer.Data.GenesisChecksum = update.GenesisChecksum
	}
}

func fetchRoleIDsFromSekaidBin(ctx context.Context, cm *docker.ContainerManager, containerID, adr string, updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching roleID from sekai container")
	if containerID == "" || adr == "" {
		done <- fmt.Errorf("containerID or address can't be empty")
		return
	}
	command := []string{"/sekaid", "q", "customgov", "roles", adr, "--node", types.SEKAI_RPC_LADDR, "--output", "json"}
	ouput, err := cm.ExecInContainer(ctx, containerID, command)
	if err != nil {
		done <- fmt.Errorf("failed to execute command in container %s: %w", containerID, err)
		return
	}
	var result struct {
		RoleIDs []string `json:"roleids"`
	}
	if err = json.Unmarshal(ouput, &result); err != nil {
		done <- fmt.Errorf("failed to parse JSON output: %w", err)
		return

	}
	updates <- &Dashboard{RoleIDs: result.RoleIDs}
}

func fetchAccAddressFromSekaidBin(ctx context.Context, cm *docker.ContainerManager, containerID string, updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching address from sekai container")
	if containerID == "" {
		done <- fmt.Errorf("containerID can't be empty")
		return
	}
	command := []string{"/sekaid", "keys", "show", "validator", "--home", "/sekai", "--keyring-backend", "test", "--output", "json"}
	output, err := cm.ExecInContainer(ctx, containerID, command)
	if err != nil {
		done <- fmt.Errorf("failed to execute command in container %s: %w", containerID, err)
		return
	}

	var result struct {
		Address string `json:"address"`
	}
	if err = json.Unmarshal(output, &result); err != nil {
		done <- fmt.Errorf("faile d to parse JSON output: %w", err)
		return
	}
	updates <- &Dashboard{ValidatorAddress: result.Address}
}

func fetchValidatorDataFromValopersAPI(ctx context.Context, address string, updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching data from valopers endpoint")
	if address == "" {
		done <- fmt.Errorf("address can't be empty")
		return
	}

	// url := fmt.Sprintf("http://interx.local:11000/api/valopers?address=%s", address)
	url := fmt.Sprintf("http://%v:%v/%v?address=%s",
		types.INTERX_CONTAINER_ADDRESS,
		types.DEFAULT_INTERX_PORT,
		types.ENDPOINT_INTERX_VALOPERS,
		address,
	)

	resp, err := http.Get(url)
	if err != nil {
		done <- fmt.Errorf("failed to make HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		done <- fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		return
	}

	var apiResponse struct {
		Validators []struct {
			Top                   string `json:"top"`
			Moniker               string `json:"moniker"`
			Status                string `json:"status"`
			Streak                string `json:"streak"`
			Mischance             string `json:"mischance"`
			MischanceConfidence   string `json:"mischance_confidence"`
			StartHeight           string `json:"start_height"`
			LastPresentBlock      string `json:"last_present_block"`
			ProducedBlocksCounter string `json:"produced_blocks_counter"`
		} `json:"validators"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		done <- fmt.Errorf("failed to decode JSON response: %w", err)
		return
	}

	if len(apiResponse.Validators) == 0 {
		done <- fmt.Errorf("no validators found in response")
		return
	}

	validator := apiResponse.Validators[0]
	update := &Dashboard{
		Top:                 validator.Top,
		Moniker:             validator.Moniker,
		ValidatorStatus:     validator.Status,
		Streak:              validator.Streak,
		Mischance:           validator.Mischance,
		MischanceConfidence: validator.MischanceConfidence,
		StartHeight:         validator.StartHeight,
		LastProducedBlock:   validator.LastPresentBlock,
		ProducedBlocks:      validator.ProducedBlocksCounter,
	}

	updates <- update
}
func fetchValidatorsStatus(ctx context.Context, address string, updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching validators status block")
	if address == "" {
		done <- fmt.Errorf("address can't be empty")
		return
	}

	// url := "http://interx.local:11000/api/valopers?all=true"
	url := fmt.Sprintf("http://%v:%v/%v?all=true",
		types.INTERX_CONTAINER_ADDRESS,
		types.DEFAULT_INTERX_PORT,
		types.ENDPOINT_INTERX_VALOPERS,
	)

	resp, err := http.Get(url)
	if err != nil {
		done <- fmt.Errorf("failed to make HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		done <- fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		return
	}

	var apiResponse struct {
		Status struct {
			ActiveValidators   int `json:"active_validators"`
			PausedValidators   int `json:"paused_validators"`
			InactiveValidators int `json:"inactive_validators"`
			JailedValidators   int `json:"jailed_validators"`
			TotalValidators    int `json:"total_validators"`
			WaitingValidators  int `json:"waiting_validators"`
		} `json:"status"`
		Waiting []string `json:"waiting"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		done <- fmt.Errorf("failed to decode JSON response: %w", err)
		return
	}

	// Update Dashboard structure
	update := &Dashboard{
		ActiveValidators:   apiResponse.Status.ActiveValidators,
		PausedValidators:   apiResponse.Status.PausedValidators,
		InactiveValidators: apiResponse.Status.InactiveValidators,
		JailedValidators:   apiResponse.Status.JailedValidators,
		WaitingValidators:  apiResponse.Status.WaitingValidators,
	}

	// Check if the provided address is in the waiting list
	for _, waitingAddress := range apiResponse.Waiting {
		if waitingAddress == address {
			update.Waiting = true
			break
		}
	}

	updates <- update
}

func fetchNodeStatus(ctx context.Context, updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching node status from interx")
	// url := "http://interx.local:11000/api/status"
	url := fmt.Sprintf("http://%v:%v/%v", types.INTERX_CONTAINER_ADDRESS, types.DEFAULT_INTERX_PORT, types.ENDPOINT_INTERX_STATUS)
	resp, err := http.Get(url)
	if err != nil {
		done <- fmt.Errorf("failed to make HTTP request: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		done <- fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		return
	}

	var apiResponse struct {
		ID         string `json:"id"`
		InterxInfo struct {
			ChainID         string `json:"chain_id"`
			GenesisChecksum string `json:"genesis_checksum"`
		} `json:"interx_info"`
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
			CatchingUp        bool   `json:"catching_up"`
		} `json:"sync_info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		done <- fmt.Errorf("failed to decode JSON response: %w", err)
		return
	}
	log.Debug("Parsed CatchingUp status", zap.Bool("CatchingUp: ", apiResponse.SyncInfo.CatchingUp))
	// Create an update based on the fetched data
	update := &Dashboard{
		NodeID:          apiResponse.ID,
		ChainID:         apiResponse.InterxInfo.ChainID,
		Blocks:          apiResponse.SyncInfo.LatestBlockHeight,
		CatchingUp:      apiResponse.SyncInfo.CatchingUp,
		GenesisChecksum: apiResponse.InterxInfo.GenesisChecksum,
	}

	log.Info("Sending update to dashboardUpdates")
	updates <- update
}

//	func fetchDummy(updates chan<- *Dashboard, done chan<- error) {
//		defer func() { done <- nil }()
//		done <- fmt.Errorf("error")
//		updates <- &Dashboard{Key: Value}
//	}
func fetchDateNow(updates chan<- *Dashboard, done chan<- error) {
	defer func() { done <- nil }()
	log.Debug("Fetching date ")
	now := time.Now()
	formattedDate := now.Format("2006-01-02 15:04:05")
	updates <- &Dashboard{Date: formattedDate}

}

// DeepCopy creates a deep copy of the Dashboard.
func (d *Dashboard) DeepCopy() *Dashboard {
	clone := *d // Copy all primitive fields

	// Manually copy slices to ensure they are independent
	if d.RoleIDs != nil {
		clone.RoleIDs = make([]string, len(d.RoleIDs))
		copy(clone.RoleIDs, d.RoleIDs)
	}

	return &clone
}

// GetDashboardData retrieves a deep copy of the Dashboard data safely.
func (dp *DashboardPointer) GetDashboardData() *Dashboard {
	dp.mu.RLock()         // Acquire a read lock
	defer dp.mu.RUnlock() // Ensure the lock is released when the function exits

	return dp.Data.DeepCopy()
}
