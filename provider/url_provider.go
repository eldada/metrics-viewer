package provider

import (
	"bytes"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
)

func newUrlProvider(metricsFetcher UrlMetricsFetcher) (*urlProvider, error) {
	return &urlProvider{
		metricsFetcher: metricsFetcher,
	}, nil
}

type urlProvider struct {
	metricsFetcher UrlMetricsFetcher
}

func (p *urlProvider) Get() ([]models.Metrics, error) {
	data, err := p.metricsFetcher.Get()
	if err != nil {
		return nil, err
	}
	data = append(data, byte('\n'))
	r := bytes.NewReader(data)
	metrics, err := parser.ParseMetrics(r)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
