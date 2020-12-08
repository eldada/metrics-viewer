package provider

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"regexp"
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
	Interval() time.Duration
	TimeWindow() time.Duration
	Filter() *regexp.Regexp
	AggregateIgnoreLabels() StringSet
}

func New(c Config) (Provider, error) {
	if c.File() != "" {
		return newFileProvider(c)
	}
	if c.UrlMetricsFetcher() != nil {
		return newUrlProvider(c)
	}
	return nil, fmt.Errorf("illegal state, could not create provider - file or url are mandatory")
}

type StringSet map[string]struct{}

func (s StringSet) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

func (s StringSet) Len() int {
	return len(s)
}

func (s *StringSet) Add(values ...string) {
	for _, v := range values {
		(*s)[v] = struct{}{}
	}
}

type MetricsFilterFunc func(metrics models.Metrics) bool

func NewRegexMetricsFilter(regex *regexp.Regexp) MetricsFilterFunc {
	return func(metrics models.Metrics) bool {
		return regex == nil || regex.MatchString(metrics.Name)
	}
}

type MetricsMapperFunc func(metricsCollection []models.Metrics) []models.Metrics

func NewLabelsMetricsMapper(ignoredLabels StringSet, delim string) MetricsMapperFunc {
	return func(metricsCollection []models.Metrics) []models.Metrics {
		metricsMap := make(map[string]models.Metrics, 0)
		for _, metrics := range metricsCollection {
			for _, metric := range metrics.Metrics {
				name := generateMetricName(ignoredLabels, delim, metrics.Key, metric.Labels)
				mappedMetrics, found := metricsMap[name]
				if !found {
					mappedMetrics = models.Metrics{
						Key:  metrics.Key,
						Name: name,
					}
				}
				if mappedMetrics.Description == "" {
					mappedMetrics.Description = metrics.Description
				}
				mappedMetrics.Metrics = append(mappedMetrics.Metrics, metric)
				metricsMap[name] = mappedMetrics
			}
		}
		newCollection := make([]models.Metrics, 0, len(metricsMap))
		for _, v := range metricsMap {
			newCollection = append(newCollection, v)
		}
		return newCollection
	}
}

func generateMetricName(ignoredLabels StringSet, delim string, key string, labels map[string]string) string {
	name := strings.Builder{}
	name.WriteString(key)
	if ignoredLabels.Len() == 1 && ignoredLabels.Contains("ALL") {
		return name.String()
	}
	includeAll := ignoredLabels.Len() == 1 && ignoredLabels.Contains("NONE")
	var orderedLabels []string
	for k := range labels {
		if includeAll || !ignoredLabels.Contains(k) {
			orderedLabels = append(orderedLabels, k)
		}
	}
	sort.SliceStable(orderedLabels, func(i, j int) bool {
		return orderedLabels[i] < orderedLabels[j]
	})
	if len(orderedLabels) > 0 {
		name.WriteRune('{')
		first := true
		for _, k := range orderedLabels {
			if first {
				first = false
			} else {
				name.WriteString(delim)
			}
			name.WriteString(fmt.Sprintf(`%s="%s"`, k, labels[k]))
		}
		name.WriteRune('}')
	}
	return name.String()
}

var nowFunc = time.Now

func now() time.Time {
	return nowFunc()
}
