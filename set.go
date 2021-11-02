package grpcmetrics

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc/codes"
)

func newSet() set {
	return set{s: metrics.NewSet()}
}

type set struct {
	s *metrics.Set
}

func (s *set) WritePrometheus(w io.Writer) {
	s.s.WritePrometheus(w)
}

func (s *set) counter(
	name string, serverStream, clientStream bool, service, method string, code codes.Code,
) *metrics.Counter {
	return s.s.GetOrCreateCounter(mkname(
		name, serverStream, clientStream, service, method, code,
	))
}

func (s *set) histogram(
	name string, serverStream, clientStream bool, service, method string,
) *metrics.Histogram {
	return s.s.GetOrCreateHistogram(mkname(
		name, serverStream, clientStream, service, method, noCode,
	))
}

const noCode = math.MaxUint32

//var (
//	mu    = sync.Mutex{}
//	cache = map[string]map[string]map[string]map[string]map[codes.Code]string{}
//)

func mkname(
	name string, serverStream, clientStream bool, service, method string, code codes.Code,
) string {
	typ := kind(serverStream, clientStream)
	//if x := cache[name]; x != nil {
	//	if x := x[typ]; x != nil {
	//		if x := x[service]; x != nil {
	//			if x := x[method]; x != nil {
	//				if x := x[code]; x != "" {
	//					return x
	//				}
	//			}
	//		}
	//	}
	//}

	var b strings.Builder
	b.Grow(4096)
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
	return b.String()
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

func serviceAndMethod(s string) (string, string) {
	if len(s) == 0 || s[0] != '/' {
		panic(fmt.Sprintf("malformed full method: %s", s))
	}
	i := strings.IndexByte(s[1:], '/')
	return s[:i+1], s[i+1+1:]
}
