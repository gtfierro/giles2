package archiver

import (
	"sync/atomic"
)

type metric struct {
	value uint64
}

func newMetric() *metric {
	return &metric{0}
}

func (m *metric) Mark(num uint64) {
	atomic.AddUint64(&m.value, num)
}

func (m *metric) Get() uint64 {
	return atomic.LoadUint64(&m.value)
}

func (m *metric) GetAndReset() uint64 {
	val := atomic.LoadUint64(&m.value)
	atomic.StoreUint64(&m.value, 0)
	return val
}

type metricMap map[string]*metric

func (m metricMap) addMetric(name string) {
	m[name] = newMetric()
}

func (m metricMap) report() map[string]uint64 {
	report := make(map[string]uint64, len(m))
	for k, v := range m {
		report[k] = v.Get()
	}
	return report
}
