package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

func Test_fileProvider(t *testing.T) {
	p, err := newFileProvider("testdata/metrics1.log", 100*time.Millisecond)
	require.NoError(t, err)
	defer p.Close()
	metrics, err := p.Get()
	require.NoError(t, err)
	actual := metricsToString(metrics)
	expected, _ := ioutil.ReadFile("testdata/metrics1.txt")
	assert.Equal(t, string(expected), actual)
}
