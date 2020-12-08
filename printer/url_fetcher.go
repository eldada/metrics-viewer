package printer

import (
	"bufio"
	"bytes"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"strings"
	"time"
)

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
