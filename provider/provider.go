package provider

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"time"
)

type Provider interface {
	Get() ([]models.Metrics, error)
}

type Config interface {
	Url() string
	File() string
	TimeWindow() time.Duration
	MetricKeys() []string
}

func New(c Config) (Provider, error) {
	if c.File() != "" {
		return newFileProvider(c)
	}
	if c.Url() != "" {
		return newUrlProvider(c)
	}
	return nil, fmt.Errorf("illegal state, could not create provider - file or url are mandatory")
}
