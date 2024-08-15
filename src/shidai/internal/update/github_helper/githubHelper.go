package githubhelper

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
)

var log = logger.GetLogger() // Initialize the logger instance at the package level

type GithubTestHelper struct{}

func (GithubTestHelper) GetLatestSekinVersion() (*types.SekinPackagesVersion, error) {
	return &types.SekinPackagesVersion{Sekai: "v0.3.45", Interx: "v0.4.49", Shidai: "v0.9.0"}, nil
}

type ComposeFileParser struct{}

func (c ComposeFileParser) GetLatestSekinVersion() (*types.SekinPackagesVersion, error) {
	ctx := context.Background()
	sekinComose, err := httpexecutor.DoHttpQuery(ctx, &http.Client{}, types.SEKIN_LATEST_COMPOSE_URL, "GET")
	if err != nil {
		return nil, err
	}
	project, err := GetDockerComposeProject(sekinComose)
	if err != nil {
		return nil, fmt.Errorf("error when getting compose project: %w", err)
	}

	// sekai:ghcr.io/kiracore/sekin/sekai:v0.3.45
	// interx:ghcr.io/kiracore/sekin/interx:v0.4.49
	// shidai:ghcr.io/kiracore/sekin/shidai:v0.9.0

	regex := regexp.MustCompile(`:v([0-9]+\.[0-9]+\.[0-9]+)`)
	const (
		sekaiServiceName  = "sekai"
		interxServiceName = "interx"
		shidaiServiceName = "shidai"
	)

	response := &types.SekinPackagesVersion{}
	for _, p := range project.Services {
		log.Debug(fmt.Sprintf("%v:%v", p.Name, p.Image))
		switch p.Name {
		case sekaiServiceName:
			match := regex.FindStringSubmatch(p.Image)
			if len(match) > 1 {
				response.Sekai = match[1]
			} else {
				return nil, fmt.Errorf("unable to parse Sekai image: <%v> not matching regex rule", p.Image)
			}
		case interxServiceName:
			match := regex.FindStringSubmatch(p.Image)
			if len(match) > 1 {
				response.Interx = match[1]
			} else {
				return nil, fmt.Errorf("unable to parse Interx image: <%v> not matching regex rule", p.Image)
			}
		case shidaiServiceName:
			match := regex.FindStringSubmatch(p.Image)
			if len(match) > 1 {
				response.Shidai = match[1]
			} else {
				return nil, fmt.Errorf("unable to parse Shidai image: <%v> not matching regex rule", p.Image)
			}
		}
	}
	return response, nil
}

func GetDockerComposeProject(dockerComposeFile []byte) (*composeTypes.Project, error) {
	project, err := loader.Load(composeTypes.ConfigDetails{
		ConfigFiles: []composeTypes.ConfigFile{{Content: dockerComposeFile}},
	}, func(o *loader.Options) {
		name, projectNameImperativel := o.GetProjectName() // Set a default project name if none is provided
		if name == "" {
			o.SetProjectName("default_project", projectNameImperativel)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("error loading compose file: %w", err)
	}
	return project, nil
}