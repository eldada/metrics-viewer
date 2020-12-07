package printer

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/hpcloud/tail"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"io"
	"regexp"
	"strings"
	"time"
)

type Config interface {
	UrlMetricsFetcher() provider.UrlMetricsFetcher
	File() string
	Interval() time.Duration
	Filter() *regexp.Regexp
	AggregateIgnoreLabels() provider.StringSet
	Format() OutputFormat
	Writer() io.Writer
	Metrics() []string
}

func NewFetcher(conf Config) (MetricEntryFetcher, error) {
	if conf.File() != "" {
		return newFileOpenMetricEntryFetcher(conf.File())
	}
	if conf.UrlMetricsFetcher() != nil {
		return newUrlOpenMetricsEntryFetcher(conf.UrlMetricsFetcher(), conf.Interval())
	}
	return nil, fmt.Errorf("illegal state, could not create fetcher - file or url are mandatory")
}

type MetricEntryFetcher interface {
	Entries() <-chan string
	Close() error
}

func newFileOpenMetricEntryFetcher(filename string) (*fileOpenMetricEntryFetcher, error) {
	t, err := tail.TailFile(filename, tail.Config{
		Follow: true,
		ReOpen: true,
	})
	if err != nil {
		return nil, err
	}
	fetcher := fileOpenMetricEntryFetcher{
		tail:    t,
		entries: make(chan string),
	}
	go fetcher.fetch()
	return &fetcher, nil
}

type fileOpenMetricEntryFetcher struct {
	tail    *tail.Tail
	entries chan string
	closed  bool
}

func (f *fileOpenMetricEntryFetcher) fetch() {
	entry := strings.Builder{}
	for line := range f.tail.Lines {
		if f.closed {
			break
		}
		if line == nil {
			continue
		}
		entry.WriteString(line.Text)
		entry.WriteRune('\n')
		if line.Text == "" || strings.HasPrefix(line.Text, "#") {
			continue
		}
		f.entries <- entry.String()
		entry.Reset()
	}
}

func (f *fileOpenMetricEntryFetcher) Close() error {
	f.closed = true
	close(f.entries)
	return f.tail.Stop()
}

func (f *fileOpenMetricEntryFetcher) Entries() <-chan string {
	return f.entries
}

func newUrlOpenMetricsEntryFetcher(urlMetricsFetcher provider.UrlMetricsFetcher, interval time.Duration) (*urlOpenMetricsEntryFetcher, error) {
	fetcher := urlOpenMetricsEntryFetcher{
		urlMetricsFetcher: urlMetricsFetcher,
		interval:          interval,
		entries:           make(chan string),
	}
	go fetcher.fetch()
	return &fetcher, nil
}

type urlOpenMetricsEntryFetcher struct {
	urlMetricsFetcher provider.UrlMetricsFetcher
	interval          time.Duration
	entries           chan string
	closed            bool
}

func (f *urlOpenMetricsEntryFetcher) fetch() {
	entry := strings.Builder{}
	for {
		if f.closed {
			break
		}
		data, err := f.urlMetricsFetcher.Get()
		if err != nil {
			log.Error(err)
			sleep(f.interval)
			continue
		}
		entry.Reset()
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			txt := scanner.Text()
			entry.WriteString(txt)
			entry.WriteRune('\n')
			if txt == "" || strings.HasPrefix(txt, "#") {
				continue
			}
			f.entries <- entry.String()
			entry.Reset()
		}
		if f.closed {
			break
		}
		sleep(f.interval)
	}
}

func (f *urlOpenMetricsEntryFetcher) Entries() <-chan string {
	return f.entries
}

func (f *urlOpenMetricsEntryFetcher) Close() error {
	f.closed = true
	close(f.entries)
	return nil
}

var sleepFunc = time.Sleep

func sleep(d time.Duration) {
	sleepFunc(d)
}
