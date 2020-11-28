package provider

import (
	"bytes"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
)

func newUrlProvider(c Config) (*urlProvider, error) {
	return &urlProvider{
		conf: c,
	}, nil
}

type urlProvider struct {
	conf          Config
	cachedMetrics []models.Metrics
}

func (p *urlProvider) Get() ([]models.Metrics, error) {
	data, err := p.conf.UrlMetricsFetcher().Get()
	if err != nil {
		return nil, err
	}
	data = append(data, byte('\n'))
	r := bytes.NewReader(data)
	metrics, err := parser.ParseMetrics(r)
	if err != nil {
		return nil, err
	}
	metrics = p.appendToCachedMetrics(metrics)
	return metrics, nil
}

func (p *urlProvider) appendToCachedMetrics(metricsCollection []models.Metrics) []models.Metrics {
	metricsMap := make(map[string]models.Metrics, len(metricsCollection))
	for _, m := range metricsCollection {
		metricsMap[m.Name] = m
	}
	var newCollection []models.Metrics
	for _, m := range p.cachedMetrics {
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
	newCollection = filterByTimeWindow(newCollection, p.conf.TimeWindow())
	p.cachedMetrics = newCollection
	return newCollection
}
