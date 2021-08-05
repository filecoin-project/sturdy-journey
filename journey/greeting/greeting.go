package greeting

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/sturdy-journey/internal/config"
	"github.com/filecoin-project/sturdy-journey/registry"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sturdy-journey/journey/greeting")

const (
	JourneyName = "greeting"
)

func init() {
	registry.Register(JourneyName, JourneyConstructor, DefaultConfig())
}

func DefaultConfig() *Config {
	return &Config{
		Response: "Save Travels!",
	}
}

func JourneyConstructor(cfg config.CommonJourney) (http.Handler, error) {
	j, err := NewJourney(cfg)
	if err != nil {
		return nil, err
	}

	return j, nil
}

type Config struct {
	// Response string returned to user on request
	Response string
}

type Journey struct {
	ctx      context.Context
	response string
}

func LoadConfig(configPath string) (*Config, error) {
	icfg, err := config.FromFile(configPath, DefaultConfig())
	if err != nil {
		return nil, err
	}

	cfg := icfg.(*Config)

	return cfg, nil
}

func NewJourney(ccfg config.CommonJourney) (*Journey, error) {
	cfg, err := LoadConfig(ccfg.ConfigPath)
	if err != nil {
		return nil, err
	}

	return &Journey{
		response: cfg.Response,
	}, nil
}

func (j *Journey) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infow("new request")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", j.response)
}
