package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/jfrog/jfrog-client-go/utils/log"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"time"
)

func ParseMetrics(r io.Reader) ([]models.Metrics, error) {
	txtParser := expfmt.TextParser{}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics; cause: %w", err)
	}
	br := bytes.NewReader(data)
	prometheusMetrics, err := txtParser.TextToMetricFamilies(br)
	// Handle parsing errors due to bad comments, such as "second HELP line for metric"
	if err != nil {
		originalErr := err
		br = bytes.NewReader(data)
		txtWithoutComments := strings.Builder{}
		scanner := bufio.NewScanner(br)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "#") {
				continue
			}
			txtWithoutComments.WriteString(scanner.Text())
			txtWithoutComments.WriteRune('\n')
		}
		if scanner.Err() != nil {
			return nil, fmt.Errorf("failed to read metrics without comments; cause: %w", err)
		}
		prometheusMetrics, err = txtParser.TextToMetricFamilies(strings.NewReader(txtWithoutComments.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse metrics without comments; cause: %w; original cause: %v", err, originalErr)
		}
	}
	metricsCollection := make([]models.Metrics, 0)
	for key, metricFamily := range prometheusMetrics {
		if metricFamily.Type == nil {
			log.Warn("undefined type for metric:", key)
			continue
		}
		switch *metricFamily.Type {
		case io_prometheus_client.MetricType_GAUGE:
		case io_prometheus_client.MetricType_COUNTER:
		case io_prometheus_client.MetricType_UNTYPED:
		default:
			log.Warn(fmt.Sprintf("metric '%s' has unsupported type: %s", key, metricFamily.Type.String()))
			continue
		}
		metrics := models.Metrics{
			Key:  key,
			Name: key,
		}
		if metricFamily.Help != nil {
			metrics.Description = *metricFamily.Help
		}
		for _, promMetric := range metricFamily.Metric {
			metric := models.Metric{}
			if promMetric.TimestampMs != nil {
				metric.Timestamp = time.Unix(0, promMetric.GetTimestampMs()*int64(1000000))
			} else {
				metric.Timestamp = time.Now()
			}
			metric.Labels = convertLabels(promMetric.Label)
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

func convertLabels(labels []*io_prometheus_client.LabelPair) map[string]string {
	result := make(map[string]string, 0)
	for _, label := range labels {
		result[*label.Name] = *label.Value
	}
	return result
}
