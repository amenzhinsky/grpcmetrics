package grpcmetrics

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/VictoriaMetrics/metrics"
	"google.golang.org/grpc/codes"
)

type set struct {
	*metrics.Set
}

func (s *set) counter(name string) any {
	if s.Set != nil {
		return s.Set.NewCounter(name)
	}
	return metrics.NewCounter(name)
}

func (s *set) histogram(name string) any {
	if s.Set != nil {
		return s.Set.NewHistogram(name)
	}
	return metrics.NewHistogram(name)
}

func newCounter(name string) *counter {
	return &counter{newMetric(name)}
}

type counter struct {
	*metric
}

func (c *counter) with(s *set, typ, method string, code codes.Code) *metrics.Counter {
	return c.metric.with(typ, method, code, s.counter).(*metrics.Counter)
}

func newHistogram(name string) *histogram {
	return &histogram{newMetric(name)}
}

type histogram struct {
	*metric
}

func (h *histogram) with(s *set, typ, method string) *metrics.Histogram {
	return h.metric.with(typ, method, noCode, s.histogram).(*metrics.Histogram)
}

func newMetric(name string) *metric {
	return &metric{
		name:    name,
		methods: map[string]map[codes.Code]any{},
	}
}

type metric struct {
	mu      sync.RWMutex
	name    string
	methods map[string]map[codes.Code]any // TODO: use metrics.Metric when it's exported
}

func (m *metric) with(
	typ, method string, code codes.Code, new func(name string) any,
) any {
	m.mu.RLock() // try read lock first and promote to write lock if needed
	var upgraded bool
	defer func() {
		if upgraded {
			m.mu.Unlock()
		} else {
			m.mu.RUnlock()
		}
	}()
	methods, ok := m.methods[method]
	if !ok {
		upgraded = true
		m.mu.RUnlock()
		m.mu.Lock()

		// r to w mutex upgrading is not atomic,
		// so we need to check that another routine hasn't got here first
		if m.methods[method] == nil {
			m.methods[method] = map[codes.Code]any{}
		}
		methods = m.methods[method]
	}
	metric, ok := methods[code]
	if !ok {
		wasUpgraded := upgraded
		if !upgraded {
			upgraded = true
			m.mu.RUnlock()
			m.mu.Lock()
		}
		if wasUpgraded || methods[code] == nil {
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
			methods[code] = new(b.String())
		}
		metric = methods[code]
	}
	return metric
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
