package unknownconnect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ connect.Interceptor = (*interceptor)(nil) // we make sure it implements the interface

type interceptor struct {
	callback func(proto.Message)
}

func NewInterceptor(callback func(proto.Message)) *interceptor {
	return &interceptor{callback: callback}
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		isClient := req.Spec().IsClient
		if !isClient {
			checkForUnknownFields(req.Any(), i.callback)
		}
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}
		if isClient {
			checkForUnknownFields(resp.Any(), i.callback)
		}
		return resp, err
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		return &WrappedClientConn{
			StreamingClientConn: conn,
			callback:            i.callback,
		}
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, &WrappedHandlerConn{
			StreamingHandlerConn: conn,
			callback:             i.callback,
		})
	}
}

type WrappedHandlerConn struct {
	connect.StreamingHandlerConn
	callback func(proto.Message)
}

func (w *WrappedHandlerConn) Receive(msg any) error {
	checkForUnknownFields(msg, w.callback)
	return w.StreamingHandlerConn.Receive(msg)
}

func (w *WrappedHandlerConn) RequestHeader() http.Header {
	return w.StreamingHandlerConn.RequestHeader()
}

type WrappedClientConn struct {
	connect.StreamingClientConn
	callback func(proto.Message)
}

func (w *WrappedClientConn) Receive(msg any) error {
	checkForUnknownFields(msg, w.callback)
	return w.StreamingClientConn.Receive(msg)
}

func checkForUnknownFields(m any, callback func(proto.Message)) {
	if msg, ok := (m).(proto.Message); ok {
		if messageHasUnknown(msg.ProtoReflect()) {
			callback(msg)
			return
		}

	}
}

func messageHasUnknown(msg protoreflect.Message) bool {
	if len(msg.GetUnknown()) > 0 {
		return true
	}

	var hasUnknown bool
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch fd.Kind() {
		case protoreflect.MessageKind:
			if messageHasUnknown(v.Message()) {
				hasUnknown = true
				return false
			}
		default:
		}
		return true
	})
	return hasUnknown
}
