// Command scheduler generates check-ins and medication doses, and sweeps
// overdue rows into 'missed'.
//
// It runs as its own process so it can be scaled and restarted independently
// of the API. Its statements are idempotent, so running it alongside a second
// copy — or re-running it after a crash — is safe.
//
// Pass -once to perform a single pass and exit, which is what the end-to-end
// check and any cron-style deployment use.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OderoCeasar/afyalink/backend/internal/config"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
	"github.com/OderoCeasar/afyalink/backend/internal/logging"
	"github.com/OderoCeasar/afyalink/backend/internal/scheduler"
)

func main() {
	once := flag.Bool("once", false, "run a single pass and exit")
	flag.Parse()

	if err := run(*once); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(once bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	worker := scheduler.New(pool, logger, cfg.SchedulerPeriod)

	if once {
		result, err := worker.RunOnce(ctx)
		if err != nil {
			return err
		}
		logger.Info("scheduler single pass complete",
			"checkins_created", result.CheckinsCreated,
			"dose_logs_created", result.DoseLogsCreated,
			"checkins_missed", result.CheckinsMissed,
			"doses_missed", result.DosesMissed,
			"missed_dose_alerts", result.MissedDoseAlerts,
		)
		return nil
	}

	logger.Info("scheduler starting", "period", cfg.SchedulerPeriod.String())

	if err := worker.Run(ctx); err != nil && !isShutdown(err) {
		return err
	}
	return nil
}

// isShutdown reports whether an error is just the signal-cancelled context.
func isShutdown(err error) bool {
	return err == context.Canceled
}
