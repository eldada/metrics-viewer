package provider

import (
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
	"os"
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
	// filter
	return metrics, nil
}
