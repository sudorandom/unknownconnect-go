package unknownconnect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

var _ connect.Interceptor = (*interceptor)(nil) // we make sure it implements the interface

// UnknownCallback is called whenever there is an unknown field. Note that the proto.Message
// is the base protobuf message for the RPC call. The message with the unknown field(s) can
// be nested deeper into this given message.
type UnknownCallback func(context.Context, connect.Spec, proto.Message) error

type interceptorOpts struct {
	drop      bool
	callbacks []UnknownCallback
}

type interceptor struct {
	opts *interceptorOpts
}

// NewInterceptor creates a new interceptor appropriate to pass into a new ConnectRPC client or server.
// The given callback is called whenever a message is detected to have an unknown field. That means
// a field is being given to this client/server that does not. The callback can decide what to do.
// Any error returned from the callback will be used as an error in the request or response.
func NewInterceptor(opts ...option) *interceptor {
	o := &interceptorOpts{
		callbacks: []UnknownCallback{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return &interceptor{opts: o}
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		spec := req.Spec()
		isClient := spec.IsClient
		if !isClient {
			if err := handleMessage(ctx, req.Any(), spec, i.opts); err != nil {
				return nil, err
			}
		}
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}
		if isClient {
			if err := handleMessage(ctx, resp.Any(), spec, i.opts); err != nil {
				return resp, err
			}
		}
		return resp, err
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		return &wrappedClientConn{
			ctx:                 ctx,
			StreamingClientConn: conn,
			spec:                spec,
			opts:                i.opts,
		}
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, &wrappedHandlerConn{
			ctx:                  ctx,
			StreamingHandlerConn: conn,
			spec:                 conn.Spec(),
			opts:                 i.opts,
		})
	}
}

type wrappedHandlerConn struct {
	connect.StreamingHandlerConn
	ctx  context.Context
	spec connect.Spec
	opts *interceptorOpts
}

func (w *wrappedHandlerConn) Receive(msg any) error {
	if err := handleMessage(w.ctx, msg, w.spec, w.opts); err != nil {
		return err
	}
	return w.StreamingHandlerConn.Receive(msg)
}

func (w *wrappedHandlerConn) RequestHeader() http.Header {
	return w.StreamingHandlerConn.RequestHeader()
}

type wrappedClientConn struct {
	connect.StreamingClientConn
	ctx  context.Context
	spec connect.Spec
	opts *interceptorOpts
}

func (w *wrappedClientConn) Receive(msg any) error {
	if err := handleMessage(w.ctx, msg, w.spec, w.opts); err != nil {
		return err
	}
	return w.StreamingClientConn.Receive(msg)
}

func handleMessage(ctx context.Context, m any, spec connect.Spec, opts *interceptorOpts) error {
	if msg, ok := (m).(proto.Message); ok {
		if opts.drop {
			defer func() {
				DropUnknownFields(msg.ProtoReflect())
			}()
		}
		if len(opts.callbacks) > 0 && MessageHasUnknownFields(msg.ProtoReflect()) {
			for _, cb := range opts.callbacks {
				if err := cb(ctx, spec, msg); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
