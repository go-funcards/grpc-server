package grpcserver

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		res, err := handler(ctx, req)
		return res, normalizeError(err, logger)
	}
}

func ErrorStreamServerInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		return normalizeError(err, logger)
	}
}

func normalizeError(err error, logger *zap.Logger) error {
	if err == nil {
		return nil
	}

	logger.Error("error interceptor", zap.Error(err))

	if _, ok := status.FromError(err); !ok {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = status.Error(codes.NotFound, err.Error())
		} else if mongo.IsDuplicateKeyError(err) {
			err = status.Error(codes.AlreadyExists, err.Error())
		} else if mongo.IsTimeout(err) {
			err = status.Error(codes.DeadlineExceeded, err.Error())
		} else {
			err = status.Error(codes.Internal, err.Error())
		}
	}

	return err
}
