package unknownconnect_test

import (
	"bytes"
	"fmt"
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
)

func TestOutdatedClient(t *testing.T) {
	var calledCount int
	interceptor := unknownconnect.NewInterceptor(func(proto.Message) {
		calledCount++
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
	fmt.Println("req", req.URL.Path)
	c.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}
