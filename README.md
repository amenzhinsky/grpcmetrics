# grpcmetrics

## Usage

### Server

```go
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
m.Initialize(s)

http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
	m.WritePrometheus(w)
	metrics.WritePrometheus(w, true)
})
```

### Client

## Benchmarks

