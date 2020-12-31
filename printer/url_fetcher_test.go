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
		counter := 0
		for entry := range f.Entries() {
			counter++
			//t.Logf("Received entry #%d, length: %d", counter, len(entry))
			s.WriteString(entry)
			lastUpdate = time.Now()
			//t.Logf("Last updated at %s", lastUpdate.Format(time.RFC3339Nano))
		}
		t.Log("Iteration stopped.")
		gofuncStopped = true
	}()
	durWithoutEntries := 10 * time.Millisecond
	t.Logf("Waiting for at least %s without new entries...", durWithoutEntries)
	for {
		sinceLastUpdate := time.Since(lastUpdate)
		if sinceLastUpdate > durWithoutEntries {
			t.Log("Finished waiting.")
			break
		}
		t.Logf("Last entry since %s, waiting some more...", sinceLastUpdate)
		time.Sleep(5 * time.Millisecond)
	}
	f.Close()
	expected, _ := ioutil.ReadFile("testdata/metrics1_2_3.log")
	assert.Equal(t, string(expected), s.String())
	assert.Eventually(t, func() bool { return gofuncStopped }, 100*time.Millisecond, 5*time.Millisecond, "Iteration did not stop")

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
