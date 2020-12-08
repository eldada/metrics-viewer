package printer

import (
	"github.com/eldada/metrics-viewer/provider"
	"github.com/stretchr/testify/assert"
	"io"
	"regexp"
	"strings"
	"testing"
)

func Test_csvPrinter_Print(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
		config   configMock
		expected string
	}{
		{
			name: "single metric",
			entries: []string{
				"jfrt_runtime_heap_freememory_bytes 2.319814e+08 1606343802324",
				"jfrt_runtime_heap_freememory_bytes 2.645814e+08 1606343813456",
				"jfrt_runtime_heap_freememory_bytes 2.356814e+08 1606343834567",
				"jfrt_runtime_heap_freememory_bytes 2.234814e+08 1606343845678",
			},
			config: configMock{
				metrics: []string{"jfrt_runtime_heap_freememory_bytes"},
			},
			expected: `timestamp,jfrt_runtime_heap_freememory_bytes
2020-11-25T22:36:42.324,231981400.000000
2020-11-25T22:36:53.456,264581400.000000
2020-11-25T22:37:14.567,235681400.000000
2020-11-25T22:37:25.678,223481400.000000
`,
		},
		{
			name: "single metric, no header",
			entries: []string{
				"jfrt_runtime_heap_freememory_bytes 2.319814e+08 1606343802324",
				"jfrt_runtime_heap_freememory_bytes 2.645814e+08 1606343813456",
				"jfrt_runtime_heap_freememory_bytes 2.356814e+08 1606343834567",
				"jfrt_runtime_heap_freememory_bytes 2.234814e+08 1606343845678",
			},
			config: configMock{
				metrics:  []string{"jfrt_runtime_heap_freememory_bytes"},
				noHeader: true,
			},
			expected: `2020-11-25T22:36:42.324,231981400.000000
2020-11-25T22:36:53.456,264581400.000000
2020-11-25T22:37:14.567,235681400.000000
2020-11-25T22:37:25.678,223481400.000000
`,
		},
		{
			name: "multiple metrics with labels",
			entries: []string{
				"jfrt_runtime_heap_freememory_bytes 2.319814e+08 1606343802324",
				"jfrt_runtime_heap_maxmemory_bytes 2.147484e+09 1606343802324",
				"app_disk_free_bytes 5.685859e+10 1606343802324",
				`foo{bar="hello",baz="world",bla="123"} 7.1234343e+05 1606343802324`,

				"jfrt_runtime_heap_freememory_bytes 2.645814e+08 1606343813456",
				"jfrt_runtime_heap_maxmemory_bytes 2.234534e+09 1606343813456",
				"app_disk_free_bytes 5.4523455e+10 1606343813456",
				`foo{bar="hello",baz="world",bla="234"} 4.1234343e+05 1606343813456`,

				"jfrt_runtime_heap_freememory_bytes 2.356814e+08 1606343834567",
				"jfrt_runtime_heap_maxmemory_bytes 1.147484e+09 1606343834567",
				"app_disk_free_bytes 5.3235345e+10 1606343834567",
				`foo{bar="hello",baz="world",bla="345"} 5.1234343e+05 1606343834567`,

				"jfrt_runtime_heap_freememory_bytes 2.234814e+08 1606343845678",
				"jfrt_runtime_heap_maxmemory_bytes 3.147484e+09 1606343845678",
				"app_disk_free_bytes 5.452345e+10 1606343845678",
				`foo{bar="hello",baz="world",bla="456"} 6.1234343e+05 1606343845678`,
			},
			config: configMock{
				metrics: []string{
					"jfrt_runtime_heap_freememory_bytes",
					"jfrt_runtime_heap_maxmemory_bytes",
					`foo{bar="hello",baz="world"}`,
				},
				aggregateIgnoreLabels: provider.StringSet{
					"bla": struct{}{},
				},
			},
			expected: `timestamp,jfrt_runtime_heap_freememory_bytes,jfrt_runtime_heap_maxmemory_bytes,"foo{bar=""hello"",baz=""world""}"
2020-11-25T22:36:42.324,231981400.000000,2147484000.000000,712343.430000
2020-11-25T22:36:53.456,264581400.000000,2234534000.000000,412343.430000
2020-11-25T22:37:14.567,235681400.000000,1147484000.000000,512343.430000
2020-11-25T22:37:25.678,223481400.000000,3147484000.000000,612343.430000
`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := strings.Builder{}
			tc.config.writer = &out
			p := newCSVPrinter(tc.config)
			for _, entry := range tc.entries {
				p.Print(entry)
			}
			p.flushLastRecord()
			assert.Equal(t, tc.expected, out.String())
		})
	}
}

type configMock struct {
	filter                *regexp.Regexp
	aggregateIgnoreLabels provider.StringSet
	format                OutputFormat
	writer                io.Writer
	metrics               []string
	noHeader              bool
}

func (c configMock) Filter() *regexp.Regexp {
	return c.filter
}

func (c configMock) AggregateIgnoreLabels() provider.StringSet {
	return c.aggregateIgnoreLabels
}

func (c configMock) Format() OutputFormat {
	return c.format
}

func (c configMock) Writer() io.Writer {
	return c.writer
}

func (c configMock) Metrics() []string {
	return c.metrics
}

func (c configMock) NoHeader() bool {
	return c.noHeader
}
