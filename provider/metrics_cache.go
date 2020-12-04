package provider

import "github.com/eldada/metrics-viewer/models"

func newMetricsCache(conf Config) *metricsCache {
	return &metricsCache{
		conf: conf,
	}
}

type metricsCache struct {
	conf              Config
	metricsCollection []models.Metrics
}

func (m *metricsCache) Add(metricsCollection []models.Metrics) []models.Metrics {
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
	newCollection = filterByTimeWindow(newCollection, m.conf.TimeWindow())
	newCollection = aggregateByLabels(m.conf, newCollection)
	newCollection = filterByRegex(newCollection, m.conf.Filter())
	m.metricsCollection = newCollection
	return newCollection
}
