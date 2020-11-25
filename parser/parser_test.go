package parser

import (
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestParseMetrics(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "multiple gauge metrics, single value each",
			inputFile:    "testdata/metrics1.log",
			expectedFile: "testdata/metrics1-expected.txt",
		},
		{
			name:         "multiple gauge metrics, multiple values each",
			inputFile:    "testdata/metrics-gauge-multi.log",
			expectedFile: "testdata/metrics-gauge-multi-expected.txt",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metricsFile, err := os.Open(tc.inputFile)
			require.NoError(t, err, "could not open input file")
			defer metricsFile.Close()
			expected, err := ioutil.ReadFile(tc.expectedFile)
			require.NoError(t, err, "could not read expected results file")
			metrics, err := ParseMetrics(metricsFile)
			require.NoError(t, err, "unexpected error while parsing metrics")
			//fmt.Println(metricsToString(metrics))
			assert.Equal(t, string(expected), metricsToString(metrics))
		})
	}
}

func metricsToString(metricsCollection []models.Metrics) string {
	s := strings.Builder{}
	for _, metrics := range metricsCollection {
		_, _ = fmt.Fprintf(&s, "%s:%s\n", metrics.Name, metrics.Description)
		for _, metric := range metrics.Metrics {
			_, _ = fmt.Fprintf(&s, "  %s %.3f\n", metric.Timestamp.UTC().Format("2006-01-02T15:04:05.000"), metric.Value)
		}
	}
	return s.String()
}
