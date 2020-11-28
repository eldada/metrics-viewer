package provider

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"os"
	"time"
)

type Provider interface {
	Get() ([]models.Metrics, error)
}

type Config interface {
	UrlMetricsFetcher() UrlMetricsFetcher
	File() string
	TimeWindow() time.Duration
	MetricKeys() []string
}

func New(c Config) (Provider, error) {
	if os.Getenv("MOCK_METRICS_DATA") == "true" {
		return newMockDataProvider(c)
	}
	if c.File() != "" {
		return newFileProvider(c)
	}
	if c.UrlMetricsFetcher() != nil {
		return newUrlProvider(c)
	}
	return nil, fmt.Errorf("illegal state, could not create provider - file or url are mandatory")
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
