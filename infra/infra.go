package infra

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func Context() context.Context {
	ctx, cancelFunc := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("%s signal received, cancelling service context\n", sig.String())
		cancelFunc()
	}()

	return ctx
}
