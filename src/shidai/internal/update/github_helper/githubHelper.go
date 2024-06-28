package githubhelper

import "github.com/kiracore/sekin/src/shidai/internal/types"

type GithubTestHelper struct{}

func (GithubTestHelper) GetLatestSekinVersion() (*types.SekinPackagesVersion, error) {
	return &types.SekinPackagesVersion{Sekai: "v0.3.45", Interx: "v0.4.49", Shidai: "v0.9.0"}, nil
}
