# grpcmetrics

Prometheus metrics for gGPC servers and clients via [VictoriaMetrics](https://github.com/VictoriaMetrics/metrics).

Drop-in replacement for [go-grpc-prometheus](https://github.com/grpc-ecosystem/go-grpc-prometheus).

## Usage

### Server

```go
import (
	"google.golang.org/grpc"
	"github.com/amenzhinsky/grpcmetrics"
)

m := grpcmetrics.NewServerMetrics()
s := grpc.NewServer(
	grpc.ChainUnaryInterceptor(
		grpcmetrics.UnaryServerInterceptor(m),
		// other interceptors...
	),
	grpc.ChainStreamInterceptor(
		grpcmetrics.StreamServerInterceptor(m),
		// other interceptors...
	),
)

// register grpc services
grpc_health_v1.RegisterHealthServer(s, health.NewServer())

// optionally pre-populate metrics with services and methods registered by the server
m.InitializeMetrics(s)

http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
	metrics.WritePrometheus(w, true)
})
```

### Client

```go
m := grpcmetrics.NewClientMetrics()
c, err := grpc.Dial("",
	grpc.WithChainUnaryInterceptor(
		grpcmetrics.UnaryClientInterceptor(m),
		// other interceptors...
	),
	grpc.WithChainStreamInterceptor(
		grpcmetrics.UnaryClientInterceptor(m)),
		// other interceptors...
	),
)
if err != nil {
	return err
}

http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
	metrics.WritePrometheus(w, true)
})
```
