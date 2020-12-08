package printer

import (
	"encoding/csv"
	"fmt"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/eldada/metrics-viewer/provider"
	"strings"
	"sync"
	"time"
)

func newCSVPrinter(conf Config) *csvPrinter {
	metrics := make(map[string]int)
	for i, m := range conf.Metrics() {
		metrics[m] = i
	}
	return &csvPrinter{
		writer:     csv.NewWriter(conf.Writer()),
		metrics:    metrics,
		mapMetrics: provider.NewLabelsMetricsMapper(conf.AggregateIgnoreLabels(), ","),
		noHeader:   conf.NoHeader(),
	}
}

type csvPrinter struct {
	writer     *csv.Writer
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
				p.writer.Flush()
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
				p.printAndClearLastRecord()
			})
		}
	}
	return nil
}

func (p *csvPrinter) printAndClearLastRecord() {
	r := p.record
	p.record = nil
	r.Print(p.writer)
	p.writer.Flush()
}

func (p *csvPrinter) flushLastRecord() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recordTimer.Stop()
	p.printAndClearLastRecord()
}

func (p *csvPrinter) printHeader() {
	if p.noHeader {
		return
	}
	header := make([]string, len(p.metrics)+1)
	header[0] = "timestamp"
	for m, i := range p.metrics {
		header[i+1] = m
	}
	p.writer.Write(header)
}

type csvRecord struct {
	ts     time.Time
	values []*float64
}

func (r csvRecord) Print(w *csv.Writer) {
	record := make([]string, len(r.values)+1)
	record[0] = fmt.Sprintf("%s", r.ts.UTC().Format("2006-01-02T15:04:05.000"))
	for i, v := range r.values {
		record[i+1] = ""
		if v != nil {
			record[i+1] = fmt.Sprintf("%f", *v)
		}
	}
	w.Write(record)
}

func (r csvRecord) IsFull() bool {
	for _, v := range r.values {
		if v == nil {
			return false
		}
	}
	return true
}
