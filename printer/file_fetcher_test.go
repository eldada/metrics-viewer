package printer

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func Test_fileFetcher(t *testing.T) {
	filename := path.Join(t.TempDir(), "result.log")
	f, err := newFileOpenMetricEntryFetcher(filename)
	require.NoError(t, err)
	s := strings.Builder{}
	lastUpdate := time.Now()
	gofuncStopped := false
	go func() {
		for entry := range f.Entries() {
			s.WriteString(entry)
			lastUpdate = time.Now()
		}
		t.Log("Iteration stopped.")
		gofuncStopped = true
	}()
	go func() {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
		defer file.Close()
		require.NoError(t, err)
		for i := 1; i <= 3; i++ {
			data, _ := ioutil.ReadFile(fmt.Sprintf("testdata/metrics%d.log", i))
			fmt.Fprintln(file, string(data))
		}
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
