package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func setupContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	var term = make(chan os.Signal)
	go func() {
		select {
		case <-term:
			logrus.Warningf("signal received, stopping...")
			cancel()
		}
	}()
	signal.Notify(term, syscall.SIGTERM, os.Interrupt)

	return ctx
}
