package printer

import (
	"fmt"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/eldada/metrics-viewer/provider"
	"io"
	"strings"
	"sync"
)

type OutputFormat string

const (
	OpenMetricsFormat OutputFormat = "open-metrics"
	CSVFormat         OutputFormat = "csv"
)

var SupportedOutputFormats = map[string]OutputFormat{
	string(OpenMetricsFormat): OpenMetricsFormat,
	string(CSVFormat):         CSVFormat,
}

func NewPrinter(conf Config) (Printer, error) {
	switch conf.Format() {
	case OpenMetricsFormat:
		return &openMetricsPrinter{writer: conf.Writer()}, nil
	case CSVFormat:
		return newCSVPrinter(conf), nil
	}
	return nil, fmt.Errorf("unexpected output format: %s", conf.Format())
}

type Printer interface {
	Print(entry string) error
}

type openMetricsPrinter struct {
	writer io.Writer
}

func (p *openMetricsPrinter) Print(entry string) error {
	_, err := fmt.Fprintln(p.writer, entry)
	return err
}

func newCSVPrinter(conf Config) *csvPrinter {
	metrics := make(map[string]int)
	for i, m := range conf.Metrics() {
		metrics[m] = i
	}
	return &csvPrinter{
		writer:     conf.Writer(),
		metrics:    metrics,
		mapMetrics: provider.NewLabelsMetricsMapper(conf.AggregateIgnoreLabels(), "|"),
	}
}

type csvPrinter struct {
	printHeaderOnce sync.Once
	writer          io.Writer
	metrics         map[string]int
	mapMetrics      provider.MetricsMapperFunc
}

func (p *csvPrinter) Print(entry string) error {
	p.printHeaderOnce.Do(p.printHeader)
	metricsCollection, err := parser.ParseMetrics(strings.NewReader(entry))
	if err != nil {
		return err
	}
	metricsCollection = p.mapMetrics(metricsCollection)
	for _, metrics := range metricsCollection {
		i, found := p.metrics[metrics.Name]
		if !found {
			continue
		}
		for _, m := range metrics.Metrics {
			fmt.Fprintf(p.writer, "%s", m.Timestamp.UTC().Format("2006-01-02T15:04:05.000"))
			for j := 0; j < i; j++ {
				fmt.Fprint(p.writer, ",")
			}
			fmt.Fprintf(p.writer, ",%f", m.Value)
			for j := i + 1; j < len(p.metrics); j++ {
				fmt.Fprint(p.writer, ",")
			}
			fmt.Fprintln(p.writer)
		}
	}
	return nil
}

func (p *csvPrinter) printHeader() {
	_, _ = fmt.Fprint(p.writer, "timestamp")
	metrics := make([]string, len(p.metrics))
	for m, i := range p.metrics {
		metrics[i] = m
	}
	for _, m := range metrics {
		_, _ = fmt.Fprintf(p.writer, ",%s", m)
	}
	_, _ = fmt.Fprintln(p.writer)
}
