package provider

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"os"
	"sort"
	"strings"
	"time"
)

type Provider interface {
	Get() ([]models.Metrics, error)
}

type Config interface {
	UrlMetricsFetcher() UrlMetricsFetcher
	File() string
	TimeWindow() time.Duration
	MetricKeys() []string
}

func New(c Config) (Provider, error) {
	if os.Getenv("MOCK_METRICS_DATA") == "true" {
		return newMockDataProvider(c)
	}
	if c.File() != "" {
		return newFileProvider(c)
	}
	if c.UrlMetricsFetcher() != nil {
		return newUrlProvider(c)
	}
	return nil, fmt.Errorf("illegal state, could not create provider - file or url are mandatory")
}

func filterByTimeWindow(metricsCollection []models.Metrics, window time.Duration) []models.Metrics {
	startFrom := now().UTC().Add(window * time.Duration(-1))
	var newCollection []models.Metrics
	for _, metrics := range metricsCollection {
		var filtered []models.Metric
		for _, metric := range metrics.Metrics {
			if metric.Timestamp.After(startFrom) {
				filtered = append(filtered, metric)
			}
		}
		metrics.Metrics = filtered
		newCollection = append(newCollection, metrics)
	}
	return newCollection
}

func aggregateByLabels(c Config, metricsCollection []models.Metrics) []models.Metrics {
	metricsMap := make(map[string]models.Metrics, 0)
	for _, metrics := range metricsCollection {
		for _, metric := range metrics.Metrics {
			name := generateMetricName(c, metrics.Key, metric.Labels)
			aggMetrics, found := metricsMap[name]
			if !found {
				aggMetrics = models.Metrics{
					Key:  metrics.Key,
					Name: name,
				}
				if metrics.Description != "" {
					aggMetrics.Description = metrics.Description
				}
			}
			aggMetrics.Metrics = append(aggMetrics.Metrics, metric)
			metricsMap[name] = aggMetrics
		}
	}
	newCollection := make([]models.Metrics, 0, len(metricsMap))
	for _, v := range metricsMap {
		newCollection = append(newCollection, v)
	}
	sort.SliceStable(newCollection, func(i, j int) bool {
		return newCollection[i].Name < newCollection[j].Name
	})
	return newCollection
}

func generateMetricName(c Config, key string, labels map[string]string) string {
	name := strings.Builder{}
	name.WriteString(key)
	if len(labels) > 0 {
		name.WriteRune('{')
		first := true
		for k, v := range labels {
			if first {
				first = false
			} else {
				name.WriteRune(',')
			}
			name.WriteString(fmt.Sprintf(`%s="%s"`, k, v))
		}
		name.WriteRune('}')
	}
	return name.String()
}

var nowFunc = time.Now

func now() time.Time {
	return nowFunc()
}
