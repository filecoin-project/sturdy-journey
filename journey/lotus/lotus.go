package lotus

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/filecoin-project/sturdy-journey/internal/circleci"
	"github.com/filecoin-project/sturdy-journey/internal/config"
	"github.com/filecoin-project/sturdy-journey/internal/secretloader"
	"github.com/filecoin-project/sturdy-journey/journey"
	"github.com/filecoin-project/sturdy-journey/registry"

	"github.com/google/go-github/v37/github"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sturdy-journey/journey/lotus")

const (
	JourneyName = "lotus"
)

func init() {
	registry.Register(JourneyName, JourneyConstructor, DefaultConfig())
}

func DefaultConfig() *Config {
	return &Config{
		PipelineBranch:  "master",
		CircleTokenPath: "",
		CircleProject:   "filecoin-project/lotus-infra",
		CircleBaseURL:   &config.URL{Host: "circleci.com", Scheme: "https", Path: "/api/v2/"},
	}
}

func JourneyConstructor(cfg config.CommonJourney) (http.Handler, error) {
	j, err := NewJourney(cfg)
	if err != nil {
		return nil, err
	}

	gej := journey.NewGithubEventJourney(cfg, j)

	return gej, nil
}

type Config struct {
	// PipelineBranch git branch circle api requests will be made against
	PipelineBranch string

	// CircleTokenPath file system path where the circleci token secret is located
	CircleTokenPath string

	// CircleBaseURL URL prefix to circleci requests, mostly used to testing
	CircleBaseURL *config.URL

	// CircleProject project-slug used to construct api requests
	CircleProject string
}

type Journey struct {
	ctx            context.Context
	circleToken    secretloader.SecretLoader
	pipelineBranch string
	circleBaseURL  *url.URL
	circleProject  string
}

var _ journey.GithubEventHandler = (*Journey)(nil)

func NewJourney(ccfg config.CommonJourney) (*Journey, error) {
	icfg, err := config.FromFile(ccfg.ConfigPath, &Config{})
	if err != nil {
		return nil, err
	}

	cfg := icfg.(*Config)

	u := url.URL(*cfg.CircleBaseURL)
	return &Journey{
		circleToken:    secretloader.NewSecretLoader(cfg.CircleTokenPath, time.Second*15),
		circleBaseURL:  &u,
		circleProject:  cfg.CircleProject,
		pipelineBranch: cfg.PipelineBranch,
	}, nil
}

func (j *Journey) HandleEvent(event interface{}) error {
	switch event := event.(type) {
	case *github.ReleaseEvent:
		return j.processReleaseEvent(event)
	default:
		return journey.ErrUnhandledEvent
	}
}

func (j *Journey) processReleaseEvent(event *github.ReleaseEvent) error {
	log.Debugw("processing release event", "github_release_name", event.Release.Name, "github_tag_name", event.Release.TagName, "github_prerelease", event.Release.Prerelease, "action", *event.Action)
	// https://docs.github.com/en/developers/webhooks-and-events/webhooks/webhook-events-and-payloads#release
	if !(*event.Action == "prerelease" || *event.Action == "released") {
		return nil
	}

	_, circleToken, err := j.circleToken.Get()
	if err != nil {
		log.Warnw("failed to load circle token", "err", err)
		return err
	}

	c := &circleci.Client{BaseURL: j.circleBaseURL, Token: string(circleToken), Project: j.circleProject}

	parameters := map[string]interface{}{
		"api_workflow_requested": "api-lotus-release-automation",
		"release":                event.Release.TagName,
	}

	resp, err := c.CreatePipeline(j.pipelineBranch, parameters)
	if err != nil {
		return err
	}

	log.Infow("pipeline created", "circleci_pipeline_id", resp.ID, "circleci_pipeline_number", resp.Number, "github_release_name", event.Release.Name, "github_tag_name", event.Release.TagName, "github_prerelease", event.Release.Prerelease)

	return nil
}
