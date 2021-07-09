package process

import (
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "node"

type procCollector struct {
	upDesc         *prometheus.Desc
	stateDesc      *prometheus.Desc
	exitStatusDesc *prometheus.Desc
	startTimeDesc  *prometheus.Desc
	procMgr        *Manager
}

// NewProcCollector returns new Collector exposing supervisord statistics.
func NewProcCollector(mgr *Manager) *procCollector {
	var (
		subsystem  = "supervisord"
		labelNames = []string{"name", "group"}
	)

	return &procCollector{
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Process Up",
			labelNames,
			nil,
		),
		stateDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "state"),
			"Process State",
			labelNames,
			nil,
		),
		exitStatusDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "exit_status"),
			"Process Exit Status",
			labelNames,
			nil,
		),
		startTimeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "start_time_seconds"),
			"Process start time",
			labelNames,
			nil,
		),
		procMgr: mgr,
	}
}

// Describe generates prometheus metric description
func (c *procCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upDesc
	ch <- c.stateDesc
	ch <- c.exitStatusDesc
	ch <- c.startTimeDesc
}

// Collect gathers prometheus metrics for all supervised processes
func (c *procCollector) Collect(ch chan<- prometheus.Metric) {
	c.procMgr.ForEachProcess(func(proc *Process) {
		c.collectProcessMetrics(proc, ch)
	})
}

func (c *procCollector) collectProcessMetrics(proc *Process, ch chan<- prometheus.Metric) {
	labels := []string{proc.GetName(), proc.GetGroup()}

	ch <- prometheus.MustNewConstMetric(c.stateDesc, prometheus.GaugeValue, float64(proc.GetState()), labels...)
	ch <- prometheus.MustNewConstMetric(c.exitStatusDesc, prometheus.GaugeValue, float64(proc.GetExitstatus()), labels...)

	if proc.isRunning() {
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1, labels...)
		ch <- prometheus.MustNewConstMetric(c.startTimeDesc, prometheus.CounterValue, float64(proc.GetStartTime().Unix()), labels...)
	} else {
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0, labels...)
	}

}
