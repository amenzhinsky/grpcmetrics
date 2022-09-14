package grpcmetrics

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const unary = "unary"

type ServerOption func(m *ServerMetrics)

func WithServerHandlingTimeHistogram(enable bool) ServerOption {
	return func(m *ServerMetrics) {
		if enable {
			m.handling = newHistogram("grpc_server_handling_seconds")
		}
	}
}

func WithServerMetricsSet(s *metrics.Set) ServerOption {
	return func(m *ServerMetrics) {
		m.s = &set{s}
	}
}

func NewServerMetrics(opts ...ServerOption) *ServerMetrics {
	s := &ServerMetrics{
		s:       &set{},
		started: newCounter("grpc_server_started_total"),
		handled: newCounter("grpc_server_handled_total"),
		msgSent: newCounter("grpc_server_msg_sent_total"),
		msgRecv: newCounter("grpc_server_msg_received_total"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ServerMetrics struct {
	s        *set
	started  *counter
	handled  *counter
	msgSent  *counter
	msgRecv  *counter
	handling *histogram
}

func (m *ServerMetrics) InitializeMetrics(s *grpc.Server) {
	for service, info := range s.GetServiceInfo() {
		for _, method := range info.Methods {
			typ := streamType(method.IsServerStream, method.IsClientStream)
			fullMethod := "/" + service + "/" + method.Name
			_ = m.started.with(m.s, typ, fullMethod, noCode)
			_ = m.msgSent.with(m.s, typ, fullMethod, noCode)
			_ = m.msgRecv.with(m.s, typ, fullMethod, noCode)
			for _, code := range [...]codes.Code{
				codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument,
				codes.DeadlineExceeded, codes.NotFound, codes.AlreadyExists,
				codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition,
				codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal,
				codes.Unavailable, codes.DataLoss, codes.Unauthenticated,
			} {
				_ = m.handled.with(m.s, typ, fullMethod, code)
			}
			if m.handling != nil {
				_ = m.handling.with(m.s, typ, fullMethod)
			}
		}
	}
}

func UnaryServerInterceptor(m *ServerMetrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		var startedAt time.Time
		if m.handling != nil {
			startedAt = time.Now()
		}
		m.started.with(m.s, unary, info.FullMethod, noCode).Inc()
		m.msgRecv.with(m.s, unary, info.FullMethod, noCode).Inc()
		res, err := handler(ctx, req)
		m.handled.with(m.s, unary, info.FullMethod, status.Code(err)).Inc()
		if err == nil {
			m.msgSent.with(m.s, unary, info.FullMethod, noCode).Inc()
		}
		if m.handling != nil {
			m.handling.with(m.s, unary, info.FullMethod).UpdateDuration(startedAt)
		}
		return res, err
	}
}

func StreamServerInterceptor(m *ServerMetrics) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		var startedAt time.Time
		if m.handling != nil {
			startedAt = time.Now()
		}
		typ := streamType(info.IsServerStream, info.IsClientStream)
		m.started.with(m.s, typ, info.FullMethod, noCode).Inc()
		err := handler(srv, &serverStream{
			ss,
			m, typ, info.FullMethod,
		})
		m.handled.with(m.s, typ, info.FullMethod, status.Code(err)).Inc()
		if m.handling != nil {
			m.handling.with(m.s, typ, info.FullMethod).UpdateDuration(startedAt)
		}
		return err
	}
}

type serverStream struct {
	grpc.ServerStream

	m           *ServerMetrics
	typ, method string
}

func (ss *serverStream) SendMsg(m interface{}) error {
	err := ss.ServerStream.SendMsg(m)
	if err == nil {
		ss.m.msgSent.with(ss.m.s, ss.typ, ss.method, noCode).Inc()
	}
	return err
}

func (ss *serverStream) RecvMsg(m interface{}) error {
	err := ss.ServerStream.RecvMsg(m)
	if err == nil {
		ss.m.msgRecv.with(ss.m.s, ss.typ, ss.method, noCode).Inc()
	}
	return err
}
