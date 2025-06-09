package printer

import (
	"context"
	"strings"

	"github.com/hpcloud/tail"
)

func newFileOpenMetricEntryFetcher(filename string) (*fileOpenMetricEntryFetcher, error) {
	return newFileOpenMetricEntryFetcherWithContext(context.Background(), filename)
}

func newFileOpenMetricEntryFetcherWithContext(ctx context.Context, filename string) (*fileOpenMetricEntryFetcher, error) {
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
		ctx:     ctx,
	}
	go fetcher.fetch()
	return &fetcher, nil
}

type fileOpenMetricEntryFetcher struct {
	tail    *tail.Tail
	entries chan string
	closed  bool
	ctx     context.Context
}

func (f *fileOpenMetricEntryFetcher) fetch() {
	defer close(f.entries)
	entry := strings.Builder{}
	for line := range f.tail.Lines {
		select {
		case <-f.ctx.Done():
			return
		default:
		}
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
		select {
		case <-f.ctx.Done():
			return
		case f.entries <- entry.String():
			entry.Reset()
		}
	}
}

func (f *fileOpenMetricEntryFetcher) Close() error {
	f.closed = true
	return f.tail.Stop()
}

func (f *fileOpenMetricEntryFetcher) Entries() <-chan string {
	return f.entries
}
