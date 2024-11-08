package types

type SekinPackagesVersion struct {
	Sekai  string
	Interx string
	Shidai string
}

type (
	AppInfo struct {
		Version string `json:"version"`
		Infra   bool   `json:"infra"`
	}

	StatusResponse struct {
		Sekai  AppInfo `json:"sekai"`
		Interx AppInfo `json:"interx"`
		Shidai AppInfo `json:"shidai"`
		Syslog AppInfo `json:"syslog-ng"`
	}
)

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

const SEKIN_LATEST_COMPOSE_URL = "https://raw.githubusercontent.com/KiraCore/sekin/main/compose.yml"
const SEKAI_CONTAINER_ID = "sekin-sekai-1"

const UPDATE_PLAN string = "./upgrade_plan.json"

const SEKIN_HOME string = "/home/km/sekin"
const SEKAI_HOME string = SEKIN_HOME + CONTAINERIZED_SEKAI_HOME

// TODO: FOR TESTING PURPOSES, DELETE AFTER
// const SEKIN_HOME string = "/home/d/sekin"
// const SEKAI_HOME string = SEKIN_HOME + CONTAINERIZED_SEKAI_HOME

const CONTAINERIZED_SEKAI_HOME string = "/sekai"                               //from sekai container prospective
const CONTAINERIZED_SEKAI_CONFIG string = CONTAINERIZED_SEKAI_HOME + "/config" //from sekai container prospective

const SEKAI_IMAGE_NAME string = "ghcr.io/kiracore/sekin/sekai"

const SEKAI_CONTAINER_ADDRESS string = "sekai.local"

const INTERX_CONTAINER_ID = "sekin-interx-1"
const CONTAINERIZED_INTERX_HOME string = "/interx"
