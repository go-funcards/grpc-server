package grpcserver

import (
	"context"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Register func(srv *grpc.Server)

func Start(ctx context.Context, lis net.Listener, fn Register, logger *zap.Logger, opts ...grpc.ServerOption) {
	appCtx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer stop()

	g, ctx := errgroup.WithContext(appCtx)

	srv := NewWithDefaultInterceptors(ctx, lis, logger, opts...)

	fn(srv.Server())

	g.Go(srv.Start)

	go func() {
		if err := g.Wait(); err != nil {
			logger.Fatal("Unexpected error", zap.Error(err))
		}
		logger.Info("Goodbye.....")
		os.Exit(0)
	}()

	// Listen for the interrupt signal.
	<-appCtx.Done()

	// notify user of shutdown
	switch ctx.Err() {
	case context.DeadlineExceeded:
		logger.Info("Shutting down gracefully, press Ctrl+C again to force", zap.String("cause", "timeout"))
	case context.Canceled:
		logger.Info("Shutting down gracefully, press Ctrl+C again to force", zap.String("cause", "interrupt"))
	}

	// Restore default behavior on the interrupt signal.
	stop()

	// Perform application shutdown with a maximum timeout of 30s.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// force termination after shutdown timeout
	<-timeoutCtx.Done()
	logger.Error("Shutdown grace period elapsed. Force exit...")
	// force stop any daemon services here:
	srv.Stop()
	os.Exit(1)
}
