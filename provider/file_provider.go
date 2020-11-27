package provider

import (
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
	"os"
	"time"
)

func newFileProvider(c Config) (*fileProvider, error) {
	return &fileProvider{
		conf: c,
	}, nil
}

type fileProvider struct {
	conf Config
}

func (p *fileProvider) Get() ([]models.Metrics, error) {
	f, err := os.Open(p.conf.File())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	metrics, err := parser.ParseMetrics(f)
	if err != nil {
		return nil, err
	}
	metrics = filterByTimeWindow(metrics, p.conf.TimeWindow())
	return metrics, nil
}

func filterByTimeWindow(metricsCollection []models.Metrics, window time.Duration) []models.Metrics {
	startFrom := now().UTC().Add(window * time.Duration(-1))
	var newCollection []models.Metrics
	for _, metrics := range metricsCollection {
		var filtered []models.Metric
		for _, metric := range metrics.Metrics {
			if metric.Timestamp.After(startFrom) {
				filtered = append(filtered, metric)
			}
		}
		metrics.Metrics = filtered
		newCollection = append(newCollection, metrics)
	}
	return newCollection
}
