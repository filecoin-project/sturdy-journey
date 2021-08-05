package registry

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/filecoin-project/sturdy-journey/internal/config"

	"golang.org/x/xerrors"
)

var (
	journeys = NewRegistry()
)

func Register(name string, constructor NewJourneyFunc, defaultConfig interface{}) {
	journeys.Register(name, constructor, defaultConfig)
}

func Get(name string) (*Journey, error) {
	return journeys.Get(name)
}

func Registered() []string {
	return journeys.Registered()
}

type NewJourneyFunc func(config.CommonJourney) (http.Handler, error)

type Journey struct {
	Constructor   NewJourneyFunc
	DefaultConfig interface{}
}

type Registry struct {
	Journeys map[string]*Journey
}

func NewRegistry() *Registry {
	return &Registry{
		Journeys: make(map[string]*Journey),
	}
}

func (r *Registry) Register(name string, constructor NewJourneyFunc, defaultConfig interface{}) {
	if _, exists := r.Journeys[name]; exists {
		panic("already exists")
	}

	r.Journeys[name] = &Journey{
		Constructor:   constructor,
		DefaultConfig: defaultConfig,
	}
}

func (r *Registry) Get(name string) (*Journey, error) {
	if _, exists := r.Journeys[name]; exists {
		return r.Journeys[name], nil
	}

	return nil, xerrors.Errorf("journey not found: %s", name)
}

func (r *Registry) Registered() []string {
	names := make([]string, len(r.Journeys))
	for name := range r.Journeys {
		fmt.Printf("%s\n", name)
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
