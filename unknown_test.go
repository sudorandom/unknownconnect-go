package unknownconnect_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudorandom/unknownconnect-go"
	"github.com/sudorandom/unknownconnect-go/internal/proto/new"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protopack"
)

func TestMessageHasUnknownFields(t *testing.T) {
	t.Run("with unknown field", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		assert.True(t, unknownconnect.MessageHasUnknownFields(user.ProtoReflect()))
	})
	t.Run("without unknown field", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		assert.False(t, unknownconnect.MessageHasUnknownFields(user.ProtoReflect()))
	})
	t.Run("with nested unknown field", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		unknownField := protopack.Message{protopack.Tag{Number: 300, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields(unknownField.Marshal()))
		req := &new.NewUserRequest{
			User: user,
		}
		assert.True(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested primative map", func(t *testing.T) {
		req := &new.NewUserRequest{
			PrimativeMap: map[int32]int32{1: 2, 3: 4},
		}
		assert.False(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested msg map", func(t *testing.T) {
		req := &new.NewUserRequest{
			MsgMap: map[int32]*new.User{
				1: {Name: "bob1", Email: "bob@example.com"},
				2: {Name: "bob2", Email: "bob@example.com"},
			},
		}
		assert.False(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested msg map with unknown", func(t *testing.T) {
		user1 := &new.User{Name: "bob1", Email: "bob@example.com"}
		unknownField := protopack.Message{protopack.Tag{Number: 300, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		user1.ProtoReflect().SetUnknown(protoreflect.RawFields(unknownField.Marshal()))
		req := &new.NewUserRequest{
			MsgMap: map[int32]*new.User{1: user1},
		}
		assert.True(t, unknownconnect.MessageHasUnknownFields(user1.ProtoReflect()))
		assert.True(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested primative list", func(t *testing.T) {
		req := &new.NewUserRequest{
			PrimativeList: []int32{1: 2, 3: 4},
		}
		assert.False(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested msg list", func(t *testing.T) {
		req := &new.NewUserRequest{
			MsgList: []*new.User{
				{Name: "bob1", Email: "bob@example.com"},
				{Name: "bob2", Email: "bob@example.com"},
			},
		}
		assert.False(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
	t.Run("with nested msg list with unknown", func(t *testing.T) {
		user1 := &new.User{Name: "bob1", Email: "bob@example.com"}
		unknownField := protopack.Message{protopack.Tag{Number: 300, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		user1.ProtoReflect().SetUnknown(protoreflect.RawFields(unknownField.Marshal()))
		req := &new.NewUserRequest{MsgList: []*new.User{1: user1}}
		assert.True(t, unknownconnect.MessageHasUnknownFields(user1.ProtoReflect()))
		assert.True(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
}

func TestDropUnknownFields(t *testing.T) {
	interceptor := unknownconnect.NewInterceptor(
		unknownconnect.WithCallback(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
			msg.ProtoReflect().SetUnknown(nil)
			return nil
		}))

	{
		req := &new.NewUserRequest{
			User: &new.User{
				Name:  "bob",
				Email: "bob@example.com",
			},
		}
		unknownField := protopack.Message{protopack.Tag{Number: 3, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		req.ProtoReflect().SetUnknown(unknownField.Marshal())
		wrapped := interceptor.WrapUnary(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			msg := req.Any().(proto.Message)
			assert.Empty(t, msg.ProtoReflect().GetUnknown())
			return connect.NewResponse(&new.NewUserResponse{}), nil
		})
		_, err := wrapped(context.Background(), connect.NewRequest(req))
		require.NoError(t, err)
	}
	{
		req := &new.NewUserRequest{
			User: &new.User{
				Name:  "bob",
				Email: "bob@example.com",
			},
		}
		unknownField := protopack.Message{protopack.Tag{Number: 3, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		req.ProtoReflect().SetUnknown(unknownField.Marshal())
		unary := func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			msg := req.Any().(proto.Message)
			assert.NotEmpty(t, msg.ProtoReflect().GetUnknown())
			return connect.NewResponse(&new.NewUserResponse{}), nil
		}
		_, err := unary(context.Background(), connect.NewRequest(req))
		require.NoError(t, err)
	}
}
