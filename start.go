package grpcserver

import (
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Register func(srv *grpc.Server)

func Start(ctx context.Context, lis net.Listener, fn Register, log logrus.FieldLogger, opts ...grpc.ServerOption) {
	appCtx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer stop()

	g, ctx := errgroup.WithContext(appCtx)

	srv := New(ctx, lis, log, opts...)

	fn(srv.Server())

	g.Go(srv.Start)

	go func() {
		if err := g.Wait(); err != nil {
			log.WithField("error", err).Fatal("unexpected error")
		}
		log.Info("goodbye.....")
		os.Exit(0)
	}()

	// Listen for the interrupt signal.
	<-appCtx.Done()

	// notify user of shutdown
	switch ctx.Err() {
	case context.DeadlineExceeded:
		log.WithField("cause", "timeout").Info("shutting down gracefully, press Ctrl+C again to force")
	case context.Canceled:
		log.WithField("cause", "interrupt").Info("shutting down gracefully, press Ctrl+C again to force")
	}

	// Restore default behavior on the interrupt signal.
	stop()

	// Perform application shutdown with a maximum timeout of 30s.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// force termination after shutdown timeout
	<-timeoutCtx.Done()
	log.Error("shutdown grace period elapsed. Force exit...")
	// force stop any daemon services here:
	srv.Stop()
	os.Exit(1)
}
