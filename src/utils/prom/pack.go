package prom

import "github.com/prometheus/client_golang/prometheus"

var Client = prometheus.NewRegistry()
