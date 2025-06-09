package printer

import (
	"context"
	"fmt"
	"time"

	"github.com/eldada/metrics-viewer/provider"
)

type FetcherConfig interface {
	UrlMetricsFetcher() provider.UrlMetricsFetcher
	File() string
	Interval() time.Duration
}

func NewFetcher(conf FetcherConfig) (MetricEntryFetcher, error) {
	if conf.File() != "" {
		return newFileOpenMetricEntryFetcher(conf.File())
	}
	if conf.UrlMetricsFetcher() != nil {
		return newUrlOpenMetricsEntryFetcher(conf.UrlMetricsFetcher(), conf.Interval())
	}
	return nil, fmt.Errorf("illegal state, could not create fetcher - file or url are mandatory")
}

func NewFetcherWithContext(ctx context.Context, conf FetcherConfig) (MetricEntryFetcher, error) {
	if conf.File() != "" {
		return newFileOpenMetricEntryFetcherWithContext(ctx, conf.File())
	}
	if conf.UrlMetricsFetcher() != nil {
		return newUrlOpenMetricsEntryFetcherWithContext(ctx, conf.UrlMetricsFetcher(), conf.Interval())
	}
	return nil, fmt.Errorf("illegal state, could not create fetcher - file or url are mandatory")
}

type MetricEntryFetcher interface {
	Entries() <-chan string
	Close() error
}

var sleepFunc = time.Sleep

func sleep(d time.Duration) {
	sleepFunc(d)
}
