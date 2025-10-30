package types

type Gateway interface {
	Handle(data []byte) (interface{}, error)
	Close()
}

type GatewayFactory interface {
	CreateGateway(gatewayType string) (Gateway, error)
}

type PathMappings struct {
	Pattern     string
	Replacement string
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   struct {
		Code    float64 `json:"code"`
		Message string  `json:"message"`
		Data    string  `json:"data"`
	} `json:"error,omitempty"`
}
