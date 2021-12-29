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

### Benchmarks

`benchcmp` against `client_golang`.

```
benchmark                               old ns/op     new ns/op     delta
BenchmarkUnaryClientInterceptor-12      139           85.9          -38.31%
BenchmarkStreamClientInterceptor-12     156           90.0          -42.15%
BenchmarkServerScrape-12                215109        1681          -99.22%
BenchmarkUnaryServerInterceptor-12      184           86.7          -52.94%
BenchmarkStreamServerInterceptor-12     129           83.0          -35.83%

benchmark                               old allocs     new allocs     delta
BenchmarkUnaryClientInterceptor-12      4              0              -100.00%
BenchmarkStreamClientInterceptor-12     6              2              -66.67%
BenchmarkServerScrape-12                267            9              -96.63%
BenchmarkUnaryServerInterceptor-12      5              0              -100.00%
BenchmarkStreamServerInterceptor-12     4              1              -75.00%

benchmark                               old bytes     new bytes     delta
BenchmarkUnaryClientInterceptor-12      240           0             -100.00%
BenchmarkStreamClientInterceptor-12     264           128           -51.52%
BenchmarkServerScrape-12                60630         992           -98.36%
BenchmarkUnaryServerInterceptor-12      288           0             -100.00%
BenchmarkStreamServerInterceptor-12     216           32            -85.19%
```
