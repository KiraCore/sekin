package types

type SaiData struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Payload interface{} `json:"payload"`
}

type SaiRequest struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
}
