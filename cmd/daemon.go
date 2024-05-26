package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/CubicrootXYZ/gologger"
	"github.com/Cubicroots-Playground/trollinfo/internal/angelapi"
	"github.com/Cubicroots-Playground/trollinfo/internal/matrixmessenger"
	"github.com/Cubicroots-Playground/trollinfo/internal/shiftnotifier"
	"golang.org/x/sync/errgroup"
)

func main() {
	angelConfig := angelapi.Config{}
	angelConfig.ParseFromEnvironment()

	matrixConfig := matrixmessenger.Config{}
	matrixConfig.ParseFromEnvironment()

	notifierConfig := shiftnotifier.Config{}
	notifierConfig.ParseFromEnvironment()

	angelService := angelapi.New(&angelConfig)
	messenger, err := matrixmessenger.NewMessenger(
		&matrixConfig, gologger.New(gologger.LogLevelDebug, 0),
	)
	if err != nil {
		panic(err)
	}

	shiftNotifier := shiftnotifier.New(&notifierConfig, angelService, messenger)

	eg, ctx := errgroup.WithContext(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case s := <-sigChan:
			slog.Info("received signal, shutting down", "signal", s)
		case <-ctx.Done():
			slog.Info("at least one process exited, shutting down")
		}

		err = shiftNotifier.Stop()
		if err != nil {
			slog.Error("failed stopping notifier", "error", err.Error())
		}
	}()

	eg.Go(shiftNotifier.Start)
	err = eg.Wait()
	if err != nil {
		slog.Error("error group failed", "error", err.Error())
	}
}
