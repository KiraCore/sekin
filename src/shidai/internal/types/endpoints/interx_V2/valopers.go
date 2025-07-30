package interxv2

type Valopers struct {
	Validators []Validator `json:"validators"`
	Actors     []string    `json:"actors"`
	Pagination any         `json:"pagination"`
}

type Validator struct {
	Top                   string     `json:"top"`
	Address               string     `json:"address"`
	ValKey                string     `json:"valkey"`
	PubKey                string     `json:"pubkey"`
	Proposer              string     `json:"proposer"`
	Moniker               string     `json:"moniker"`
	Status                string     `json:"status"`
	Rank                  string     `json:"rank"`
	Streak                string     `json:"streak"`
	Mischance             string     `json:"mischance"`
	MischanceConfidence   string     `json:"mischance_confidence"`
	Identity              []Identity `json:"identity"`
	StartHeight           string     `json:"start_height"`
	InactiveUntil         string     `json:"inactive_until"`
	LastPresentBlock      string     `json:"last_present_block"`
	MissedBlocksCounter   string     `json:"missed_blocks_counter"`
	ProducedBlocksCounter string     `json:"produced_blocks_counter"`
}

type Identity struct {
	ID        string        `json:"id"`
	Key       string        `json:"key"`
	Value     string        `json:"value"`
	Date      any           `json:"date"`      // Empty object `{}` — can be changed to `map[string]any` if needed
	Verifiers []interface{} `json:"verifiers"` // Empty array `[]` — can be changed to a specific type if known
}
