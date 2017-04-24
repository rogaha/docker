// +build !windows

package daemon

import (
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/mount"
	"github.com/docker/docker/pkg/plugingetter"
	"github.com/docker/docker/pkg/plugins"
	metrics "github.com/docker/go-metrics"
	"github.com/pkg/errors"
)

func (d *Daemon) registerMetricsPluginCallback() error {
	var err error
	sockPath := filepath.Join(d.configStore.ExecRoot, "metrics.sock")
	d.metricsPluginListener, err = net.Listen("unix", sockPath)
	if err != nil {
		return errors.Wrap(err, "error setting up metrics plugin listener")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	go func() {
		http.Serve(d.metricsPluginListener, mux)
	}()

	d.PluginStore.Handle(metricsPluginType, func(name string, client *plugins.Client) {
		p, err := d.PluginStore.Get(name, metricsPluginType, plugingetter.Acquire)
		if err != nil {
			return
		}

		defer func() {
			if err != nil {
				d.PluginStore.Get(name, metricsPluginType, plugingetter.Release)
			}
		}()

		mp := metricsPlugin{p}
		sockBase := mp.sockBase()
		if err := os.MkdirAll(sockBase, 0755); err != nil {
			logrus.WithError(err).WithField("name", name).WithField("path", sockBase).Error("error creating metrics plugin base path")
			return
		}

		pluginSockPath := filepath.Join(sockBase, mp.sock())
		_, err = os.Stat(pluginSockPath)
		if err == nil {
			mount.Unmount(pluginSockPath)
		} else {
			logrus.WithField("path", pluginSockPath).Debugf("creating plugin socket")
			f, err := os.OpenFile(pluginSockPath, os.O_CREATE, 0600)
			if err != nil {
				return
			}
			f.Close()
		}

		if err := mount.Mount(sockPath, pluginSockPath, "none", "bind,ro"); err != nil {
			logrus.WithError(err).WithField("name", name).Error("could not mount metrics socket to plugin")
			return
		}

		if err := startMetricsPlugin(p); err != nil {
			logrus.WithError(err).WithField("name", name).Error("error while activating metrics plugin")
		}
	})

	return nil
}
