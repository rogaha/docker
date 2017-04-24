package cluster

import "github.com/docker/go-metrics"

var (
	clusterInfo     metrics.LabeledGauge
	clusterNodes    metrics.Gauge
	clusterManagers metrics.Gauge
)

func init() {
	ns := metrics.NewNamespace("engine", "daemon", nil)

	clusterInfo = ns.NewLabeledGauge("swarm", "The information related to the cluster", metrics.Unit("info"),
		"cluster_id",
		"node_id",
		"node_addr",
	)
	clusterManagers = ns.NewGauge("cluster_managers", "The number of node managers the cluster has", metrics.Unit("managers"))
	clusterNodes = ns.NewGauge("cluster_nodes", "The number of nodes the cluster has", metrics.Unit("nodes"))

	metrics.Register(ns)
}
