package grpcmetrics

import (
	"context"
	"testing"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryClientInterceptor(t *testing.T) {
	m := NewClientMetrics(WithClientHandlingTimeHistogram(true))
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
}

func TestStreamClientInterceptor(t *testing.T) {
	m := NewClientMetrics(WithClientHandlingTimeHistogram(true))
	stream, err := StreamClientInterceptor(m)(
		context.Background(), &grpc.StreamDesc{
			ServerStreams: true,
		}, nil, "/grpc.health.v1.Health/Watch",
		func(
			ctx context.Context, desc *grpc.StreamDesc,
			cc *grpc.ClientConn, method string,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			return &fakeClientStream{}, nil
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
}

func BenchmarkUnaryClientInterceptor(b *testing.B) {
	benchUnaryClientInterceptor(b, UnaryClientInterceptor(NewClientMetrics()))
}

func BenchmarkUnaryClientInterceptor_client_golang(b *testing.B) {
	h := grpc_prometheus.NewClientMetrics()
	benchUnaryClientInterceptor(b, h.UnaryClientInterceptor())
}

func BenchmarkStreamClientInterceptor(b *testing.B) {
	benchStreamClientInterceptor(b, StreamClientInterceptor(NewClientMetrics()))
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

type fakeClientStream struct{}

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
	return nil
}

func (f *fakeClientStream) RecvMsg(m interface{}) error {
	return nil
}
