package prome

import (
	"context"
	"fmt"
	"github.com/wsx864321/kim/pkg/log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var once sync.Once

// StartAgent 开启prometheus
func StartAgent(host string, port int) {
	go func() {
		once.Do(func() {
			http.Handle("/", promhttp.Handler())
			addr := fmt.Sprintf("%s:%d", host, port)
			log.Info(context.Background(), "Starting prometheus agent", log.String("addr", addr))
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Error(context.Background(), "prometheus agent listen failed", log.String("error", err.Error()))
			}
		})
	}()
}
