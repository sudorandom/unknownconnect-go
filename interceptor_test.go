package unknownconnect_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudorandom/unknownconnect-go"
	"github.com/sudorandom/unknownconnect-go/internal/proto/new"
	"github.com/sudorandom/unknownconnect-go/internal/proto/old/oldconnect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protopack"
)

func TestDropUnknownFields(t *testing.T) {
	interceptor := unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
		msg.ProtoReflect().SetUnknown(nil)
		return nil
	})

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

func TestOutdatedClient(t *testing.T) {
	var calledCount int
	interceptor := unknownconnect.NewInterceptor(func(context.Context, connect.Spec, proto.Message) error {
		calledCount++
		return nil
	})
	_, handler := oldconnect.NewUserManagementHandler(
		oldconnect.UnimplementedUserManagementHandler{},
		connect.WithInterceptors(interceptor),
	)

	localClient := NewLocalClient(handler)

	body, err := proto.Marshal(&new.NewUserRequest{
		User: &new.User{
			Name:  "bob",
			Email: "bob@example.com",
		},
	})
	require.NoError(t, err)

	req, err := http.NewRequest("POST", oldconnect.UserManagementNewUserProcedure, bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Add("Content-Type", "application/proto")

	resp, err := localClient.Do(req)
	assert.NotNil(t, resp)
	assert.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(respBody), "is not implemented")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, 1, calledCount)
}

func TestMessageHasUnknownFields(t *testing.T) {
	t.Run("with unknown field", func(t *testing.T) {
		user := &new.User{
			Name:  "bob",
			Email: "bob@example.com",
		}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		assert.True(t, unknownconnect.MessageHasUnknownFields(user.ProtoReflect()))
	})
	t.Run("without unknown field", func(t *testing.T) {
		user := &new.User{
			Name:  "bob",
			Email: "bob@example.com",
		}
		assert.False(t, unknownconnect.MessageHasUnknownFields(user.ProtoReflect()))
	})
	t.Run("with nested unknown field", func(t *testing.T) {
		user := &new.User{
			Name:  "bob",
			Email: "bob@example.com",
		}
		unknownField := protopack.Message{protopack.Tag{Number: 3, Type: protopack.Fixed32Type}, protopack.Int32(42)}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields(unknownField.Marshal()))
		req := &new.NewUserRequest{
			User: user,
		}
		assert.True(t, unknownconnect.MessageHasUnknownFields(req.ProtoReflect()))
	})
}

type LocalClient struct {
	handler http.Handler
}

func NewLocalClient(handler http.Handler) *LocalClient {
	return &LocalClient{
		handler: handler,
	}
}

func (c LocalClient) Do(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}
