package metrics

import (
	"expvar"
	"log"
	"time"
)

var (
	MetricsMap = expvar.NewMap("sesha3")

	MetricsConnectionCount   = new(expvar.Int)
	MetricsCurrentConnection = new(expvar.Int)
	MetricsTokenRequest      = new(expvar.Int)
	MetricsTokenRequestCount = new(expvar.Int)
	MetricsTokenResponseTime = new(expvar.String)
	MetricsTTYRequest        = new(expvar.Int)
	MetricsTTYResponseTime   = new(expvar.String)
	MetricsHandler           = expvar.Handler()
)

func init() {
	MetricsMap.Set("connection_count", MetricsConnectionCount)
	MetricsMap.Set("current_connection", MetricsCurrentConnection)
	MetricsMap.Set("token_req", MetricsTokenRequest)
	MetricsMap.Set("token_req_count", MetricsTokenRequestCount)
	MetricsMap.Set("token_responce", MetricsTokenResponseTime)
	MetricsMap.Set("tty_req", MetricsTTYRequest)
	MetricsMap.Set("tty_responce", MetricsTTYResponseTime)
}

func StoreData() {
	go func() {
		for {
			time.Sleep(3 * time.Second)
			log.Println(expvar.Get("sesha3"))
			//			expvar.Do(func(variable expvar.KeyValue) {
			//				log.Printf("expvar.Key: %s expvar.Value: %s", variable.Key, variable.Value)
			//			})
		}
	}()
}
