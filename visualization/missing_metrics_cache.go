package visualization

import (
	"github.com/eldada/metrics-viewer/models"
	"sort"
	"time"
)

type missingMetricsCache map[string]metricsCacheValue

type metricsCacheValue struct {
	name        string
	description string
	lastSeen    time.Time
}

func newMissingMetricsCache() missingMetricsCache {
	return make(missingMetricsCache, 0)
}

func (c missingMetricsCache) AddToMetrics(metrics []models.Metrics) []models.Metrics {
	var updatedMetrics []models.Metrics
	seenJustNow := map[string]bool{}
	for _, m := range metrics {
		updatedMetrics = append(updatedMetrics, m)
		c.put(m)
		seenJustNow[m.Name] = true
	}
	c.evictAllLastSeenBefore(now().Add(-metricsCacheMaxStaleDuration))
	for k, v := range c {
		if _, ok := seenJustNow[k]; ok {
			continue
		}
		metricPlaceholder := models.Metrics{
			Name:        v.name,
			Description: v.description,
			Metrics:     []models.Metric{},
		}
		updatedMetrics = append(updatedMetrics, metricPlaceholder)
	}
	sort.SliceStable(updatedMetrics, func(i, j int) bool {
		return updatedMetrics[i].Name < updatedMetrics[j].Name
	})
	return updatedMetrics
}

func (c missingMetricsCache) put(m models.Metrics) {
	c[m.Name] = metricsCacheValue{
		name:        m.Name,
		description: m.Description,
		lastSeen:    now(),
	}
}

func (c missingMetricsCache) evictAllLastSeenBefore(t time.Time) (evictedKeys []string) {
	for k, v := range c {
		if v.lastSeen.Before(t) {
			delete(c, k)
			evictedKeys = append(evictedKeys, k)
		}
	}
	return evictedKeys
}

var metricsCacheMaxStaleDuration = time.Hour
var nowFunc = time.Now

func now() time.Time {
	return nowFunc()
}
