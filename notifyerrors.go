package jet

import "errors"

// Error error for events
type Error struct {
	Message    string                 `json:"message"`               // Message is error description
	Code       string                 `json:"code,omitempty"`        // Code is error code provided by error producer
	Type       string                 `json:"type,omitempty"`        // Type is error type (panic, system, business)
	HttpStatus *uint32                `json:"http_status,omitempty"` // HttpStatus http status
	Details    map[string]interface{} `json:"details,omitempty"`     // Details is additional info provided by error producer
	Stack      string                 `json:"stack,omitempty"`       // Stack trace
	GrpcStatus *uint32                `json:"grpc_status,omitempty"` // GrpcStatus grpc status
}

func ToErrorFromString(e *string) (*Error, error) {
	if e == nil || len(*e) == 0 {
		return nil, nil
	}
	return JsonDecode[Error]([]byte(*e))
}

func ToErrorFromStringEmptyIfInvalid(e *string) *Error {
	res, _ := ToErrorFromString(e)
	return res
}

func ToError(err error) *Error {
	if err == nil {
		return nil
	}
	if appErr, ok := IsAppErr(err); ok {
		return &Error{
			Code:       appErr.Code(),
			Type:       appErr.Type(),
			Message:    appErr.Message(),
			HttpStatus: appErr.HttpStatus(),
			GrpcStatus: appErr.GrpcStatus(),
			Details:    appErr.Fields(),
			// Do not send stack in notification
			// Stack:      appErr.WithStack(),
		}
	}
	return &Error{
		Message: err.Error(),
	}
}

func (e *Error) ToError() error {
	if e == nil {
		return nil
	}
	if e.Code != "" {
		builder := NewAppErrBuilder(e.Code, e.Message).Type(e.Type).F(e.Details)
		if e.HttpStatus != nil {
			builder.HttpSt(*e.HttpStatus)
		}
		if e.GrpcStatus != nil {
			builder.GrpcSt(*e.GrpcStatus)
		}
		return builder.Err()
	}
	return errors.New(e.Message)
}
