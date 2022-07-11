package grpcserver

import (
	"context"
	"fmt"
	"github.com/go-funcards/validate"
	"github.com/go-playground/validator/v10"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidatorUnaryServerInterceptor(v *validate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		err := reqValidation(v, req)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func ValidatorStreamServerInterceptor(v *validate.Validator) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &serverStream{ServerStream: stream, v: v})
	}
}

type serverStream struct {
	grpc.ServerStream
	v *validate.Validator
}

func (s *serverStream) RecvMsg(m interface{}) error {
	err := reqValidation(s.v, m)
	if err != nil {
		return err
	}
	return s.ServerStream.RecvMsg(m)
}

func reqValidation(v *validate.Validator, req any) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "Request struct is required")
	}

	err := v.ValidateStruct(req)
	if err != nil {
		switch tmp := err.(type) {
		case validate.SliceValidateError:
			br := new(errdetails.BadRequest)
			for _, items := range tmp {
				if vErrors, ok := items.(validator.ValidationErrors); ok {
					addValidationErrors(br, vErrors)
				}
			}
			st, err1 := status.New(codes.InvalidArgument, tmp.Error()).WithDetails(br)
			if err1 != nil {
				panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err1))
			}
			err = st.Err()
		case validator.ValidationErrors:
			br := new(errdetails.BadRequest)
			addValidationErrors(br, tmp)
			st, err1 := status.New(codes.InvalidArgument, tmp.Error()).WithDetails(br)
			if err1 != nil {
				panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err1))
			}
			err = st.Err()
		}
	}
	return err
}

func addValidationErrors(br *errdetails.BadRequest, vErrors validator.ValidationErrors) {
	for _, ve := range vErrors {
		v := &errdetails.BadRequest_FieldViolation{
			Field:       ve.StructField(),
			Description: ve.Error(),
		}
		br.FieldViolations = append(br.FieldViolations, v)
	}
}
