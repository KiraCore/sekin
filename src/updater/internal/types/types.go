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

type UpgradePlan struct {
	Plan interface{} `json:"plan"`
}
