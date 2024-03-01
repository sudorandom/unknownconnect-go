// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: internal/proto/new/user.proto

package newconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	new1 "github.com/sudorandom/unknownconnect-go/internal/proto/new"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// UserManagementName is the fully-qualified name of the UserManagement service.
	UserManagementName = "helloworld.new.UserManagement"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// UserManagementNewUserProcedure is the fully-qualified name of the UserManagement's NewUser RPC.
	UserManagementNewUserProcedure = "/helloworld.new.UserManagement/NewUser"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	userManagementServiceDescriptor       = new1.File_internal_proto_new_user_proto.Services().ByName("UserManagement")
	userManagementNewUserMethodDescriptor = userManagementServiceDescriptor.Methods().ByName("NewUser")
)

// UserManagementClient is a client for the helloworld.new.UserManagement service.
type UserManagementClient interface {
	NewUser(context.Context, *connect.Request[new1.NewUserRequest]) (*connect.Response[new1.NewUserResponse], error)
}

// NewUserManagementClient constructs a client for the helloworld.new.UserManagement service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewUserManagementClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) UserManagementClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &userManagementClient{
		newUser: connect.NewClient[new1.NewUserRequest, new1.NewUserResponse](
			httpClient,
			baseURL+UserManagementNewUserProcedure,
			connect.WithSchema(userManagementNewUserMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// userManagementClient implements UserManagementClient.
type userManagementClient struct {
	newUser *connect.Client[new1.NewUserRequest, new1.NewUserResponse]
}

// NewUser calls helloworld.new.UserManagement.NewUser.
func (c *userManagementClient) NewUser(ctx context.Context, req *connect.Request[new1.NewUserRequest]) (*connect.Response[new1.NewUserResponse], error) {
	return c.newUser.CallUnary(ctx, req)
}

// UserManagementHandler is an implementation of the helloworld.new.UserManagement service.
type UserManagementHandler interface {
	NewUser(context.Context, *connect.Request[new1.NewUserRequest]) (*connect.Response[new1.NewUserResponse], error)
}

// NewUserManagementHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewUserManagementHandler(svc UserManagementHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	userManagementNewUserHandler := connect.NewUnaryHandler(
		UserManagementNewUserProcedure,
		svc.NewUser,
		connect.WithSchema(userManagementNewUserMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/helloworld.new.UserManagement/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case UserManagementNewUserProcedure:
			userManagementNewUserHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedUserManagementHandler returns CodeUnimplemented from all methods.
type UnimplementedUserManagementHandler struct{}

func (UnimplementedUserManagementHandler) NewUser(context.Context, *connect.Request[new1.NewUserRequest]) (*connect.Response[new1.NewUserResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("helloworld.new.UserManagement.NewUser is not implemented"))
}
