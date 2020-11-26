package provider

import "github.com/eldada/metrics-viewer/models"

func newFileProvider(c Config) (*fileProvider, error) {
	return &fileProvider{
		conf: c,
	}, nil
}

type fileProvider struct {
	conf Config
}

func (p *fileProvider) Get() ([]models.Metrics, error) {
	return nil, nil
}
