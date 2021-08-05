package journeyservice

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"

	"github.com/filecoin-project/sturdy-journey/internal/config"
	"github.com/filecoin-project/sturdy-journey/internal/operator"
	"github.com/filecoin-project/sturdy-journey/registry"
)

var log = logging.Logger("sturdy-journey/service/journey")

type JourneyService struct {
	ctx            context.Context
	ServiceRouter  *mux.Router
	OperatorRouter *mux.Router

	rpc      *jsonrpc.RPCServer
	operator operator.Operator

	ready   bool
	readyMu sync.Mutex
}

func NewJourneyService(ctx context.Context) *JourneyService {
	return &JourneyService{
		ctx:            ctx,
		ServiceRouter:  mux.NewRouter(),
		OperatorRouter: mux.NewRouter(),
		rpc:            jsonrpc.NewServer(),
	}

}

func (bs *JourneyService) SetupService(cfgPath string) error {
	defer bs.setReady()
	mdlw := middleware.New(middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})
	bs.ServiceRouter.Use(std.HandlerProvider("", mdlw))

	icfg, err := config.FromFile(cfgPath, &config.Config{})
	if err != nil {
		return err
	}

	cfg := icfg.(*config.Config)

	for _, jcfg := range cfg.Journeys {
		log.Debugw("loading journey", "name", jcfg.Name)
		journey, err := registry.Get(jcfg.Name)
		if err != nil {
			log.Errorw("failed to get journey", "journey", jcfg.Name, "err", err)
			continue
		}

		handler, err := journey.Constructor(jcfg)
		if err != nil {
			log.Errorw("failed to build journey", "journey", jcfg.Name, "err", err)
			continue
		}

		bs.ServiceRouter.Handle(jcfg.RoutePath, handler)
	}

	return bs.dumpRoutes(bs.ServiceRouter)
}

func (bs *JourneyService) SetupOperator() error {
	bs.operator = &operator.OperatorImpl{}
	bs.rpc.Register("Operator", bs.operator)
	bs.OperatorRouter.Handle("/rpc/v0", bs.rpc)

	bs.OperatorRouter.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	bs.OperatorRouter.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	bs.OperatorRouter.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		isReady := bs.IsReady()

		if isReady {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	bs.OperatorRouter.Handle("/metrics", promhttp.Handler())

	return bs.dumpRoutes(bs.OperatorRouter)
}

func (bs *JourneyService) dumpRoutes(router *mux.Router) error {
	return router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			log.Debugw("route template", "path", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			log.Debugw("route regexp", "path", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			log.Debugw("queries templates", "queries", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			log.Debugw("queries regex", "queries", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			log.Debugw("method", "queries", strings.Join(methods, ","))
		}
		return nil
	})
}

func (bs *JourneyService) setReady() {
	bs.readyMu.Lock()
	defer bs.readyMu.Unlock()
	bs.ready = true
}

func (bs *JourneyService) IsReady() bool {
	bs.readyMu.Lock()
	defer bs.readyMu.Unlock()
	return bs.ready
}

func (bs *JourneyService) Shutdown() {
	bs.unsetReady()
}

func (bs *JourneyService) unsetReady() {
	bs.readyMu.Lock()
	defer bs.readyMu.Unlock()
	bs.ready = false
}

func (bs *JourneyService) Close() {
}
