package grpcmetrics

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestUnaryServerInterceptor(t *testing.T) {
	m := newServerMetrics()
	if _, err := UnaryServerInterceptor(m)(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}, func(
		context.Context, interface{},
	) (interface{}, error) {
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	m.WritePrometheus(&b)
	checkContains(t, &b,
		`grpc_server_started_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
		`grpc_server_handled_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check",grpc_code="OK"} 1`,
	)
}

func TestStreamServerInterceptor(t *testing.T) {
	m := newServerMetrics()
	if err := StreamServerInterceptor(m)(nil, nil, &grpc.StreamServerInfo{
		FullMethod:     "/grpc.health.v1.Health/Watch",
		IsServerStream: true,
	}, func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	m.WritePrometheus(&b)
	checkContains(t, &b,
		`grpc_server_started_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
		`grpc_server_handled_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch",grpc_code="OK"} 1`,
	)
}

func BenchmarkUnaryServerInterceptor(b *testing.B) {

}

func BenchmarkUnaryServerInterceptor_Prom(b *testing.B) {

}

func checkContains(t *testing.T, s fmt.Stringer, what ...string) {
	t.Helper()
	for i := range what {
		if !strings.Contains(s.String(), what[i]) {
			t.Fatalf("output doesn't contain: %s", what[i])
		}
	}
}

func newServerMetrics() *ServerMetrics {
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	m := NewServerMetrics(WithServerHandlingTimeHistogram(true))
	m.Initialize(s)
	return m
}
