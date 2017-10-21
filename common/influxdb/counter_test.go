package influxdb_test

import (
	"testing"

	"github.com/bytearena/bytearena/common/influxdb"
)

func TestAdd(t *testing.T) {
	counter := influxdb.NewCounter()

	counter.Add(1)

	if counter.GetAndReset() != 1 {
		panic("Unexpected result")
	}

	if counter.GetAndReset() != 0 {
		panic("Unexpected result")
	}
}
