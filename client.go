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

func WithClientHandlingTimeHistogram(enable bool) ClientOption {
	return func(m *ClientMetrics) {
		if enable {
			m.handling = newHistogram("grpc_client_handling_seconds")
		}
	}
}

func WithClientMetricsSet(s *metrics.Set) ClientOption {
	return func(m *ClientMetrics) {
		m.s = &set{s}
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
	s        *set
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
		var started time.Time
		if m.handling != nil {
			started = time.Now()
		}
		typ := streamType(desc.ServerStreams, desc.ClientStreams)
		m.started.with(m.s, typ, fullMethod, noCode).Inc()
		cs, err := streamer(ctx, desc, cc, fullMethod, opts...)
		if err != nil {
			m.handled.with(m.s, typ, fullMethod, status.Code(err)).Inc()
			return nil, err
		}
		return &clientStream{
			cs,
			m,
			typ, fullMethod,
			started,
		}, err
	}
}

type clientStream struct {
	grpc.ClientStream

	m           *ClientMetrics
	typ, method string
	startedAt   time.Time
}

func (cs *clientStream) SendMsg(m interface{}) error {
	err := cs.ClientStream.SendMsg(m)
	if err == nil {
		cs.m.msgSent.with(cs.m.s, cs.typ, cs.method, noCode).Inc()
	}
	return err
}

func (cs *clientStream) RecvMsg(m interface{}) error {
	err := cs.ClientStream.RecvMsg(m)
	if err == nil {
		cs.m.msgRecv.with(cs.m.s, cs.typ, cs.method, noCode).Inc()
		return nil
	}
	code := codes.OK
	if err != io.EOF {
		code = status.Code(err)
	}
	cs.m.handled.with(cs.m.s, cs.typ, cs.method, code).Inc()
	if cs.m.handling != nil {
		cs.m.handling.with(cs.m.s, cs.typ, cs.method).UpdateDuration(cs.startedAt)
	}
	return err
}
