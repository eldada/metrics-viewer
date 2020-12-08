package printer

import (
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"io"
	"regexp"
)

type Config interface {
	Filter() *regexp.Regexp
	AggregateIgnoreLabels() provider.StringSet
	Format() OutputFormat
	Writer() io.Writer
	Metrics() []string
	NoHeader() bool
}

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
