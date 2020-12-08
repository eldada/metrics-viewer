package printer

import (
	"github.com/hpcloud/tail"
	"strings"
)

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
