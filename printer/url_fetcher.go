package printer

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/eldada/metrics-viewer/provider"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

func newUrlOpenMetricsEntryFetcher(urlMetricsFetcher provider.UrlMetricsFetcher, interval time.Duration) (*urlOpenMetricsEntryFetcher, error) {
	return newUrlOpenMetricsEntryFetcherWithContext(context.Background(), urlMetricsFetcher, interval)
}

func newUrlOpenMetricsEntryFetcherWithContext(ctx context.Context, urlMetricsFetcher provider.UrlMetricsFetcher, interval time.Duration) (*urlOpenMetricsEntryFetcher, error) {
	fetcher := urlOpenMetricsEntryFetcher{
		urlMetricsFetcher: urlMetricsFetcher,
		interval:          interval,
		entries:           make(chan string),
		ctx:               ctx,
	}
	go fetcher.fetch()
	return &fetcher, nil
}

type urlOpenMetricsEntryFetcher struct {
	urlMetricsFetcher provider.UrlMetricsFetcher
	interval          time.Duration
	entries           chan string
	closed            bool
	ctx               context.Context
}

func (f *urlOpenMetricsEntryFetcher) fetch() {
	defer close(f.entries)
	entry := strings.Builder{}
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			if f.closed {
				return
			}
			data, err := f.urlMetricsFetcher.Get()
			if err != nil {
				log.Error(err)
				continue
			}
			entry.Reset()
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				select {
				case <-f.ctx.Done():
					return
				default:
				}
				txt := scanner.Text()
				entry.WriteString(txt)
				entry.WriteRune('\n')
				if txt == "" || strings.HasPrefix(txt, "#") {
					continue
				}
				select {
				case <-f.ctx.Done():
					return
				case f.entries <- entry.String():
					entry.Reset()
				}
			}
		}
	}
}

func (f *urlOpenMetricsEntryFetcher) Entries() <-chan string {
	return f.entries
}

func (f *urlOpenMetricsEntryFetcher) Close() error {
	f.closed = true
	return nil
}
