package provider

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/pkg/errors"
	"math/rand"
	"time"
)

func newMockDataProvider(c Config) (*mockDataProvider, error) {
	return &mockDataProvider{
		conf: c,
	}, nil
}

type mockDataProvider struct {
	conf Config
}

func (p *mockDataProvider) Get() ([]models.Metrics, error) {
	n := rand.Intn(10)

	metrics := make([]models.Metrics, 0, n)
	for i := 0; i < n; i++ {
		metrics = append(metrics, models.Metrics{
			Metrics: []models.Metric{
				{Value: 1.2323 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now()},
				{Value: 1.56443213 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(1 * time.Second)},
				{Value: 1.923491 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(2 * time.Second)},
				{Value: 2.31231 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(3 * time.Second)},
				{Value: 1.223132 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(4 * time.Second)},
				{Value: 3.21321 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(5 * time.Second)},
				{Value: 1.213213 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(6 * time.Second)},
			},
			Name:        fmt.Sprintf("Metric %d", i),
			Description: fmt.Sprintf("Metric %d description", i),
		})
	}

	if rand.Intn(10) < 1 {
		return nil, errors.New("can't get metrics")
	}

	return metrics, nil
}
