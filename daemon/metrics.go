package daemon

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/mount"
	"github.com/docker/docker/pkg/plugingetter"
	"github.com/docker/go-metrics"
	"github.com/pkg/errors"
)

var (
	containerActions          metrics.LabeledTimer
	imageActions              metrics.LabeledTimer
	networkActions            metrics.LabeledTimer
	engineInfo                metrics.LabeledGauge
	engineCpus                metrics.Gauge
	engineMemory              metrics.Gauge
	healthChecksCounter       metrics.Counter
	healthChecksFailedCounter metrics.Counter
)

func init() {
	ns := metrics.NewNamespace("engine", "daemon", nil)

	containerActions = ns.NewLabeledTimer("container_actions", "The number of seconds it takes to process each container action", "action")
	for _, a := range []string{
		"start",
		"changes",
		"commit",
		"create",
		"delete",
	} {
		containerActions.WithValues(a).Update(0)
	}
	networkActions = ns.NewLabeledTimer("network_actions", "The number of seconds it takes to process each network action", "action")
	engineInfo = ns.NewLabeledGauge("engine", "The information related to the engine and the OS it's running on", metrics.Unit("info"),
		"version",
		"commit",
		"architecture",
		"graph_driver",
		"kernel", "os",
		"os_type", "id",
	)
	engineCpus = ns.NewGauge("engine_cpus", "The number of cpus that the host system of the engine has", metrics.Unit("cpus"))
	engineMemory = ns.NewGauge("engine_memory", "The number of bytes of memory that the host system of the engine has", metrics.Bytes)
	healthChecksCounter = ns.NewCounter("health_checks", "The total number of health checks")
	healthChecksFailedCounter = ns.NewCounter("health_checks_failed", "The total number of failed health checks")
	imageActions = ns.NewLabeledTimer("image_actions", "The number of seconds it takes to process each image action", "action")
	metrics.Register(ns)
}

const metricsPluginType = "MetricsCollector"

func (d *Daemon) cleanupMetricsPlugins() {
	ls := d.PluginStore.GetAllManagedPluginsByCap(metricsPluginType)
	var wg sync.WaitGroup
	wg.Add(len(ls))

	for _, p := range ls {
		go func() {
			defer wg.Done()
			stopMetricsPlugin(p)
		}()
	}
	wg.Wait()

	if d.metricsPluginListener != nil {
		d.metricsPluginListener.Close()
	}
}

type metricsPlugin struct {
	plugingetter.CompatPlugin
}

func (p metricsPlugin) sock() string {
	return "metrics.sock"
}

func (p metricsPlugin) sockBase() string {
	return filepath.Join(p.BasePath(), "run", "docker")
}

func startMetricsPlugin(p plugingetter.CompatPlugin) error {
	type metricsPluginResponse struct {
		Err string
	}
	var res metricsPluginResponse
	if err := p.Client().Call(metricsPluginType+".StartMetrics", nil, &res); err != nil {
		return errors.Wrap(err, "could not start metrics plugin")
	}
	if res.Err != "" {
		return errors.New(res.Err)
	}
	return nil
}

func stopMetricsPlugin(p plugingetter.CompatPlugin) {
	if err := p.Client().Call(metricsPluginType+".StopMetrics", nil, nil); err != nil {
		logrus.WithError(err).WithField("name", p.Name()).Error("error stopping metrics collector")
	}

	mp := metricsPlugin{p}
	sockPath := filepath.Join(mp.sockBase(), mp.sock())
	if err := mount.Unmount(sockPath); err != nil {
		if mounted, _ := mount.Mounted(sockPath); mounted {
			logrus.WithError(err).WithField("name", p.Name()).WithField("socket", sockPath).Error("error unmounting metrics socket for plugin")
		}
	}
	return
}
