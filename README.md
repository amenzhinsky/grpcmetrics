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

`benchcmp` vs `client_golang`.

```
benchmark                                old ns/op     new ns/op     delta
BenchmarkUnaryClientInterceptor_-12      139           94.6          -31.88%
BenchmarkStreamClientInterceptor_-12     164           86.4          -47.37%
BenchmarkServerScrape_-12                206090        1792          -99.13%
BenchmarkUnaryServerInterceptor_-12      198           89.5          -54.72%
BenchmarkStreamServerInterceptor_-12     212           107           -49.29%

benchmark                                old allocs     new allocs     delta
BenchmarkUnaryClientInterceptor_-12      4              0              -100.00%
BenchmarkStreamClientInterceptor_-12     6              2              -66.67%
BenchmarkServerScrape_-12                267            9              -96.63%
BenchmarkUnaryServerInterceptor_-12      5              0              -100.00%
BenchmarkStreamServerInterceptor_-12     7              2              -71.43%

benchmark                                old bytes     new bytes     delta
BenchmarkUnaryClientInterceptor_-12      240           0             -100.00%
BenchmarkStreamClientInterceptor_-12     264           96            -63.64%
BenchmarkServerScrape_-12                60659         992           -98.36%
BenchmarkUnaryServerInterceptor_-12      288           0             -100.00%
BenchmarkStreamServerInterceptor_-12     328           80            -75.61%
```
