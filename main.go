package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")

	appName     = "appuio-cloud-reporting"
	appLongName = "Reporting for APPUiO Cloud"

	// envPrefix is the global prefix to use for the keys in environment variables
	envPrefix = "ACR"
)

func main() {
	ctx, stop, app := newApp()
	defer stop()
	err := app.RunContext(ctx, os.Args)
	// If required flags aren't set, it will exit before we could set up logging
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newApp() (context.Context, context.CancelFunc, *cli.App) {
	logInstance := &atomic.Value{}
	logInstance.Store(logr.Discard())
	app := &cli.App{
		Name:     appName,
		Usage:    appLongName,
		Version:  fmt.Sprintf("%s, revision=%s, date=%s", version, commit, date),
		Compiled: compilationDate(),

		EnableBashCompletion: true,

		Before: setupLogging,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"verbose", "d"},
				Usage:   "sets the log level to debug",
				EnvVars: envVars("DEBUG"),
			},
			&cli.StringFlag{
				Name:        "log-format",
				Usage:       "sets the log format (values: [json, console])",
				EnvVars:     envVars("LOG_FORMAT"),
				DefaultText: "console",
			},
		},
		Commands: []*cli.Command{
			newMigrateCommand(),
			newReportCommand(),
			newCheckMissingCommand(),
			newInvoiceCommand(),
			newTmapCommand(),
		},
		ExitErrHandler: func(context *cli.Context, err error) {
			if err == nil {
				return
			}
			// Don't show stack trace if the error is expected (someone called cli.Exit())
			var exitErr cli.ExitCoder
			if errors.As(err, &exitErr) {
				cli.HandleExitCoder(err)
				return
			}
			AppLogger(context.Context).WithCallDepth(1).Error(err, "fatal error")
			cli.OsExiter(1)
		},
	}
	// There is logr.NewContext(...) which returns a context that carries the logger instance.
	// However, since we are configuring and replacing this logger after starting up and parsing the flags,
	// we'll store a thread-safe atomic reference.
	parentCtx := context.WithValue(context.Background(), loggerContextKey{}, logInstance)
	ctx, stop := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	return ctx, stop, app
}

// env combines envPrefix with given suffix delimited by underscore.
func env(suffix string) string {
	if envPrefix == "" {
		return suffix
	}
	return envPrefix + "_" + suffix
}

// envVars combines envPrefix with each given suffix delimited by underscore.
func envVars(suffixes ...string) []string {
	arr := make([]string, len(suffixes))
	for i := range suffixes {
		arr[i] = env(suffixes[i])
	}
	return arr
}

func compilationDate() time.Time {
	compiled, err := time.Parse(time.RFC3339, date)
	if err != nil {
		// an empty Time{} causes cli.App to guess it from binary's file timestamp.
		return time.Time{}
	}
	return compiled
}
