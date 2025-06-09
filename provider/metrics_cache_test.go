package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_filterByTimeWindow(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		timeWindow   time.Duration
		nowFunc      func() time.Time
		expectedFile string
	}{
		{
			name:         "empty result",
			inputFile:    "testdata/metrics-gauge-multi.log",
			timeWindow:   5 * time.Second,
			expectedFile: "testdata/metrics-gauge-multi-expected-filtered-empty.txt",
		},
		{
			name:       "filtered results - 3min",
			inputFile:  "testdata/metrics-gauge-multi.log",
			timeWindow: 3 * time.Minute,
			nowFunc: func() time.Time {
				return time.Date(2020, 11, 25, 23, 28, 0, 0, time.UTC)
			},
			expectedFile: "testdata/metrics-gauge-multi-expected-filtered-3min.txt",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.nowFunc != nil {
				defer func() {
					nowFunc = time.Now
				}()
				nowFunc = tc.nowFunc
			}
			metricsFile, err := os.Open(tc.inputFile)
			require.NoError(t, err, "could not open input file")
			defer metricsFile.Close()
			expected, err := os.ReadFile(tc.expectedFile)
			require.NoError(t, err, "could not read expected results file")
			metrics, err := parser.ParseMetrics(metricsFile)
			require.NoError(t, err, "unexpected error while parsing metrics")
			filtered := filterByTimeWindow(metrics, tc.timeWindow)
			assert.Equal(t, string(expected), metricsToString(filtered), "filtered metrics not as expected")
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
