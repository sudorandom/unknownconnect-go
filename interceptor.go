package unknownconnect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ connect.Interceptor = (*interceptor)(nil) // we make sure it implements the interface

type UnknownCallback func(context.Context, connect.Spec, proto.Message) error

type interceptor struct {
	callback UnknownCallback
}

func NewInterceptor(callback UnknownCallback) *interceptor {
	return &interceptor{callback: callback}
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		isClient := req.Spec().IsClient
		if !isClient {
			if err := checkForUnknownFields(ctx, req.Any(), req.Spec(), i.callback); err != nil {
				return nil, err
			}
		}
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}
		if isClient {
			if err := checkForUnknownFields(ctx, resp.Any(), req.Spec(), i.callback); err != nil {
				return resp, err
			}
		}
		return resp, err
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		return &WrappedClientConn{
			ctx:                 ctx,
			StreamingClientConn: conn,
			callback:            i.callback,
			spec:                spec,
		}
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {

		return next(ctx, &WrappedHandlerConn{
			ctx:                  ctx,
			StreamingHandlerConn: conn,
			callback:             i.callback,
			spec:                 conn.Spec(),
		})
	}
}

type WrappedHandlerConn struct {
	connect.StreamingHandlerConn
	ctx      context.Context
	callback UnknownCallback
	spec     connect.Spec
}

func (w *WrappedHandlerConn) Receive(msg any) error {
	if err := checkForUnknownFields(w.ctx, msg, w.spec, w.callback); err != nil {
		return err
	}
	return w.StreamingHandlerConn.Receive(msg)
}

func (w *WrappedHandlerConn) RequestHeader() http.Header {
	return w.StreamingHandlerConn.RequestHeader()
}

type WrappedClientConn struct {
	connect.StreamingClientConn
	ctx      context.Context
	callback UnknownCallback
	spec     connect.Spec
}

func (w *WrappedClientConn) Receive(msg any) error {
	if err := checkForUnknownFields(w.ctx, msg, w.spec, w.callback); err != nil {
		return err
	}
	return w.StreamingClientConn.Receive(msg)
}

func checkForUnknownFields(ctx context.Context, m any, spec connect.Spec, callback UnknownCallback) error {
	if msg, ok := (m).(proto.Message); ok {
		if MessageHasUnknownFields(msg.ProtoReflect()) {
			return callback(ctx, spec, msg)
		}
	}
	return nil
}

func MessageHasUnknownFields(msg protoreflect.Message) bool {
	if len(msg.GetUnknown()) > 0 {
		return true
	}

	var hasUnknown bool
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch fd.Kind() {
		case protoreflect.MessageKind:
			if MessageHasUnknownFields(v.Message()) {
				hasUnknown = true
				return false
			}
		default:
		}
		return true
	})
	return hasUnknown
}
