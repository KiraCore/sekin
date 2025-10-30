package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/cors"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interax-proxy/logger"
	"github.com/saiset-co/sai-interax-proxy/types"
	"github.com/saiset-co/sai-service/service"
)

type InternalService struct {
	Context  *service.Context
	ProxyUrl string
}

func (is *InternalService) Init() {
	is.ProxyUrl = cast.ToString(is.Context.GetConfig("manager.url", ""))
}

func (is *InternalService) Process() {
	is.StartHttpProxy()
}

func (is *InternalService) StartHttpProxy() {
	port := is.Context.GetConfig("common.http.port", 80).(int)
	handler := http.HandlerFunc(is.handleHttpConnections)
	corsHandler := cors.AllowAll().Handler(handler)

	http.Handle("/", corsHandler)
	logger.Logger.Info("Starting HTTP server on port", zap.Int("Port", port))

	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		logger.Logger.Error("StartHttpProxy", zap.Error(err))
	}
}

func (is *InternalService) handleHttpConnections(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Debug("handleHttpConnections", zap.Any("method", r.Method), zap.Any("path", r.URL.Path))

	var requestData interface{}

	if r.Method == "GET" {
		queryParams := r.URL.Query()
		paramMap := make(map[string]string)
		for key, values := range queryParams {
			if len(values) > 0 {
				paramMap[key] = values[0]
			}
		}
		requestData = paramMap
	} else if r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			logger.Logger.Error("handleHttpConnections", zap.Error(err))
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		if r.Header.Get("Content-Type") == "application/json" {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(body, &jsonData); err == nil {
				requestData = jsonData
			} else {
				requestData = string(body)
			}
		} else {
			requestData = string(body)
		}
	}

	method := determineMethod(r.URL.Path)
	path := strings.Replace(r.URL.Path, "/api", "", -1)

	if method == "ethereum" {
		path = strings.Replace(path, "/"+method, "", -1)
	}

	request := types.SaiRequest{
		Method: method,
		Data: types.SaiData{
			Method:  r.Method,
			Path:    path,
			Payload: requestData,
		},
	}

	response, err := is.SendProxyRequest(request)
	if err != nil {
		logger.Logger.Error("handleHttpConnections", zap.Error(err))
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (is *InternalService) SendProxyRequest(r types.SaiRequest) ([]byte, error) {
	reqData, err := json.Marshal(r)
	if err != nil {
		logger.Logger.Error("SendProxyRequest", zap.Error(err))
		return nil, err
	}

	resp, err := http.Post(is.ProxyUrl, "application/json", io.NopCloser(io.Reader(bytes.NewBuffer(reqData))))
	if err != nil {
		logger.Logger.Error("SendProxyRequest", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func determineMethod(path string) string {
	path = strings.Replace(path, "/api", "", -1)

	if strings.HasPrefix(path, "/rosetta/") {
		return "rosetta"
	}

	if strings.HasPrefix(path, "/bitcoin/") {
		return "bitcoin"
	}

	if strings.HasPrefix(path, "/ethereum/") {
		return "ethereum"
	}

	return "cosmos"
}
