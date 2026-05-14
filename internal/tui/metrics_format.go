package tui

import (
	"time"
)

// metricUnit tells the formatter how to render a metric value.
type metricUnit int

const (
	unitCount metricUnit = iota
	unitBytes
	unitDuration
)

// metricUnits maps the ClickHouse metric / event / async-metric names we care
// about to their semantic unit. Unknown keys default to unitCount.
var metricUnits = map[string]metricUnit{
	"MemoryTracking":                          unitBytes,
	"MemoryResident":                          unitBytes,
	"jemalloc.allocated":                      unitBytes,
	"OSMemoryAvailable":                       unitBytes,
	"OSMemoryTotal":                           unitBytes,
	"NetworkReceiveBytes":                     unitBytes,
	"NetworkSendBytes":                        unitBytes,
	"DiskUsed_default":                        unitBytes,
	"DiskTotal_default":                       unitBytes,
	"InsertedBytes":                           unitBytes,
	"ReadBufferFromFileDescriptorReadBytes":   unitBytes,
	"WriteBufferFromFileDescriptorWriteBytes": unitBytes,
	"Uptime": unitDuration,
}

func formatMetric(key string, value int64) string {
	switch metricUnits[key] {
	case unitBytes:
		return humanBytes(value)
	case unitDuration:
		return humanDuration(time.Duration(value) * time.Second)
	}
	return humanCount(uint64(value))
}

// perSecondRate computes the per-second rate between two counter samples.
// Returns 0 when the elapsed time is zero or negative, or when the counter
// reset (curr < prev) — typical after a ClickHouse restart.
func perSecondRate(prev, curr uint64, elapsed time.Duration) float64 {
	if elapsed <= 0 || curr < prev {
		return 0
	}
	return float64(curr-prev) / elapsed.Seconds()
}
