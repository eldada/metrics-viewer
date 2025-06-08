package commands

import (
	"os"
	"path"
	"testing"

	"github.com/eldada/metrics-viewer/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parsePrintCmdConfig(t *testing.T) {
	defaultCliCtx := cliContextMock{
		stringFlags: map[string]string{
			"format": string(printer.OpenMetricsFormat),
		},
	}
	testParseCommonConfig(t, defaultCliCtx, func(ctx cliContext) (commonConfig, error) {
		return parsePrintCmdConfig(ctx)
	})
	testFilepath := path.Join(t.TempDir(), "foo")
	require.NoError(t, os.WriteFile(testFilepath, []byte("hello"), 0777))
	defaultCliCtx.stringFlags["file"] = testFilepath
	tests := []struct {
		name    string
		cliCtx  cliContextMock
		want    printConfiguration
		wantErr string
	}{
		{
			name: "output format unknown",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"format": "foo",
				},
			},
			wantErr: "unknown output format: foo",
		},
		{
			name: "output format default",
			want: printConfiguration{
				format: printer.OpenMetricsFormat,
			},
		},
		{
			name: "output format csv, no metrics",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"format": string(printer.CSVFormat),
				},
			},
			wantErr: "--metrics is required when output format is csv",
		},
		{
			name: "output format csv, with metrics",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"format":  string(printer.CSVFormat),
					"metrics": "foo,bar,baz",
				},
			},
			want: printConfiguration{
				format:  printer.CSVFormat,
				metrics: []string{"foo", "bar", "baz"},
			},
		},
		{
			name: "no-header",
			cliCtx: cliContextMock{
				boolFlags: map[string]bool{
					"no-header": true,
				},
			},
			want: printConfiguration{
				format:   printer.OpenMetricsFormat,
				noHeader: true,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cliCtx := defaultCliCtx.OverrideWith(tc.cliCtx)
			conf, err := parsePrintCmdConfig(cliCtx)
			if tc.wantErr != "" {
				require.NotNil(t, err, "error")
				assert.Equal(t, tc.wantErr, err.Error(), "error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want.Format(), conf.Format(), "format")
			assert.Equal(t, tc.want.Metrics(), conf.Metrics(), "metrics")
			assert.Equal(t, tc.want.NoHeader(), conf.NoHeader(), "no-header")
			assert.Equal(t, os.Stdout, conf.Writer(), "writer")
		})
	}
}

func Test_splitCommaSeparatedMetricsNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty",
			input:    "",
			expected: nil,
		},
		{
			name:     "single",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:     "single with labels",
			input:    `foo{bar="hello",baz="world"}`,
			expected: []string{`foo{bar="hello",baz="world"}`},
		},
		{
			name:     "multiple",
			input:    "foo,bar,baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "multiple with labels",
			input:    `foo{bar="hello",baz="world"},hello,bla{abc="123",qwerty="3.1415"}`,
			expected: []string{`foo{bar="hello",baz="world"}`, "hello", `bla{abc="123",qwerty="3.1415"}`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := splitCommaSeparatedMetricsNames(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
