package provider

import (
	"github.com/eldada/metrics-viewer/models"
	"time"
)

func NewMetricsCache(timeWindow time.Duration) *MetricsCache {
	return &MetricsCache{
		timeWindow: timeWindow,
	}
}

type MetricsCache struct {
	timeWindow        time.Duration
	metricsCollection []models.Metrics
}

func (m *MetricsCache) Add(metricsCollection []models.Metrics) []models.Metrics {
	metricsMap := make(map[string]models.Metrics, len(metricsCollection))
	for _, m := range metricsCollection {
		metricsMap[m.Name] = m
	}
	var newCollection []models.Metrics
	for _, m := range m.metricsCollection {
		fetchedMetrics, found := metricsMap[m.Name]
		if found {
			m.Metrics = append(m.Metrics, fetchedMetrics.Metrics...)
			delete(metricsMap, m.Name)
		}
		newCollection = append(newCollection, m)
	}
	for _, m := range metricsMap {
		newCollection = append(newCollection, m)
	}
	newCollection = filterByTimeWindow(newCollection, m.timeWindow)
	m.metricsCollection = newCollection
	return newCollection
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
