package provider

import "github.com/eldada/metrics-viewer/models"

func newUrlProvider(c Config) (*urlProvider, error) {
	return &urlProvider{
		conf: c,
	}, nil
}

type urlProvider struct {
	conf Config
}

func (p *urlProvider) Get() ([]models.Metrics, error) {
	return nil, nil
}
