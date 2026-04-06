package observability

import "time"

type Metrics interface {
	IncCounter(name string, labels ...string)
	ObserveDuration(name string, value time.Duration, labels ...string)
}

type NoopMetrics struct{}

func NewNoopMetrics() Metrics {
	return NoopMetrics{}
}

func (NoopMetrics) IncCounter(string, ...string) {}

func (NoopMetrics) ObserveDuration(string, time.Duration, ...string) {}
