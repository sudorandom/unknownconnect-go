# unknownconnect-go
[![Go](https://github.com/sudorandom/unknownconnect-go/actions/workflows/go.yml/badge.svg)](https://github.com/sudorandom/unknownconnect-go/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/sudorandom/unknownconnect-go)](https://goreportcard.com/report/github.com/sudorandom/unknownconnect-go) [![Go Reference](https://pkg.go.dev/badge/github.com/sudorandom/unknownconnect-go.svg)](https://pkg.go.dev/github.com/sudorandom/unknownconnect-go)

unknownconnect-go has an interceptor for [ConnectRPC](https://connectrpc.com/) clients and servers that tells you if you are receiving protobuf messages with unknown fields. This is useful to know when you should upgrade your gRPC clients or servers to the latest version.

```bash
go get -u github.com/sudorandom/unknownconnect-go
```

## Server Examples
Short example:
```golang
import (
    unknownconnect "github.com/sudorandom/unknownconnect-go"
)

...
unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
    slog.Warn("received a protobuf message with unknown fields", slog.Any("spec", spec), slog.Any("msg", msg))
    return nil
})
```

Full:
```golang
import (
    "log/slog"

    "connectrpc.com/connect"
    unknownconnect "github.com/sudorandom/unknownconnect-go"
)

func main() {
    greeter := &GreetServer{}
    mux := http.NewServeMux()
    path, handler := greetv1connect.NewGreetServiceHandler(greeter, connect.WithInterceptors(
        unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
            return connect.NewError(connect.InvalidArgument, err)
        }),
    ))
    mux.Handle(path, handler)
    http.ListenAndServe("localhost:8080", h2c.NewHandler(mux, &http2.Server{}))
}
```

The first example simply emits a warning log and the second example will fail the request if the server receives a message with unknown fields. You can decide what to do. Here are some ideas:

- Add to a metric that counts how often this happens
- Fail the request/response; maybe the most useful in non-production integration environments
- Emit a log
- Add an annotation to the context to be used in the handler
- ???

## Client Examples
And it works the same for clients, too:

```golang
package main

import (
    "context"
    "log"
    "net/http"

    greetv1 "example/gen/greet/v1"
    "example/gen/greet/v1/greetv1connect"

    "connectrpc.com/connect"
)

func main() {
    client := greetv1connect.NewGreetServiceClient(
        http.DefaultClient,
        "http://localhost:8080",
        connect.WithInterceptors(
            unknownconnect.NewInterceptor(func(ctx context.Context, spec connect.Spec, msg proto.Message) error {
                slog.Warn("received a protobuf message with unknown fields", slog.Any("spec", spec), slog.Any("msg", msg))
                return nil
            })
        ),
    )
    res, err := client.Greet(
        context.Background(),
        connect.NewRequest(&greetv1.GreetRequest{Name: "Jane"}),
    )
    if err != nil {
        log.Println(err)
        return
    }
    log.Println(res.Msg.Greeting)
}
```

## Why?
gRPC systems can be quite complex. When making additions to protobuf files the server or the client often gets updated at different times. In a perfect world, this would all be synchronized. But we live in reality. Sometimes release schedules differ between components. Sometimes you just forget to update a component. Many times you might be consuming a gRPC service managed by another team and *they don't tell you that they're changing things*. I believe this interceptor helps with all of these cases. It allows you to raise the issue before it becomes a problem.
