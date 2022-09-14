# grpcmetrics

Prometheus metrics for gGPC servers and clients via [VictoriaMetrics](https://github.com/VictoriaMetrics/metrics).

Drop-in replacement for [go-grpc-prometheus](https://github.com/grpc-ecosystem/go-grpc-prometheus).

## Usage

### Server

```go
import (
	"net/http"
	
	"google.golang.org/grpc"
	"github.com/amenzhinsky/grpcmetrics"
	"github.com/VictoriaMetrics/metrics"
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
import (
	"net/http"

	"google.golang.org/grpc"
	"github.com/amenzhinsky/grpcmetrics"
	"github.com/VictoriaMetrics/metrics"
)

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

Benchmarks against [client_golang](github.com/grpc-ecosystem/go-grpc-prometheus) interceptors (MacBook Air M1).

```
go test -run=none -bench=_client_golang$ -benchmem -benchtime=5s | sed s/_client_golang//g > old.txt
go test -run=none -bench=_metrics$ -benchmem -benchtime=5s | sed s/_metrics//g > new.txt
benchcmp old.txt new.txt

benchmark                              old ns/op     new ns/op     delta
BenchmarkScrapeClient-8                9191          553           -93.98%
BenchmarkUnaryClientInterceptor-8      358           235           -34.44%
BenchmarkStreamClientInterceptor-8     276           197           -28.58%
BenchmarkScrapeServer-8                41107         542           -98.68%
BenchmarkUnaryServerInterceptor-8      310           209           -32.72%
BenchmarkStreamServerInterceptor-8     331           244           -26.40%

benchmark                              old allocs     new allocs     delta
BenchmarkScrapeClient-8                65             9              -86.15%
BenchmarkUnaryClientInterceptor-8      5              0              -100.00%
BenchmarkStreamClientInterceptor-8     6              2              -66.67%
BenchmarkScrapeServer-8                267            9              -96.63%
BenchmarkUnaryServerInterceptor-8      5              0              -100.00%
BenchmarkStreamServerInterceptor-8     7              2              -71.43%

benchmark                              old bytes     new bytes     delta
BenchmarkScrapeClient-8                36861         992           -97.31%
BenchmarkUnaryClientInterceptor-8      288           0             -100.00%
BenchmarkStreamClientInterceptor-8     264           96            -63.64%
BenchmarkScrapeServer-8                59324         992           -98.33%
BenchmarkUnaryServerInterceptor-8      288           0             -100.00%
BenchmarkStreamServerInterceptor-8     328           80            -75.61%
```
