package unknownconnect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type dropInterceptor struct{}

// NewDropUnknownInterceptor will recursively clean any unknown fields on any incoming messages
func NewDropUnknownInterceptor() *dropInterceptor {
	return &dropInterceptor{}
}

func (i *dropInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		spec := req.Spec()
		isClient := spec.IsClient
		if !isClient {
			dropUnknownFieldsAny(req.Any())
		}
		resp, err := next(ctx, req)
		if err != nil {
			return resp, err
		}
		if isClient {
			dropUnknownFieldsAny(resp.Any())
		}
		return resp, err
	}
}

func (i *dropInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return &wrappedDropClientConn{}
	}
}

func (i *dropInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, &wrappedDropHandlerConn{})
	}
}

type wrappedDropHandlerConn struct {
	connect.StreamingHandlerConn
}

func (w *wrappedDropHandlerConn) Receive(msg any) error {
	dropUnknownFieldsAny(msg)
	return w.StreamingHandlerConn.Receive(msg)
}

func (w *wrappedDropHandlerConn) RequestHeader() http.Header {
	return w.StreamingHandlerConn.RequestHeader()
}

type wrappedDropClientConn struct {
	connect.StreamingClientConn
}

func (w *wrappedDropClientConn) Receive(msg any) error {
	dropUnknownFieldsAny(msg)
	return w.StreamingClientConn.Receive(msg)
}

// dropUnknownFieldsAny recursively drops any unknown fields from the provided protobuf message.
func dropUnknownFieldsAny(msg any) {
	pmsg, ok := (msg).(proto.Message)
	if !ok {
		return
	}
	DropUnknownFields(pmsg.ProtoReflect())
}

// DropUnknownFields recursively drops any unknown fields from the provided protobuf message.
func DropUnknownFields(msg protoreflect.Message) {
	if len(msg.GetUnknown()) > 0 {
		msg.SetUnknown(nil)
	}

	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dropFieldFromUnknownField(fd, v)
		return true
	})
}

func dropFieldFromUnknownField(fd protoreflect.FieldDescriptor, v protoreflect.Value) {
	if fd.IsMap() {
		v.Map().Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
			dropFieldFromUnknownField(fd.MapValue(), mv)
			return true
		})
		return
	}

	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if fd.IsList() {
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				vv := list.Get(i)
				DropUnknownFields(vv.Message())
			}
		} else {
			DropUnknownFields(v.Message())
		}
	default:
	}
}
