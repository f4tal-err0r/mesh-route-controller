package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	routesAdded = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "routes_added",
		Help: "Routes added per refresh of routing manifest",
	})
	routesUpdated = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "routes_updated",
		Help: "Routes updated per refresh of routing manifest",
	})
	routesDeleted = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "routes_deleted",
		Help: "Routes deleted per refresh of routing manifest",
	})
)

type metric interface {
	prometheus.Gauge
	Run()
}

func init() {
	prometheus.MustRegister(routesAdded)
	prometheus.MustRegister(routesUpdated)
	prometheus.MustRegister(routesDeleted)
}

func Run() {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":2442", nil))
}
