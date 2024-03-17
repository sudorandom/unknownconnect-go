package unknownconnect_test

import (
	"bytes"
	"context"
	"errors"
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
)

func TestOutdatedClient(t *testing.T) {
	var calledCount int
	interceptor := unknownconnect.NewInterceptor(
		unknownconnect.WithCallback(func(context.Context, connect.Spec, proto.Message) error {
			calledCount++
			return nil
		}))
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

func TestInterceptor(t *testing.T) {
	t.Run("with unknown field", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		var called bool
		interceptor := unknownconnect.NewInterceptor(unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
			called = true
			proto.Equal(m, user)
			assert.Len(t, m.ProtoReflect().GetUnknown(), 3)
			return nil
		}))
		unary := interceptor.WrapUnary(connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			msg := req.Any().(proto.Message)
			assert.Len(t, msg.ProtoReflect().GetUnknown(), 3)
			return nil, nil
		}))
		resp, err := unary(context.Background(), connect.NewRequest(user))
		assert.NoError(t, err)
		assert.Nil(t, resp)
		assert.True(t, called)
	})
	t.Run("without unknown field", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		var called bool
		interceptor := unknownconnect.NewInterceptor(unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
			called = true
			proto.Equal(m, user)
			return nil
		}))
		unary := interceptor.WrapUnary(connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, nil
		}))
		resp, err := unary(context.Background(), connect.NewRequest(user))
		assert.NoError(t, err)
		assert.Nil(t, resp)
		assert.False(t, called)
	})
	t.Run("two callbacks", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		var callCount int
		interceptor := unknownconnect.NewInterceptor(
			unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
				assert.Equal(t, 0, callCount)
				proto.Equal(m, user)
				assert.Len(t, m.ProtoReflect().GetUnknown(), 3)
				callCount++
				t.Log("called 1st interceptor")
				return nil
			}),
			unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
				assert.Equal(t, 1, callCount)
				proto.Equal(m, user)
				assert.Len(t, m.ProtoReflect().GetUnknown(), 3)
				callCount++
				t.Log("called 2nd interceptor")
				return nil
			}))
		unary := interceptor.WrapUnary(connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			msg := req.Any().(proto.Message)
			assert.Len(t, msg.ProtoReflect().GetUnknown(), 3)
			return nil, nil
		}))
		resp, err := unary(context.Background(), connect.NewRequest(user))
		assert.NoError(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, 2, callCount)
	})
	t.Run("with unknown field and drop", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		var called bool
		interceptor := unknownconnect.NewInterceptor(
			unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
				called = true
				proto.Equal(m, user)
				assert.Len(t, m.ProtoReflect().GetUnknown(), 3)
				return nil
			}),
			unknownconnect.WithDrop(),
		)
		unary := interceptor.WrapUnary(connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			msg := req.Any().(proto.Message)
			assert.Len(t, msg.ProtoReflect().GetUnknown(), 0)
			return nil, nil
		}))
		resp, err := unary(context.Background(), connect.NewRequest(user))
		assert.NoError(t, err)
		assert.Nil(t, resp)
		assert.True(t, called)
	})
	t.Run("return error", func(t *testing.T) {
		user := &new.User{Name: "bob", Email: "bob@example.com"}
		user.ProtoReflect().SetUnknown(protoreflect.RawFields([]byte{8, 96, 01}))
		var called bool
		interceptor := unknownconnect.NewInterceptor(
			unknownconnect.WithCallback(func(ctx context.Context, s connect.Spec, m proto.Message) error {
				called = true
				proto.Equal(m, user)
				assert.Len(t, m.ProtoReflect().GetUnknown(), 3)
				return errors.New("unknown fields error")
			}),
			unknownconnect.WithDrop(),
		)
		unary := interceptor.WrapUnary(connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			assert.Fail(t, "this should not be called because an error should have been returned earlier")
			return nil, nil
		}))
		resp, err := unary(context.Background(), connect.NewRequest(user))
		assert.ErrorContains(t, err, "unknown fields error")
		assert.Nil(t, resp)
		assert.True(t, called)
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
