package build

import "github.com/docker/go-metrics"

var (
	triggeredBuilds metrics.LabeledTimer
)

func init() {
	ns := metrics.NewNamespace("engine", "daemon", nil)

	triggeredBuilds = ns.NewLabeledTimer("triggered_builds", "The number of seconds it takes to build the image artifact", "status")
	for _, a := range []string{
		"success",
		"fail",
	} {
		triggeredBuilds.WithValues(a).Update(0)
	}

	metrics.Register(ns)
}
