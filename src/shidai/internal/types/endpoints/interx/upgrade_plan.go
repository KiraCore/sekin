package interx

const ENDPOINT_NEXT_PLAN string = "api/kira/upgrade/next_plan"       //returns UpgradePlan struct
const ENDPOINT_CURRENT_PLAN string = "api/kira/upgrade/current_plan" //returns UpgradePlan struct

type UpgradePlanResource struct {
	Checksum string `json:"checksum"`
	ID       string `json:"id"`
	URL      string `json:"url"`
	Version  string `json:"version"`
}

type UpgradePlan struct {
	InstateUpgrade            bool                  `json:"instateUpgrade"`
	MaxEnrolmentDuration      string                `json:"maxEnrolmentDuration"`
	Name                      string                `json:"name"`
	NewChainID                string                `json:"newChainId"`
	OldChainID                string                `json:"oldChainId"`
	ProcessedNoVoteValidators bool                  `json:"processedNoVoteValidators"`
	ProposalID                string                `json:"proposalID"`
	RebootRequired            bool                  `json:"rebootRequired"`
	Resources                 []UpgradePlanResource `json:"resources"`
	RollbackChecksum          string                `json:"rollbackChecksum"`
	SkipHandler               bool                  `json:"skipHandler"`
	UpgradeTime               string                `json:"upgradeTime"`
}

type PlanData struct {
	Plan UpgradePlan `json:"plan"`
}
