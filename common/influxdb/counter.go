package influxdb

import (
	"sync/atomic"
)

type Counter struct {
	count int32
}

func NewCounter() *Counter {
	return &Counter{0}
}

func (counter *Counter) Add(nbr int) {
	atomic.AddInt32(&counter.count, int32(nbr))
}

func (counter *Counter) GetAndReset() int {
	count := atomic.LoadInt32(&counter.count)

	atomic.StoreInt32(&counter.count, 0)

	return int(count)
}
