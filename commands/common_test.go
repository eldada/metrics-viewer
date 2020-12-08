package commands

import (
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"regexp"
	"testing"
	"time"
)

func Test_parseCommonConfig(t *testing.T) {
	testParseCommonConfig(t, cliContextMock{}, func(ctx cliContext) (commonConfig, error) {
		return parseCommonConfig(ctx)
	})
}

func testParseCommonConfig(t *testing.T, defaultCliCtx cliContextMock, parse func(ctx cliContext) (commonConfig, error)) {
	if defaultCliCtx.stringFlags == nil {
		defaultCliCtx.stringFlags = make(map[string]string, 1)
	}
	defaultCliCtx.stringFlags["interval"] = "5"
	testFilepath := path.Join(t.TempDir(), "foo")
	require.NoError(t, ioutil.WriteFile(testFilepath, []byte("hello"), 0777))
	tests := []struct {
		name    string
		cliCtx  cliContextMock
		want    commonConfig
		wantUrl string
		wantErr string
	}{
		{
			name:    "no options",
			wantErr: "one flag is required: --file | --url | --artifactory",
		},
		{
			name: "both file and url",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"file": "foo",
					"url":  "boo",
				},
			},
			wantErr: "only one flag is required: --file | --url | --artifactory",
		},
		{
			name: "both file and artifactory",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"file": "foo",
				},
				boolFlags: map[string]bool{
					"artifactory": true,
				},
			},
			wantErr: "only one flag is required: --file | --url | --artifactory",
		},
		{
			name: "both url and artifactory",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url": "boo",
				},
				boolFlags: map[string]bool{
					"artifactory": true,
				},
			},
			wantErr: "only one flag is required: --file | --url | --artifactory",
		},
		{
			name: "file",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"file":     testFilepath,
					"interval": "5",
				},
			},
			want: commonConfiguration{
				file:                  testFilepath,
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
			},
		},
		{
			name: "no such file",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"file": "foo",
				},
			},
			wantErr: "could not open file foo: open foo: no such file or directory",
		},
		{
			name: "url without auth",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"interval": "5",
				},
			},
			want: commonConfiguration{
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
			},
			wantUrl: "url: foo",
		},
		{
			name: "url with basic auth",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"user":     "kermit",
					"password": "strong-password",
					"interval": "5",
				},
			},
			want: commonConfiguration{
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
			},
			wantUrl: "url: foo, auth-by-user: kermit",
		},
		{
			name: "url with token auth",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"token":    "bar",
					"interval": "5",
				},
			},
			want: commonConfiguration{
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
			},
			wantUrl: "url: foo, auth-by-token: *****",
		},
		{
			name: "url with both basic auth and token auth",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"user":     "kermit",
					"password": "strong-password",
					"token":    "bar",
					"interval": "5",
				},
			},
			want: commonConfiguration{
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
			},
			wantErr: "cannot use both user-password credentials and an access token; choose one",
		},
		{
			name: "interval is NaN",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"interval": "bar",
				},
			},
			wantErr: `failed to parse interval value: bar; cause: strconv.ParseInt: parsing "bar": invalid syntax`,
		},
		{
			name: "interval is zero",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"interval": "0",
				},
			},
			wantErr: "interval value must be positive; got: 0",
		},
		{
			name: "interval is negative",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":      "foo",
					"interval": "-7",
				},
			},
			wantErr: "interval value must be positive; got: -7",
		},
		{
			name: "filter",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":    "foo",
					"filter": "foo.*",
				},
			},
			want: commonConfiguration{
				interval:              5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{},
				filter:                regexp.MustCompile("foo.*"),
			},
			wantUrl: "url: foo",
		},
		{
			name: "filter with bad regex",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":    "foo",
					"filter": "(",
				},
			},
			wantErr: "invalid filter expression; cause: error parsing regexp: missing closing ): `(`",
		},
		{
			name: "aggregate ignore labels",
			cliCtx: cliContextMock{
				stringFlags: map[string]string{
					"url":                     "foo",
					"aggregate-ignore-labels": "foo,bar,baz",
				},
			},
			want: commonConfiguration{
				interval: 5 * time.Second,
				aggregateIgnoreLabels: provider.StringSet{
					"foo": {},
					"bar": {},
					"baz": {},
				},
			},
			wantUrl: "url: foo",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cliCtx := defaultCliCtx.OverrideWith(tc.cliCtx)
			conf, err := parse(cliCtx)
			if tc.wantErr != "" {
				require.NotNil(t, err, "error")
				assert.Equal(t, tc.wantErr, err.Error(), "error")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want.Filter(), conf.Filter(), "filter")
			assert.Equal(t, tc.want.File(), conf.File(), "file")
			assert.Equal(t, tc.want.AggregateIgnoreLabels(), conf.AggregateIgnoreLabels(), "aggregate ignore labels")
			assert.Equal(t, tc.want.Interval(), conf.Interval(), "interval")
			if tc.wantUrl == "" {
				assert.Nil(t, conf.UrlMetricsFetcher(), "url metrics fetcher")
			} else {
				if assert.NotNil(t, conf.UrlMetricsFetcher(), "url metrics fetcher") {
					assert.Equal(t, tc.wantUrl, fmt.Sprintf("%s", conf.UrlMetricsFetcher()), "url metrics fetcher")
				}
			}
		})
	}
}

type cliContextMock struct {
	Arguments   []string
	stringFlags map[string]string
	boolFlags   map[string]bool
}

func (c cliContextMock) GetStringFlagValue(flagName string) string {
	return c.stringFlags[flagName]
}

func (c cliContextMock) GetBoolFlagValue(flagName string) bool {
	return c.boolFlags[flagName]
}

func (c cliContextMock) OverrideWith(other cliContextMock) cliContextMock {
	newCtx := cliContextMock{}
	newCtx.stringFlags = make(map[string]string)
	for k, v := range c.stringFlags {
		newCtx.stringFlags[k] = v
	}
	for k, v := range other.stringFlags {
		newCtx.stringFlags[k] = v
	}
	newCtx.boolFlags = make(map[string]bool)
	for k, v := range c.boolFlags {
		newCtx.boolFlags[k] = v
	}
	for k, v := range other.boolFlags {
		newCtx.boolFlags[k] = v
	}
	return newCtx
}

type commonConfig interface {
	UrlMetricsFetcher() provider.UrlMetricsFetcher
	File() string
	Interval() time.Duration
	Filter() *regexp.Regexp
	AggregateIgnoreLabels() provider.StringSet
}
