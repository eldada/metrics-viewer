package provider

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_urlProvider(t *testing.T) {
	metricsFetcher := metricsFetcherMock{
		filename: "testdata/metrics1.log",
	}
	p, err := newUrlProvider(metricsFetcher)
	require.NoError(t, err)
	metrics, err := p.Get()
	require.NoError(t, err)
	actual := metricsToString(metrics)
	expectedData, _ := os.ReadFile("testdata/metrics1_sorted.txt")
	expected := string(expectedData)
	assert.Equal(t, string(expected), actual)
}

type metricsFetcherMock struct {
	filename string
}

func (f metricsFetcherMock) Get() ([]byte, error) {
	return os.ReadFile(f.filename)
}
