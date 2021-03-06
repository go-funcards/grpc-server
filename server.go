package grpcserver

import (
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"net"
)

var _ Server = (*server)(nil)

type Server interface {
	Server() *grpc.Server
	SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus)
	Start() error
	Stop()
}

type server struct {
	ctx  context.Context
	lis  net.Listener
	log  logrus.FieldLogger
	gSrv *grpc.Server
	hSrv *health.Server
}

func New(ctx context.Context, listener net.Listener, log logrus.FieldLogger, opts ...grpc.ServerOption) *server {
	return &server{
		ctx:  ctx,
		lis:  listener,
		log:  log,
		gSrv: grpc.NewServer(opts...),
	}
}

func (s *server) Server() *grpc.Server {
	return s.gSrv
}

func (s *server) SetServingStatus(service string, servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.hSrv.SetServingStatus(service, servingStatus)
}

func (s *server) Start() error {
	ctx := s.ctx
	g, _ := errgroup.WithContext(s.ctx)

	// Add HealthChecks only after all user services are registered
	s.hSrv = health.NewServer()
	s.hSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	for name := range s.gSrv.GetServiceInfo() {
		s.hSrv.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
	}
	grpc_health_v1.RegisterHealthServer(s.gSrv, s.hSrv)

	// registers the server reflection service on the given gRPC server.
	reflection.Register(s.gSrv)

	g.Go(func() error {
		return s.gSrv.Serve(s.lis)
	})

	g.Go(func() (err error) {
		// listen for the interrupt signal
		<-ctx.Done()

		// log situation
		switch ctx.Err() {
		case context.DeadlineExceeded:
			s.log.Debug("context timeout exceeded")
		case context.Canceled:
			s.log.Debug("context cancelled by interrupt signal")
		}

		// Gracefully stop healthServer
		s.hSrv.Shutdown()

		s.log.Info("stopping grpc server...")

		// Gracefully stop server
		s.gSrv.GracefulStop()

		return
	})

	// Wait for all tasks to be finished or return if error occur at any task.
	return g.Wait()
}

func (s *server) Stop() {
	s.gSrv.Stop()
}
