package grpcmetrics

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/VictoriaMetrics/metrics"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerInterceptor(t *testing.T) {
	m := newServerMetrics(
		WithServerHandlingTimeHistogram(),
	)
	if _, err := UnaryServerInterceptor(m)(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}, func(
		context.Context, interface{},
	) (interface{}, error) {
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	checkContains(t, m.s,
		`grpc_server_started_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
		`grpc_server_handled_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check",grpc_code="OK"} 1`,
		`grpc_server_msg_received_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
		`grpc_server_msg_sent_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
	)
}

func TestStreamServerInterceptor(t *testing.T) {
	m := newServerMetrics(
		WithServerHandlingTimeHistogram(),
	)
	if err := StreamServerInterceptor(m)(nil, &fakeServerStream{}, &grpc.StreamServerInfo{
		FullMethod:     "/grpc.health.v1.Health/Watch",
		IsServerStream: true,
	}, func(srv interface{}, stream grpc.ServerStream) error {
		if err := stream.SendMsg(&grpc_health_v1.HealthCheckRequest{}); err != nil {
			return err
		}
		var res grpc_health_v1.HealthCheckResponse
		return stream.RecvMsg(&res)
	}); err != nil {
		t.Fatal(err)
	}
	checkContains(t, m.s,
		`grpc_server_started_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
		`grpc_server_handled_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch",grpc_code="OK"} 1`,
		`grpc_server_msg_received_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
		`grpc_server_msg_sent_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
	)
}

func TestServerMetrics_InitializeMetrics(t *testing.T) {
	m := newServerMetrics(
		WithServerHandlingTimeHistogram(),
	)
	m.InitializeMetrics(newServer())
	checkContains(t, m.s,
		`grpc_server_started_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 0`,
		`grpc_server_handled_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check",grpc_code="OK"} 0`,
		`grpc_server_msg_received_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 0`,
		`grpc_server_msg_sent_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 0`,
		`grpc_server_started_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 0`,
		`grpc_server_handled_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch",grpc_code="OK"} 0`,
		`grpc_server_msg_received_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 0`,
		`grpc_server_msg_sent_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 0`,
	)
}

func BenchmarkServerInterceptorScrape(b *testing.B) {
	m := newServerMetrics()
	h := func(w http.ResponseWriter, r *http.Request) {
		m.s.WritePrometheus(w)
	}
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h(httptest.NewRecorder(), r)
	}
}

func BenchmarkServerInterceptorScrape_client_golang(b *testing.B) {
	m := newServerMetrics_client_golang()
	reg := prometheus.NewRegistry()
	reg.MustRegister(m)
	h := promhttp.InstrumentMetricHandler(reg,
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	)
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
}

func BenchmarkUnaryServerInterceptor(b *testing.B) {
	benchUnaryServerInterceptor(b, UnaryServerInterceptor(newServerMetrics()))
}

func BenchmarkUnaryServerInterceptor_client_golang(b *testing.B) {
	h := newServerMetrics_client_golang()
	benchUnaryServerInterceptor(b, h.UnaryServerInterceptor())
}

func BenchmarkStreamServerInterceptor(b *testing.B) {
	benchStreamServerInterceptor(b, StreamServerInterceptor(newServerMetrics()))
}

func BenchmarkStreamServerInterceptor_client_golang(b *testing.B) {
	h := newServerMetrics_client_golang()
	benchStreamServerInterceptor(b, h.StreamServerInterceptor())
}

func benchUnaryServerInterceptor(b *testing.B, h grpc.UnaryServerInterceptor) {
	i := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := h(context.Background(), nil, i, func(context.Context, interface{}) (interface{}, error) {
				return nil, nil
			}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchStreamServerInterceptor(b *testing.B, h grpc.StreamServerInterceptor) {
	i := &grpc.StreamServerInfo{
		FullMethod:     "/grpc.health.v1.Health/Watch",
		IsServerStream: true,
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := h(nil, nil, i, func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func checkContains(t *testing.T, s *metrics.Set, what ...string) {
	t.Helper()
	var b bytes.Buffer
	s.WritePrometheus(&b)
	for i := range what {
		if !strings.Contains(b.String(), what[i]) {
			t.Fatalf("output doesn't contain: %s\n%s", what[i], b.String())
		}
	}
}

func newServer() *grpc.Server {
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	return s
}

func newServerMetrics(opts ...ServerOption) *ServerMetrics {
	m := NewServerMetrics(append([]ServerOption{
		WithServerMetricsSet(metrics.NewSet()),
	}, opts...)...)
	return m
}

func newServerMetrics_client_golang() *grpc_prometheus.ServerMetrics {
	m := grpc_prometheus.NewServerMetrics()
	m.InitializeMetrics(newServer())
	return m
}

type fakeServerStream struct{}

func (s *fakeServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (s *fakeServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (s *fakeServerStream) SetTrailer(md metadata.MD) {}

func (s *fakeServerStream) Context() context.Context {
	return context.Background()
}

func (s *fakeServerStream) SendMsg(m interface{}) error {
	return nil
}

func (s *fakeServerStream) RecvMsg(m interface{}) error {
	return nil
}
