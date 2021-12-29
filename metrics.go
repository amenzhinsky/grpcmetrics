package grpcmetrics

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc/codes"
)

func newCounter(name string) *counter {
	return &counter{newMetric(name)}
}

type counter struct {
	*metric
}

func (c *counter) with(set *metrics.Set, typ, method string, code codes.Code) *metrics.Counter {
	return c.metric.with(set, typ, method, code, func(set *metrics.Set, name string) interface{} {
		if set != nil {
			return set.NewCounter(name)
		}
		return metrics.NewCounter(name)
	}).(*metrics.Counter)
}

func newHistogram(name string) *histogram {
	return &histogram{newMetric(name)}
}

type histogram struct {
	*metric
}

func (h *histogram) with(set *metrics.Set, typ, method string) *metrics.Histogram {
	return h.metric.with(set, typ, method, noCode, func(set *metrics.Set, name string) interface{} {
		if set != nil {
			return set.NewHistogram(name)
		}
		return metrics.NewHistogram(name)
	}).(*metrics.Histogram)
}

func newMetric(name string) *metric {
	return &metric{
		name:    name,
		methods: map[string]map[codes.Code]interface{}{},
	}
}

type metric struct {
	mu      sync.RWMutex
	name    string
	methods map[string]map[codes.Code]interface{} // TODO: use metrics.Metric when it's exported
}

func (m *metric) with(
	set *metrics.Set, typ, method string, code codes.Code,
	fn func(set *metrics.Set, name string) interface{},
) interface{} {
	m.mu.RLock() // try read lock first and promote to write lock if needed
	var locked bool
	defer func() {
		if locked {
			m.mu.Unlock()
		} else {
			m.mu.RUnlock()
		}
	}()
	methods, ok := m.methods[method]
	if !ok {
		if m.lock(&locked) || m.methods[method] == nil {
			m.methods[method] = map[codes.Code]interface{}{}
		}
		methods = m.methods[method]
	}
	metric, ok := methods[code]
	if !ok {
		if m.lock(&locked) || methods[code] == nil {
			service, method := splitMethodName(method)
			var b strings.Builder
			b.Grow(1024) // should be enough for almost all metric names
			b.WriteString(m.name)
			b.WriteString(`{grpc_type="`)
			b.WriteString(typ)
			b.WriteString(`",grpc_service="`)
			b.WriteString(service)
			b.WriteString(`",grpc_method="`)
			b.WriteString(method)
			if code != noCode {
				b.WriteString(`",grpc_code="`)
				b.WriteString(code.String())
			}
			b.WriteString(`"}`)
			methods[code] = fn(set, b.String())
		}
		metric = methods[code]
	}
	return metric
}

func (m *metric) lock(locked *bool) bool {
	if *locked {
		return true
	}
	m.mu.RUnlock()
	m.mu.Lock()
	*locked = true
	return false
}

const noCode = math.MaxUint32

func splitMethodName(s string) (string, string) {
	if len(s) == 0 || s[0] != '/' {
		panic(fmt.Sprintf("malformed full method: %s", s))
	}
	i := strings.IndexByte(s[1:], '/')
	return s[1 : i+1], s[i+2:]
}

func streamType(server, client bool) string {
	switch {
	case server && client:
		return "bidi_stream"
	case server:
		return "server_stream"
	case client:
		return "client_stream"
	default:
		return unary
	}
}
