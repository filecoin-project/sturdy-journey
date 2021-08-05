package main

import (
	"os"
	"strings"

	"github.com/filecoin-project/sturdy-journey/build"
	"github.com/filecoin-project/sturdy-journey/cmd/sturdy-journey/cmds"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	_ "github.com/filecoin-project/sturdy-journey/journey/greeting"
	_ "github.com/filecoin-project/sturdy-journey/journey/lotus"
)

var log = logging.Logger("sturdy-journey")

func main() {
	app := &cli.App{
		Name:    "sturdy-journey",
		Usage:   "sturdy-journey software suite",
		Version: build.Version(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level-named",
				Usage:   "common delimiated list of named loggers and log levels formatted as name:level",
				EnvVars: []string{"STURDY_JOURNEY_LOG_LEVEL_NAMED"},
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "set all sturdy journey loggers to level",
				EnvVars: []string{"STURDY_JOURNEY_LOG_LEVEL"},
				Value:   "warn",
			},
		},
		Before: func(cctx *cli.Context) error {
			setupLogging(cctx)
			return nil
		},
		Commands: cmds.Commands,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorw("exit", "err", err)
		os.Exit(1)
	}
}

func setupLogging(cctx *cli.Context) error {
	ll := cctx.String("log-level")
	if err := logging.SetLogLevel("sturdy-journey/*", ll); err != nil {
		return xerrors.Errorf("set log level: %w", err)
	}

	llnamed := cctx.String("log-level-named")
	if llnamed != "" {
		for _, llname := range strings.Split(llnamed, ",") {
			parts := strings.Split(llname, ":")
			if len(parts) != 2 {
				return xerrors.Errorf("invalid named log level format: %q", llname)
			}
			if err := logging.SetLogLevel(parts[0], parts[1]); err != nil {
				return xerrors.Errorf("set named log level %q to %q: %w", parts[0], parts[1], err)
			}

		}
	}

	return nil
}
