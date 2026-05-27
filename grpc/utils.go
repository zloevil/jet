package grpc

import (
	"context"
	"encoding/json"
	"github.com/zloevil/jet"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// toGrpcStatus converts error (AppError) to grpc status
func toGrpcStatus(err error) error {
	// check if it's app error
	if appErr, ok := jet.IsAppErr(err); ok {

		// is app error has grpc status populated, then set it up
		var grpcStatus = codes.Unknown
		if appErr.GrpcStatus() != nil {
			c := *appErr.GrpcStatus()
			grpcStatus = codes.Code(c)
		}
		st := status.New(grpcStatus, appErr.Message())

		// marshal fields
		ff, _ := json.Marshal(appErr.Fields())

		// put details to gRPC status
		st, _ = st.WithDetails(&AppErrorDetails{
			Code:   appErr.Code(),
			Type:   appErr.Type(),
			Fields: ff,
		})
		return st.Err()
	} else {
		return status.New(codes.Unknown, err.Error()).Err()
	}
}

// ToAppError converts gRPC status to AppError
func ToAppError(ctx context.Context, method string, err error) error {
	var res error
	st := status.Convert(err)
	details := st.Details()
	if len(details) > 0 {
		errDet := details[0]
		if appErr, ok := errDet.(*AppErrorDetails); ok {
			var ff jet.KV
			if e := json.Unmarshal(appErr.Fields, &ff); e == nil {
				res = jet.NewAppErrBuilder(appErr.Code, st.Message()).F(ff).Type(appErr.Type).GrpcSt(uint32(st.Code())).Err()
			}
		}
	} else {
		res = ErrGrpcClientError(ctx, err, method)
	}
	return res
}

// ToTimestamp converts time to GRPC timestamp
func ToTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	} else {
		return timestamppb.New(*t)
	}
}

// ToTime converts GRPC timestamp to time
func ToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	} else {
		t := ts.AsTime()
		return &t
	}
}
