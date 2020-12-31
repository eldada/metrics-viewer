package printer

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test_urlFetcher(t *testing.T) {
	metricsFetcher := &metricsFetcherMock{}
	f, err := newUrlOpenMetricsEntryFetcher(metricsFetcher, time.Millisecond)
	require.NoError(t, err)
	s := strings.Builder{}
	lastUpdate := time.Now()
	gofuncStopped := false
	go func() {
		for entry := range f.Entries() {
			s.WriteString(entry)
			lastUpdate = time.Now()
		}
		gofuncStopped = true
	}()
	for {
		if time.Since(lastUpdate) > 100*time.Millisecond {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	f.Close()
	expected, _ := ioutil.ReadFile("testdata/metrics1_2_3.log")
	assert.Equal(t, string(expected), s.String())
	assert.True(t, gofuncStopped, "iteration stopped")

}

type metricsFetcherMock struct {
	mu      sync.Mutex
	counter int
}

func (f *metricsFetcherMock) Get() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counter++
	if f.counter > 3 {
		return nil, nil
	}
	filename := fmt.Sprintf("testdata/metrics%d.log", f.counter)
	return ioutil.ReadFile(filename)
}
