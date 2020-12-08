package printer

import (
	"fmt"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/eldada/metrics-viewer/provider"
	"io"
	"strings"
	"sync"
	"time"
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
		noHeader:   conf.NoHeader(),
	}
}

type csvPrinter struct {
	writer     io.Writer
	metrics    map[string]int
	mapMetrics provider.MetricsMapperFunc
	noHeader   bool

	printHeaderOnce sync.Once
	record          *csvRecord
	recordTimer     *time.Timer
	mu              sync.Mutex
}

func (p *csvPrinter) Print(entry string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
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
			if p.recordTimer != nil {
				p.recordTimer.Stop()
				p.recordTimer = nil
			}
			if p.record != nil && (p.record.ts != m.Timestamp || p.record.IsFull() || p.record.values[i] != nil) {
				p.record.Print(p.writer)
				fmt.Fprintln(p.writer)
				p.record = nil
			}
			if p.record == nil {
				p.record = &csvRecord{
					ts:     m.Timestamp,
					values: make([]*float64, len(p.metrics)),
				}
			}
			p.record.values[i] = &m.Value
			p.recordTimer = time.AfterFunc(50*time.Millisecond, func() {
				r := p.record
				p.record = nil
				r.Print(p.writer)
				fmt.Fprintln(p.writer)
			})
		}
	}
	return nil
}

func (p *csvPrinter) printHeader() {
	if p.noHeader {
		return
	}
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

type csvRecord struct {
	ts     time.Time
	values []*float64
}

func (r csvRecord) Print(w io.Writer) {
	fmt.Fprintf(w, "%s", r.ts.UTC().Format("2006-01-02T15:04:05.000"))
	for _, v := range r.values {
		if v != nil {
			fmt.Fprintf(w, ",%f", *v)
		} else {
			fmt.Fprint(w, ",")
		}
	}
}

func (r csvRecord) IsFull() bool {
	for _, v := range r.values {
		if v == nil {
			return false
		}
	}
	return true
}
