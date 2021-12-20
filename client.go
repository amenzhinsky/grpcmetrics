package grpcmetrics

import (
	"context"
	"io"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	metricClientStarted     = "grpc_client_started_total"
	metricClientHandled     = "grpc_client_handled_total"
	metricClientMsgReceived = "grpc_client_msg_received_total"
	metricClientMsgSent     = "grpc_client_msg_sent_total"
	// TODO: metricClientHandling    = "grpc_client_handling_seconds"
)

func UnaryClientInterceptor(m *ClientMetrics) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		fullMethod string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		typ, service, method := keys(fullMethod, false, false)
		m.counter(metricClientStarted, typ, service, method, noCode).Inc()
		m.counter(metricClientMsgReceived, typ, service, method, noCode).Inc()
		err := invoker(ctx, method, req, reply, cc, opts...)
		m.counter(metricClientHandled, typ, service, method, status.Code(err)).Inc()
		if err == nil {
			m.counter(metricClientMsgSent, typ, service, method, status.Code(err)).Inc()
		}
		return err
	}
}

func StreamClientInterceptor(m *ClientMetrics) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		fullMethod string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		typ, service, method := keys(fullMethod, desc.ServerStreams, desc.ClientStreams)
		m.counter(metricClientStarted, typ, service, method, noCode).Inc()
		cs, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			m.counter(metricClientHandled, typ, service, method, status.Code(err)).Inc()
			return nil, err
		}
		return &clientStream{
			cs,
			m.counter(metricClientMsgSent, typ, service, method, noCode),
			m.counter(metricClientMsgReceived, typ, service, method, noCode),
			m,
			typ, service, method,
		}, err
	}
}

type clientStream struct {
	grpc.ClientStream
	send *metrics.Counter
	recv *metrics.Counter

	m                    *ClientMetrics
	typ, service, method string
}

func (s *clientStream) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err == nil {
		s.send.Inc()
	}
	return err
}

func (s *clientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	switch err {
	case nil:
		s.recv.Inc()
	case io.EOF:
		s.m.counter(metricClientHandled, s.typ, s.service, s.method, codes.OK).Inc()
	default:
		s.m.counter(metricClientHandled, s.typ, s.service, s.method, status.Code(err)).Inc()
	}
	return err
}

type ClientOption func(m *ClientMetrics)

func WithClientHandlingTimeHistogram(enabled bool) ClientOption {
	return func(m *ClientMetrics) {
		m.handlingHistogram = enabled
	}
}

func WithClientMetricsSet(set *metrics.Set) ClientOption {
	return func(m *ClientMetrics) {
		m.set.s = set
	}
}

func NewClientMetrics(opts ...ClientOption) *ClientMetrics {
	m := &ClientMetrics{set: newSet()}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type ClientMetrics struct {
	set
	handlingHistogram bool
}
