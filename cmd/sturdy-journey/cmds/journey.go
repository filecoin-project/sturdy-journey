package cmds

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sturdy-journey/build"
	"github.com/filecoin-project/sturdy-journey/internal/config"
	"github.com/filecoin-project/sturdy-journey/internal/journey-service"
	"github.com/filecoin-project/sturdy-journey/internal/operator"
	"github.com/filecoin-project/sturdy-journey/registry"
)

var log = logging.Logger("sturdy-journey/cmds")

var (
	routeTimeout       = 30 * time.Second
	svrShutdownTimeout = 10 * time.Second
	ctxCancelWait      = 10 * time.Second
)

type versionKey struct{}

var cmdJourneyService = &cli.Command{
	Name:        "journey-service",
	Usage:       "journey service for sturdy journey",
	Description: "Description",
	Flags:       []cli.Flag{},
	Subcommands: []*cli.Command{
		{
			Name:  "operator",
			Usage: "commands for interacting with the running service through the operator jsonrpc api",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "operator-api",
					Usage:   "host and port of operator api",
					EnvVars: []string{"STURDY_JOURNEY_OPERATOR_API"},
					Value:   "http://localhost:5101",
				},
				&cli.StringFlag{
					Name:    "api-info",
					Usage:   "",
					EnvVars: []string{"STURDY_JOURNEY_OPERATOR_API_INFO"},
					Hidden:  true,
				},
			},
			Before: func(cctx *cli.Context) error {
				if cctx.IsSet("api-info") {
					return nil
				}

				apiInfo := fmt.Sprintf("%s", cctx.String("operator-api"))
				return cctx.Set("api-info", apiInfo)
			},
			Subcommands: []*cli.Command{
				{
					Name:  "version",
					Usage: "prints local and remote version",
					Action: func(cctx *cli.Context) error {
						ctx := context.Background()

						api, closer, err := getCliClient(ctx, cctx)
						defer closer()
						if err != nil {
							return err
						}

						version, err := api.Version(ctx)
						if err != nil {
							return err
						}

						fmt.Printf("local:  %s\n", build.Version())
						fmt.Printf("remote: %s\n", version)

						return nil
					},
				},
				{
					Name:  "log-list",
					Usage: "list available loggers",
					Action: func(cctx *cli.Context) error {
						ctx := context.Background()

						api, closer, err := getCliClient(ctx, cctx)
						defer closer()
						if err != nil {
							return err
						}

						loggers, err := api.LogList(ctx)
						if err != nil {
							return err
						}

						for _, logger := range loggers {
							fmt.Println(logger)
						}

						return nil
					},
				},
				{
					Name:      "log-set-level",
					Usage:     "set log level",
					ArgsUsage: "<level>",
					Description: TrimDescription(`
						The logger flag can be specified multiple times.

						eg) log set-level --chain --system chainxchg debug

						log levels
						- debug
						- info
						- warn
						- error
					`),
					Flags: []cli.Flag{
						&cli.StringSliceFlag{
							Name:  "logger",
							Usage: "limit to log system",
							Value: &cli.StringSlice{},
						},
					},
					Action: func(cctx *cli.Context) error {
						ctx := context.Background()

						api, closer, err := getCliClient(ctx, cctx)
						defer closer()
						if err != nil {
							return err
						}

						if !cctx.Args().Present() {
							return fmt.Errorf("level is required")
						}

						loggers := cctx.StringSlice("logger")
						if len(loggers) == 0 {
							var err error
							loggers, err = api.LogList(ctx)
							if err != nil {
								return err
							}
						}

						for _, logger := range loggers {
							if err := api.LogSetLevel(ctx, logger, cctx.Args().First()); err != nil {
								return xerrors.Errorf("setting log level on %s: %w", logger, err)
							}
						}

						return nil
					},
				},
			},
		},
		{
			Name:  "default-config",
			Usage: "prints the default configuration",
			Description: TrimDescription(`
				Produces a commented out configuration of the journey service by default.
				Additionally can print a commented out configuration of each journey
				by specifying the journey using the '--journey' flag.

				Examples
				 default-config --journey greeting
			`),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "journey",
					Usage: "produce the default config for the named journey",
					Value: "",
				},
			},
			Action: func(cctx *cli.Context) error {
				var icfg interface{}

				if cctx.IsSet("journey") {
					journey, err := registry.Get(cctx.String("journey"))
					if err != nil {
						return err
					}

					icfg = journey.DefaultConfig
				} else {
					cfg := config.DefaultConfig()
					registered := registry.Registered()

					for _, name := range registered {
						cfg.Journeys = append(cfg.Journeys, config.CommonJourney{
							Name:       name,
							Enabled:    false,
							RoutePath:  fmt.Sprintf("/journey/%s", name),
							SecretPath: fmt.Sprintf("/secrets/%s", name),
							ConfigPath: fmt.Sprintf("/configs/%s", name),
						})
					}

					icfg = cfg
				}

				bs, err := config.ConfigComment(icfg)
				if err != nil {
					return err
				}

				fmt.Printf("%s", string(bs))

				return nil
			},
		},
		{
			Name:  "run",
			Usage: "start the sturdy journey service",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "service-listen",
					Usage:   "host and port to listen on",
					EnvVars: []string{"STURDY_JOURNEY_SERVICE_LISTEN"},
					Value:   "localhost:5100",
				},
				&cli.StringFlag{
					Name:    "operator-listen",
					Usage:   "host and port to listen on",
					EnvVars: []string{"STURDY_JOURNEY_OPERATOR_LISTEN"},
					Value:   "localhost:5101",
				},
				&cli.StringFlag{
					Name:    "config-path",
					Usage:   "path to configuration file",
					EnvVars: []string{"STURDY_JOURNEY_CONFIG_PATH"},
					Value:   "./config.toml",
				},
			},
			Action: func(cctx *cli.Context) error {
				ctx, cancelFunc := context.WithCancel(context.Background())
				ctx = context.WithValue(ctx, versionKey{}, build.Version())

				signalChan := make(chan os.Signal, 1)
				signal.Notify(signalChan, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

				s := journeyservice.NewJourneyService(ctx)

				if err := s.SetupService(cctx.String("config-path")); err != nil {
					return err
				}

				svr := &http.Server{
					Addr:    cctx.String("service-listen"),
					Handler: s.ServiceRouter,
					BaseContext: func(listener net.Listener) context.Context {
						return context.Background()
					},
				}

				go func() {
					err := svr.ListenAndServe()
					switch err {
					case nil:
					case http.ErrServerClosed:
						log.Infow("server closed")
					case context.Canceled:
						log.Infow("context cancled")
					default:
						log.Errorw("error shutting down service server", "err", err)
					}
				}()

				if err := s.SetupOperator(); err != nil {
					return err
				}

				osvr := http.Server{
					Addr:    cctx.String("operator-listen"),
					Handler: s.OperatorRouter,
					BaseContext: func(listener net.Listener) context.Context {
						return context.Background()
					},
				}

				go func() {
					log.Debugw("Running")
					err := osvr.ListenAndServe()
					switch err {
					case nil:
					case http.ErrServerClosed:
						log.Infow("server closed")
					case context.Canceled:
						log.Infow("context cancled")
					default:
						log.Errorw("error shutting down internal server", "err", err)
					}
				}()

				<-signalChan
				s.Shutdown()

				t := time.NewTimer(svrShutdownTimeout)

				shutdownChan := make(chan error)
				go func() {
					shutdownChan <- svr.Shutdown(ctx)
				}()

				select {
				case err := <-shutdownChan:
					if err != nil {
						log.Errorw("shutdown finished with an error", "err", err)
					} else {
						log.Infow("shutdown finished successfully")
					}
				case <-t.C:
					log.Warnw("shutdown timed out")
				}

				cancelFunc()
				time.Sleep(ctxCancelWait)

				log.Infow("closing down database connections")
				s.Close()

				if err := osvr.Shutdown(ctx); err != nil {
					switch err {
					case nil:
					case http.ErrServerClosed:
						log.Infow("server closed")
					case context.Canceled:
						log.Infow("context cancled")
					default:
						log.Errorw("error shutting down operator server", "err", err)
					}
				}

				log.Infow("existing")

				return nil

			},
		},
	},
}

func getCliClient(ctx context.Context, cctx *cli.Context) (operator.Operator, jsonrpc.ClientCloser, error) {
	ai := operator.ParseApiInfo(cctx.String("api-info"))
	url, err := ai.DialArgs("v0")
	if err != nil {
		return nil, func() {}, err
	}

	return operator.NewOperatorClient(ctx, url, ai.AuthHeader())
}

func TrimDescription(desc string) string {
	lines := strings.Split(desc, "\n")
	lines = lines[1:]
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, "\t")
	}
	return strings.Join(lines, "\n")
}
