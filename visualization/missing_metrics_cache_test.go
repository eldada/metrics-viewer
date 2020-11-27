package visualization

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_metricsCache_AddMissingMetrics(t *testing.T) {
	origStaleDuration := metricsCacheMaxStaleDuration
	defer func() {
		nowFunc = time.Now
		metricsCacheMaxStaleDuration = origStaleDuration
	}()
	metricsCacheMaxStaleDuration = time.Minute
	var metrics []models.Metrics
	for i := 0; i < 3; i++ {
		metrics = append(metrics, models.Metrics{
			Name:        fmt.Sprintf("m%d", i+1),
			Description: fmt.Sprintf("m%d-desc", i+1),
			Metrics:     []models.Metric{},
		})
	}
	c := newMissingMetricsCache()
	startTime := time.Date(2020, 11, 26, 1, 2, 0, 0, time.Local)

	nowFunc = func() time.Time { return startTime }
	updated := c.AddToMetrics([]models.Metrics{metrics[0], metrics[1]})
	assert.Equal(t, []models.Metrics{metrics[0], metrics[1]}, updated)

	nowFunc = func() time.Time { return startTime.Add(20 * time.Second) }
	updated = c.AddToMetrics([]models.Metrics{metrics[0]})
	assert.Equal(t, []models.Metrics{metrics[0], metrics[1]}, updated)

	nowFunc = func() time.Time { return startTime.Add(40 * time.Second) }
	updated = c.AddToMetrics([]models.Metrics{metrics[0], metrics[2]})
	assert.Equal(t, []models.Metrics{metrics[0], metrics[1], metrics[2]}, updated)

	nowFunc = func() time.Time { return startTime.Add(70 * time.Second) }
	updated = c.AddToMetrics([]models.Metrics{metrics[2]})
	assert.Equal(t, []models.Metrics{metrics[0], metrics[2]}, updated)
}
