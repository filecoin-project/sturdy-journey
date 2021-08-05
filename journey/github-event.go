package journey

import (
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/sturdy-journey/internal/config"
	"github.com/filecoin-project/sturdy-journey/internal/secretloader"

	"github.com/google/go-github/v37/github"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("sturdy-journey/github-journey")

type GithubEventHandler interface {
	HandleEvent(payload interface{}) error
}

// GithubEventJourney provides a basic journey to handle the common requirements for accepting and
// authenticating a github webhook.
type GithubEventJourney struct {
	webhookSecretKey secretloader.SecretLoader
	eventHandler     GithubEventHandler
	journeyName      string
}

func NewGithubEventJourney(cfg config.CommonJourney, eventHandler GithubEventHandler) *GithubEventJourney {
	return &GithubEventJourney{
		webhookSecretKey: secretloader.NewSecretLoader(cfg.SecretPath, time.Second*15),
		eventHandler:     eventHandler,
		journeyName:      cfg.Name,
	}
}

var ErrUnhandledEvent = fmt.Errorf("event not handled")

func (s *GithubEventJourney) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, secret, err := s.webhookSecretKey.Get()
	if err != nil {
		log.Errorw("failed to load webhook secret", "journey_name", s.journeyName, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	payload, err := github.ValidatePayload(r, secret)
	if err != nil {
		log.Errorw("failed to validate", "journey_name", s.journeyName, "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	webhookType := github.WebHookType(r)
	event, err := github.ParseWebHook(webhookType, payload)
	if err != nil {
		deliveryID := github.DeliveryID(r)
		log.Errorw("failed to parse incoming webhook", "journey_name", s.journeyName, "webhook_type", webhookType, "delivery_id", deliveryID, "err", err)
		return
	}

	log.Infow("incoming webhook", "journey_name", s.journeyName, "webhook_type", webhookType, "request_uri", r.RequestURI)

	if err := s.eventHandler.HandleEvent(event); err != nil {
		switch err {
		case ErrUnhandledEvent:
			w.WriteHeader(http.StatusBadRequest)
			log.Warnw("unhandled event", "journey_name", s.journeyName, "err", err)
		default:
			w.WriteHeader(http.StatusInternalServerError)
			log.Warnw("unhandled error", "journey_name", s.journeyName, "err", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
