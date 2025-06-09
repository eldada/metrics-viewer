package commands

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eldada/metrics-viewer/provider"
	"github.com/jfrog/jfrog-cli-core/v2/common/commands"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
)

var FileFlag = components.NewStringFlag("file", "Log file with the open metrics format")

var UrlFlag = components.NewStringFlag("url", "Url endpoint to use to get metrics")

var UserFlag = components.NewStringFlag("user", "Username for url requiring authentication (see --password)")

var PasswordFlag = components.NewStringFlag("password", "Password for url requiring authentication (see --user)")

var TokenFlag = components.NewStringFlag("token", "Access token for url requiring authentication")

var ServerFlag = components.NewStringFlag("server-id", "Artifactory server ID to use from JFrog CLI configuration (use default if not set)")

var IntervalFlag = components.StringFlag{
	BaseFlag:     components.NewFlag("interval", "Scraping interval in seconds"),
	DefaultValue: "5",
}

var FilterFlag = components.NewStringFlag("filter", "Regular expression to use for filtering the metrics")

var AggregateIgnoreLabelsFlag = components.StringFlag{
	BaseFlag:     components.NewFlag("aggregate-ignore-labels", "Comma delimited list of labels to ignore when aggregating metrics. Use 'ALL' or 'NONE' to ignore all or none of the labels."),
	DefaultValue: "start,end,status",
}

func getCommonFlags() []components.Flag {
	return []components.Flag{
		FileFlag,
		UrlFlag,
		UserFlag,
		PasswordFlag,
		TokenFlag,
		ServerFlag,
		IntervalFlag,
		FilterFlag,
		AggregateIgnoreLabelsFlag,
	}
}

type commonConfiguration struct {
	file                  string
	urlMetricsFetcher     provider.UrlMetricsFetcher
	interval              time.Duration
	filter                *regexp.Regexp
	aggregateIgnoreLabels provider.StringSet
}

func (c commonConfiguration) UrlMetricsFetcher() provider.UrlMetricsFetcher {
	return c.urlMetricsFetcher
}

func (c commonConfiguration) File() string {
	return c.file
}

func (c commonConfiguration) Interval() time.Duration {
	return c.interval
}

func (c commonConfiguration) Filter() *regexp.Regexp {
	return c.filter
}

func (c commonConfiguration) AggregateIgnoreLabels() provider.StringSet {
	return c.aggregateIgnoreLabels
}

func (c commonConfiguration) String() string {
	return fmt.Sprintf("file: '%s', %s, interval: %s, filter: %s",
		c.file, c.urlMetricsFetcher, c.interval, c.filter.String())
}

func parseCommonConfig(c cliContext) (*commonConfiguration, error) {
	conf := commonConfiguration{
		file: c.GetStringFlagValue("file"),
	}
	url := c.GetStringFlagValue("url")

	countInputFlags := 0
	if conf.file != "" {
		countInputFlags++
	}
	if url != "" {
		countInputFlags++
	}

	if countInputFlags > 1 {
		return nil, fmt.Errorf("only zero or one flag is required: --file | --url")
	}

	if conf.file != "" {
		f, err := os.Open(conf.file)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", conf.file, err)
		}
		_ = f.Close()
	}

	if countInputFlags == 0 {
		serverId := c.GetStringFlagValue("server-id")
		rtDetails, err := commands.GetConfig(serverId, false)
		if err != nil {
			msg := fmt.Sprintf("could not load configuration for Artifactory server %s", serverId)
			if serverId == "" {
				msg = "could not load configuration for current Artifactory server"
			}
			return nil, fmt.Errorf("%s; cause: %w", msg, err)
		}
		conf.urlMetricsFetcher, err = provider.NewArtifactoryMetricsFetcher(rtDetails)
		if err != nil {
			return nil, fmt.Errorf("could not initiate metrics fetcher from Artifactory; cause: %w", err)
		}
	}

	if url != "" {
		var authenticator provider.Authenticator
		username := c.GetStringFlagValue("user")
		password := c.GetStringFlagValue("password")
		if username != "" {
			authenticator = provider.UserPassAuthenticator{
				Username: username,
				Password: password,
			}
		}
		token := c.GetStringFlagValue("token")
		if token != "" {
			if authenticator != nil {
				return nil, fmt.Errorf("cannot use both user-password credentials and an access token; choose one")
			}
			authenticator = provider.AccessTokenAuthenticator{
				Token: token,
			}
		}
		conf.urlMetricsFetcher = provider.NewUrlMetricsFetcher(url, authenticator)
	}

	var flagValue string

	flagValue = c.GetStringFlagValue("interval")
	intValue, err := strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse interval value: %s; cause: %w", flagValue, err)
	}
	if intValue <= 0 {
		return nil, fmt.Errorf("interval value must be positive; got: %d", intValue)
	}
	conf.interval = time.Duration(intValue) * time.Second

	flagValue = c.GetStringFlagValue("filter")
	if flagValue != "" {
		conf.filter, err = regexp.Compile(flagValue)
		if err != nil {
			return nil, fmt.Errorf("invalid filter expression; cause: %w", err)
		}
	}

	flagValue = c.GetStringFlagValue("aggregate-ignore-labels")
	conf.aggregateIgnoreLabels = provider.StringSet{}
	if flagValue != "" {
		conf.aggregateIgnoreLabels.Add(strings.Split(flagValue, ",")...)
	}

	return &conf, nil
}

type cliContext interface {
	GetStringFlagValue(flagName string) string
	GetBoolFlagValue(flagName string) bool
}
