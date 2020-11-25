package parser

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/jfrog/jfrog-client-go/utils/log"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io"
	"sort"
	"time"
)

func ParseMetrics(r io.Reader) ([]models.Metrics, error) {
	txtParser := expfmt.TextParser{}
	prometheusMetrics, err := txtParser.TextToMetricFamilies(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics; cause: %w", err)
	}
	metricsCollection := make([]models.Metrics, 0)
	for name, metricFamily := range prometheusMetrics {
		if metricFamily.Type == nil {
			log.Warn("undefined type for metric:", name)
			continue
		}
		switch *metricFamily.Type {
		case io_prometheus_client.MetricType_GAUGE:
		case io_prometheus_client.MetricType_COUNTER:
		case io_prometheus_client.MetricType_UNTYPED:
		default:
			log.Warn(fmt.Sprintf("metric '%s' has unsupported type: %s", name, metricFamily.Type.String()))
			continue
		}
		metrics := models.Metrics{
			Name: name,
		}
		if metricFamily.Help != nil {
			metrics.Description = *metricFamily.Help
		}
		for _, promMetric := range metricFamily.Metric {
			metric := models.Metric{}
			if promMetric.TimestampMs == nil {
				log.Warn(fmt.Sprintf("metric %s has an entry with no timestamp; skipped.", name))
				continue
			}
			metric.Timestamp = time.Unix(0, promMetric.GetTimestampMs()*int64(1000000))
			// TODO handle labels
			// metric.Labels = convertLabels(promMetric.Label)
			switch *metricFamily.Type {
			case io_prometheus_client.MetricType_COUNTER:
				metric.Value = promMetric.Counter.GetValue()
			case io_prometheus_client.MetricType_GAUGE:
				metric.Value = promMetric.Gauge.GetValue()
			case io_prometheus_client.MetricType_UNTYPED:
				metric.Value = promMetric.Untyped.GetValue()
			}
			metrics.Metrics = append(metrics.Metrics, metric)
		}
		// Sort the metric entries by timestamp
		sort.SliceStable(metrics.Metrics, func(i, j int) bool {
			return metrics.Metrics[i].Timestamp.Before(metrics.Metrics[j].Timestamp)
		})
		metricsCollection = append(metricsCollection, metrics)
	}
	// Sort by name to make the order predictable
	sort.SliceStable(metricsCollection, func(i, j int) bool {
		return metricsCollection[i].Name < metricsCollection[j].Name
	})
	return metricsCollection, nil
}
