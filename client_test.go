package grpcmetrics

import (
	"context"
	"io"
	"testing"

	"github.com/VictoriaMetrics/metrics"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryClientInterceptor(t *testing.T) {
	m := NewClientMetrics(
		WithClientMetricsSet(metrics.NewSet()),
		WithClientHandlingTimeHistogram(),
	)
	if err := UnaryClientInterceptor(m)(
		context.Background(), "/grpc.health.v1.Health/Check", nil, nil, nil,
		func(
			ctx context.Context, method string,
			req, reply interface{}, cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}

	checkContains(t, m.s,
		`grpc_client_handled_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check",grpc_code="OK"} 1`,
		`grpc_client_msg_received_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
		`grpc_client_msg_sent_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check",grpc_code="OK"} 1`,
		`grpc_client_started_total{grpc_type="unary",grpc_service="/grpc.health.v1.Health",grpc_method="Check"} 1`,
	)
}

func TestStreamClientInterceptor(t *testing.T) {
	m := NewClientMetrics(
		WithClientMetricsSet(metrics.NewSet()),
		WithClientHandlingTimeHistogram(),
	)
	fake := &fakeClientStream{}
	stream, err := StreamClientInterceptor(m)(
		context.Background(), &grpc.StreamDesc{
			ServerStreams: true,
		}, nil, "/grpc.health.v1.Health/Watch",
		func(
			ctx context.Context, desc *grpc.StreamDesc,
			cc *grpc.ClientConn, method string,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			return fake, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.SendMsg(nil); err != nil {
		t.Fatal(err)
	}
	if err := stream.RecvMsg(nil); err != nil {
		t.Fatal(err)
	}

	fake.err = io.EOF
	if err := stream.RecvMsg(nil); err != io.EOF {
		t.Fatalf("err = %v, want %v", err, io.EOF)
	}

	checkContains(t, m.s,
		`grpc_client_handled_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch",grpc_code="OK"} 1`,
		`grpc_client_msg_received_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
		`grpc_client_msg_sent_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
		`grpc_client_started_total{grpc_type="server_stream",grpc_service="/grpc.health.v1.Health",grpc_method="Watch"} 1`,
	)
}

func BenchmarkUnaryClientInterceptor(b *testing.B) {
	benchUnaryClientInterceptor(b, UnaryClientInterceptor(NewClientMetrics(
		WithClientMetricsSet(metrics.NewSet()),
	)))
}

func BenchmarkUnaryClientInterceptor_client_golang(b *testing.B) {
	h := grpc_prometheus.NewClientMetrics()
	benchUnaryClientInterceptor(b, h.UnaryClientInterceptor())
}

func BenchmarkStreamClientInterceptor(b *testing.B) {
	benchStreamClientInterceptor(b, StreamClientInterceptor(NewClientMetrics(
		WithClientMetricsSet(metrics.NewSet()),
	)))
}

func BenchmarkStreamClientInterceptor_client_golang(b *testing.B) {
	h := grpc_prometheus.NewClientMetrics()
	benchStreamClientInterceptor(b, h.StreamClientInterceptor())
}

func benchUnaryClientInterceptor(b *testing.B, h grpc.UnaryClientInterceptor) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := h(
				context.Background(), "/grpc.health.v1.Health/Check", nil, nil, nil,
				func(
					ctx context.Context, method string,
					req, reply interface{}, cc *grpc.ClientConn,
					opts ...grpc.CallOption,
				) error {
					return nil
				},
			); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchStreamClientInterceptor(b *testing.B, h grpc.StreamClientInterceptor) {
	i := &grpc.StreamDesc{
		ServerStreams: true,
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stream, err := h(
				context.Background(), i, nil, "/grpc.health.v1.Health/Watch",
				func(
					ctx context.Context, desc *grpc.StreamDesc,
					cc *grpc.ClientConn, method string,
					opts ...grpc.CallOption,
				) (grpc.ClientStream, error) {
					return &fakeClientStream{}, nil
				},
			)
			if err != nil {
				b.Fatal(err)
			}
			if err := stream.SendMsg(nil); err != nil {
				b.Fatal(err)
			}
			if err := stream.RecvMsg(nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}

type fakeClientStream struct {
	err error
}

func (f *fakeClientStream) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (f *fakeClientStream) Trailer() metadata.MD {
	return metadata.MD{}
}

func (f *fakeClientStream) CloseSend() error {
	return nil
}

func (f *fakeClientStream) Context() context.Context {
	return context.Background()
}

func (f *fakeClientStream) SendMsg(m interface{}) error {
	return f.err
}

func (f *fakeClientStream) RecvMsg(m interface{}) error {
	return f.err
}
