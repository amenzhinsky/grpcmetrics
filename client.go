package grpcmetrics

import (
	"context"
	"io"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClientOption func(m *ClientMetrics)

func WithClientHandlingTimeHistogram() ClientOption {
	return func(m *ClientMetrics) {
		m.handling = newHistogram("grpc_client_handling_seconds")
	}
}

func WithClientMetricsSet(set *metrics.Set) ClientOption {
	return func(m *ClientMetrics) {
		m.s = set
	}
}

func NewClientMetrics(opts ...ClientOption) *ClientMetrics {
	m := &ClientMetrics{
		started: newCounter("grpc_client_started_total"),
		handled: newCounter("grpc_client_handled_total"),
		msgRecv: newCounter("grpc_client_msg_received_total"),
		msgSent: newCounter("grpc_client_msg_sent_total"),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type ClientMetrics struct {
	s        *metrics.Set
	started  *counter
	handled  *counter
	msgRecv  *counter
	msgSent  *counter
	handling *histogram
}

func UnaryClientInterceptor(m *ClientMetrics) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		fullMethod string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var started time.Time
		if m.handling != nil {
			started = time.Now()
		}
		m.started.with(m.s, unary, fullMethod, noCode).Inc()
		m.msgRecv.with(m.s, unary, fullMethod, noCode).Inc()
		err := invoker(ctx, fullMethod, req, reply, cc, opts...)
		m.handled.with(m.s, unary, fullMethod, status.Code(err)).Inc()
		if err == nil {
			m.msgSent.with(m.s, unary, fullMethod, status.Code(err)).Inc()
		}
		if m.handling != nil {
			m.handling.with(m.s, unary, fullMethod).UpdateDuration(started)
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
		typ := streamType(desc.ServerStreams, desc.ClientStreams)
		var started time.Time
		var handling *metrics.Histogram
		if m.handling != nil {
			started = time.Now()
			handling = m.handling.with(m.s, typ, fullMethod)
		}
		m.started.with(m.s, typ, fullMethod, noCode).Inc()
		cs, err := streamer(ctx, desc, cc, fullMethod, opts...)
		if err != nil {
			m.handled.with(m.s, typ, fullMethod, status.Code(err)).Inc()
			return nil, err
		}
		return &clientStream{
			cs,
			m.msgSent.with(m.s, typ, fullMethod, noCode),
			m.msgRecv.with(m.s, typ, fullMethod, noCode),
			m,
			typ, fullMethod,
			handling,
			started,
		}, err
	}
}

type clientStream struct {
	grpc.ClientStream
	send *metrics.Counter
	recv *metrics.Counter

	m           *ClientMetrics
	typ, method string
	handling    *metrics.Histogram
	startedAt   time.Time
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
	if err == nil {
		s.recv.Inc()
		return nil
	}
	code := codes.OK
	if err != io.EOF {
		code = status.Code(err)
	}
	s.m.handled.with(s.m.s, s.typ, s.method, code).Inc()
	if s.handling != nil {
		s.handling.UpdateDuration(s.startedAt)
	}
	return err
}
