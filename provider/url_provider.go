package provider

import (
	"bytes"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
)

func newUrlProvider(c Config) (*urlProvider, error) {
	return &urlProvider{
		conf:          c,
		cachedMetrics: newMetricsCache(c),
	}, nil
}

type urlProvider struct {
	conf          Config
	cachedMetrics *metricsCache
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
	metrics = p.cachedMetrics.Add(metrics)
	return metrics, nil
}
