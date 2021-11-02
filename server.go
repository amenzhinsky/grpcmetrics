package grpcmetrics

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	metricServerStarted     = "grpc_server_started_total"
	metricServerHandled     = "grpc_server_handled_total"
	metricServerMsgSent     = "grpc_server_msg_sent_total"
	metricServerMsgReceived = "grpc_server_msg_received_total"
	metricServerHandling    = "grpc_server_handling_seconds"
)

func UnaryServerInterceptor(m *ServerMetrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		var started time.Time
		if m.handlingHistogram {
			started = time.Now()
		}
		service, method := serviceAndMethod(info.FullMethod)
		m.counter(metricServerStarted,
			false, false, service, method, noCode,
		).Inc()
		m.counter(metricServerMsgReceived,
			false, false, service, method, noCode,
		).Inc()
		res, err := handler(ctx, req)
		m.counter(metricServerHandled,
			false, false, service, method, status.Code(err),
		).Inc()
		if err == nil {
			m.counter(metricServerMsgSent,
				false, false, service, method, noCode,
			).Inc()
		}
		if m.handlingHistogram {
			m.histogram(metricServerHandling,
				false, false, service, method,
			).UpdateDuration(started)
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
		var started time.Time
		if m.handlingHistogram {
			started = time.Now()
		}
		service, method := serviceAndMethod(info.FullMethod)
		m.counter(metricServerStarted,
			info.IsServerStream, info.IsClientStream, service, method, noCode,
		).Inc()
		err := handler(srv, &serverStream{
			ss,
			m.counter(metricServerMsgSent,
				info.IsServerStream, info.IsClientStream, service, method, noCode,
			),
			m.counter(metricServerMsgReceived,
				info.IsServerStream, info.IsClientStream, service, method, noCode,
			),
		})
		m.counter(metricServerHandled,
			info.IsServerStream, info.IsClientStream, service, method, status.Code(err),
		).Inc()
		if m.handlingHistogram {
			m.histogram(metricServerHandling,
				info.IsServerStream, info.IsClientStream, service, method,
			).UpdateDuration(started)
		}
		return err
	}
}

type serverStream struct {
	grpc.ServerStream
	send *metrics.Counter
	recv *metrics.Counter
}

func (ss *serverStream) SendMsg(m interface{}) error {
	err := ss.ServerStream.SendMsg(m)
	if err == nil {
		ss.send.Inc()
	}
	return err
}

func (ss *serverStream) RecvMsg(m interface{}) error {
	err := ss.ServerStream.RecvMsg(m)
	if err == nil {
		ss.recv.Inc()
	}
	return err
}

type ServerOption func(m *ServerMetrics)

func WithServerHandlingTimeHistogram(enabled bool) ServerOption {
	return func(m *ServerMetrics) {
		m.handlingHistogram = enabled
	}
}

func NewServerMetrics(opts ...ServerOption) *ServerMetrics {
	s := &ServerMetrics{set: newSet()}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ServerMetrics struct {
	set
	handlingHistogram bool
}

func (m *ServerMetrics) Initialize(s *grpc.Server) {
	for service, info := range s.GetServiceInfo() {
		for _, method := range info.Methods {
			_ = m.counter(metricServerStarted,
				method.IsServerStream, method.IsClientStream, service, method.Name, noCode,
			)
			if method.IsServerStream || method.IsClientStream {
				_ = m.counter(metricServerMsgSent,
					method.IsServerStream, method.IsClientStream, service, method.Name, noCode,
				)
				_ = m.counter(metricServerMsgReceived,
					method.IsServerStream, method.IsClientStream, service, method.Name, noCode,
				)
			}
			for _, code := range [...]codes.Code{
				codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument,
				codes.DeadlineExceeded, codes.NotFound, codes.AlreadyExists,
				codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition,
				codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal,
				codes.Unavailable, codes.DataLoss, codes.Unauthenticated,
			} {
				_ = m.counter(metricServerHandled,
					method.IsServerStream, method.IsClientStream, service, method.Name, code,
				)
			}
			if m.handlingHistogram {
				_ = m.histogram(metricServerHandling,
					method.IsServerStream, method.IsClientStream, service, method.Name,
				)
			}
		}
	}
}
