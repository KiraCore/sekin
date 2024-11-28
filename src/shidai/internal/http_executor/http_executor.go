package httpexecutor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/utils"
	"go.uber.org/zap"
)

type CommandRequest struct {
	Command string      `json:"command"`
	Args    interface{} `json:"args"`
}

var log *zap.Logger = logger.GetLogger()

// ExecuteCallerCommand executes a command for iCaller and sCaller
func ExecuteCallerCommand(address, port, method string, commandRequest CommandRequest) ([]byte, error) {
	log.Debug("Starting ExecuteCallerCommand", zap.String("address", address), zap.String("port", port), zap.String("method", method))

	p, err := strconv.Atoi(port)
	if err != nil {
		log.Error("Invalid port conversion", zap.String("port", port), zap.Error(err))
		return nil, fmt.Errorf("port conversion error for value: <%v>", port)
	}
	if !utils.ValidatePort(p) {
		log.Error("Port validation failed", zap.String("port", port), zap.Int("parsedPort", p))
		return nil, fmt.Errorf("port validation failed for value: <%v>", port)
	}

	jsonData, err := json.Marshal(commandRequest)
	if err != nil {
		log.Error("Error marshaling JSON", zap.Error(err))
		return nil, err
	}
	log.Debug("JSON data marshaled successfully")

	req, err := http.NewRequest(method, fmt.Sprintf("http://%v:%v/api/execute", address, port), bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Error creating HTTP request", zap.String("method", method), zap.String("URL", fmt.Sprintf("http://%v:%v/api/execute", address, port)), zap.Error(err))
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	log.Debug("HTTP request created successfully", zap.String("url", req.URL.String()))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error making HTTP request", zap.String("URL", req.URL.String()), zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()
	log.Debug("HTTP request executed", zap.Int("status_code", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading response body", zap.Error(err))
		return nil, err
	}
	log.Debug("Response body read successfully", zap.ByteString("response", body))

	return body, nil
}

func DoHttpQuery(ctx context.Context, client *http.Client, url, method string) ([]byte, error) {
	log.Debug("Starting DoHttpQuery", zap.String("url", url), zap.String("method", method))

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		log.Error("Failed to create HTTP request with context", zap.Error(err))
		return nil, err
	}
	log.Debug("HTTP request with context created successfully", zap.String("url", req.URL.String()))

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send HTTP request", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()
	log.Debug("HTTP request sent successfully", zap.Int("status_code", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to read response body", zap.Error(err))
		return nil, err
	}
	log.Debug("Response body read successfully", zap.ByteString("response", body))

	return body, nil
}
