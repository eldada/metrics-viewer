package commands

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"testing"
	"time"
)

func Test_parseGraphCmdConfig(t *testing.T) {
	defaultCliCtx := cliContextMock{
		stringFlags: map[string]string{
			"time": "5",
		},
	}
	testParseCommonConfig(t, defaultCliCtx, func(ctx cliContext) (commonConfig, error) {
		return parseGraphCmdConfig(ctx)
	})
	testFilepath := path.Join(t.TempDir(), "foo")
	require.NoError(t, ioutil.WriteFile(testFilepath, []byte("hello"), 0777))
	defaultCliCtx.stringFlags["file"] = testFilepath
	tests := []struct {
		name    string
		cliCtx  cliContextMock
		want    graphConfiguration
		wantErr string
	}{
		{
			name: "time window",
			want: graphConfiguration{
				timeWindow: 5 * time.Second,
			},
		},
		{
			name: "zero time window",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"time": "0",
				},
			},
			wantErr: "time window value must be positive; got: 0",
		},
		{
			name: "negative time window",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"time": "-3",
				},
			},
			wantErr: "time window value must be positive; got: -3",
		},
		{
			name: "time window NaN",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"time": "foo",
				},
			},
			wantErr: `failed to parse time window value: foo; cause: strconv.ParseInt: parsing "foo": invalid syntax`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cliCtx := defaultCliCtx.OverrideWith(tc.cliCtx)
			conf, err := parseGraphCmdConfig(cliCtx)
			if tc.wantErr != "" {
				require.NotNil(t, err, "error")
				assert.Equal(t, tc.wantErr, err.Error(), "error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want.TimeWindow(), conf.TimeWindow(), "time window")
		})
	}
}
