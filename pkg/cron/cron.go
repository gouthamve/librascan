package cron

import (
	"time"

	"github.com/labstack/gommon/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var cronDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "cron_duration_seconds",
	Help: "Duration of cron jobs",
}, []string{"job", "status"})

type Job interface {
	Name() string
	Period() time.Duration
	Run() error
}

type CronRunner struct {
	Jobs []Job
}

func NewCronRunner(jobs []Job) *CronRunner {
	return &CronRunner{Jobs: jobs}
}

// TODO: Adding stopping of cron jobs

func (c *CronRunner) Run() {
	for _, job := range c.Jobs {
		go func(j Job) {
			for {
				start := time.Now()
				err := j.Run()
				status := "success"
				if err != nil {
					status = "failure"
				}
				cronDuration.WithLabelValues(j.Name(), status).Observe(time.Since(start).Seconds())
				log.Errorf("Job %s failed: %v", j.Name(), err)

				time.Sleep(j.Period())
			}
		}(job)
	}
}
