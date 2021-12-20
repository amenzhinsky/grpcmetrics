package grpcmetrics

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc/codes"
)

func newSet() set {
	return set{
		s:  metrics.NewSet(),
		mc: map[string]map[string]map[string]map[codes.Code]interface{}{},
	}
}

type set struct {
	s  *metrics.Set
	mu sync.RWMutex
	mc map[string]map[string]map[string]map[codes.Code]interface{} // TODO: use metrics.Metric interface
}

func (s *set) WritePrometheus(w io.Writer) {
	if s.s == nil {
		panic("cannot use this method with the default metrics set use metrics.WritePrometheus instead")
	}
	s.s.WritePrometheus(w)
}

func (s *set) counter(
	name string, typ, service, method string, code codes.Code,
) *metrics.Counter {
	return s.metric(name, typ, service, method, code, func(name string) interface{} {
		if s.s == nil {
			return metrics.NewCounter(name)
		}
		return s.s.NewCounter(name)
	}).(*metrics.Counter)
}

func (s *set) histogram(
	name string, typ, service, method string,
) *metrics.Histogram {
	return s.metric(name, typ, service, method, noCode, func(name string) interface{} {
		if s.s == nil {
			return metrics.NewHistogram(name)
		}
		return s.s.NewHistogram(name)
	}).(*metrics.Histogram)
}

const noCode = math.MaxUint32

func (s *set) metric(
	name string, typ, service, method string, code codes.Code,
	fn func(name string) interface{},
) interface{} {
	s.mu.RLock() // try read lock first and promote to write lock if needed
	var locked bool
	defer func() {
		if locked {
			s.mu.Unlock()
		} else {
			s.mu.RUnlock()
		}
	}()

	names, ok := s.mc[name]
	if !ok {
		if s.lock(&locked) || s.mc[name] == nil {
			s.mc[name] = map[string]map[string]map[codes.Code]interface{}{}
		}
		names = s.mc[name]
	}
	services, ok := names[service]
	if !ok {
		if s.lock(&locked) || names[service] == nil {
			names[service] = map[string]map[codes.Code]interface{}{}
		}
		services = names[service]
	}
	methods, ok := services[method]
	if !ok {
		if s.lock(&locked) || services[method] == nil {
			services[method] = map[codes.Code]interface{}{}
		}
		methods = services[method]
	}
	metric, ok := methods[code]
	if !ok {
		if s.lock(&locked) || methods[code] == nil {
			var b strings.Builder
			b.Grow(4096) // should be enough for almost all metric names
			b.WriteString(name)
			b.WriteString(`{grpc_type="`)
			b.WriteString(typ)
			b.WriteString(`",grpc_service="`)
			b.WriteString(service)
			b.WriteString(`",grpc_method="`)
			b.WriteString(method)
			if code < noCode {
				b.WriteString(`",grpc_code="`)
				b.WriteString(code.String())
			}
			b.WriteString(`"}`)
			methods[code] = fn(b.String())
		}
		metric = methods[code]
	}
	return metric
}

func (s *set) lock(locked *bool) bool {
	if *locked {
		return true
	}
	s.mu.RUnlock()
	s.mu.Lock()
	*locked = true
	return false
}

func keys(s string, isServerStream, isClientStream bool) (string, string, string) {
	if len(s) == 0 || s[0] != '/' {
		panic(fmt.Sprintf("malformed full method: %s", s))
	}
	i := strings.IndexByte(s[1:], '/')
	return kind(isServerStream, isClientStream), s[:i+1], s[i+2:]
}

func kind(server, client bool) string {
	switch {
	case server && client:
		return "bidi_stream"
	case server:
		return "server_stream"
	case client:
		return "client_stream"
	default:
		return "unary"
	}
}
