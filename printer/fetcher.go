package printer

import (
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"time"
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

type MetricEntryFetcher interface {
	Entries() <-chan string
	Close() error
}

var sleepFunc = time.Sleep

func sleep(d time.Duration) {
	sleepFunc(d)
}
