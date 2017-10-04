package metrics

import (
	"expvar"
)

var (
	MetricsMap = expvar.NewMap("sesha3")

	MetricsConnect      = new(expvar.Int)
	MetricsTokenRequest = new(expvar.Int)
	MetricsTTYRequest   = new(expvar.Int)
	MetricsHandler      = expvar.Handler()
)

func init() {
	MetricsMap.Set("connect", MetricsConnect)
	MetricsMap.Set("token_req", MetricsTokenRequest)
	MetricsMap.Set("tty_req", MetricsTTYRequest)
}
