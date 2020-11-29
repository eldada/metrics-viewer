package commands

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/eldada/metrics-viewer/visualization"
	"github.com/jfrog/jfrog-cli-core/artifactory/commands"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetGraphCommand() components.Command {
	return components.Command{
		Name:        "graph",
		Description: "Easily graph Open Metrics data in terminal",
		Aliases:     []string{"g"},
		Flags:       getGraphFlags(),
		Action: func(c *components.Context) error {
			return graphCmd(c)
		},
	}
}

func getGraphFlags() []components.Flag {
	return []components.Flag{
		components.StringFlag{
			Name:        "file",
			Description: "log file with the open metrics format",
		},
		components.StringFlag{
			Name:        "url",
			Description: "url endpoint to use to get metrics",
		},
		components.StringFlag{
			Name:        "user",
			Description: "username for url requiring authentication",
		},
		components.StringFlag{
			Name:        "password",
			Description: "password for url requiring authentication",
		},
		components.BoolFlag{
			Name:         "artifactory",
			Description:  "call Artifactory to get the metrics",
			DefaultValue: false,
		},
		components.StringFlag{
			Name:        "server",
			Description: "Artifactory server ID to call when --artifactory is given (uses current by default)",
		},
		components.StringFlag{
			Name:         "interval",
			Description:  "scraping interval in seconds",
			DefaultValue: "5",
		},
		components.StringFlag{
			Name:         "time",
			Description:  "time window to show in seconds",
			DefaultValue: "300",
		},
		components.StringFlag{
			Name:         "metrics",
			Description:  "comma delimited list of metrics to show (default: all)",
			DefaultValue: "",
		},
		components.StringFlag{
			Name:         "aggregate-ignore-labels",
			Description:  "comma delimited list of labels to ignore when aggregating metrics. Use 'ALL' or 'NONE' to ignore all or none of the labels.",
			DefaultValue: "start,end,status",
		},
	}
}

type graphConfiguration struct {
	file                  string
	urlMetricsFetcher     provider.UrlMetricsFetcher
	interval              time.Duration
	timeWindow            time.Duration
	metrics               []string
	aggregateIgnoreLabels provider.StringSet
}

func (c graphConfiguration) UrlMetricsFetcher() provider.UrlMetricsFetcher {
	return c.urlMetricsFetcher
}

func (c graphConfiguration) File() string {
	return c.file
}

func (c graphConfiguration) TimeWindow() time.Duration {
	return c.timeWindow
}

func (c graphConfiguration) MetricKeys() []string {
	return c.metrics
}

func (c graphConfiguration) AggregateIgnoreLabels() provider.StringSet {
	return c.aggregateIgnoreLabels
}

func (c graphConfiguration) String() string {
	return fmt.Sprintf("file: '%s', %s, interval: %s, time: %s, metrics: %s",
		c.file, c.urlMetricsFetcher, c.interval, c.timeWindow, c.metrics)
}

func graphCmd(c *components.Context) error {
	conf, err := parseGraphCmdConfig(c)
	if err != nil {
		return err
	}
	log.Debug("command config:", conf)

	prov, err := provider.New(conf)
	if err != nil {
		return err
	}

	visualization.NewIndex().Present(context.TODO(), conf.interval, prov)
	return nil
}

func parseGraphCmdConfig(c *components.Context) (*graphConfiguration, error) {
	conf := graphConfiguration{
		file: c.GetStringFlagValue("file"),
	}
	url := c.GetStringFlagValue("url")
	callArtifactory := c.GetBoolFlagValue("artifactory")

	countInputFlags := 0
	if conf.file != "" {
		countInputFlags++
	}
	if url != "" {
		countInputFlags++
	}
	if callArtifactory {
		countInputFlags++
	}
	if countInputFlags == 0 && os.Getenv("MOCK_METRICS_DATA") == "" {
		return nil, fmt.Errorf("one flag is required: --file | --url | --artifactory")
	}
	if countInputFlags > 1 {
		return nil, fmt.Errorf("only one flag is required: --file | --url | --artifactory")
	}

	if conf.file != "" {
		f, err := os.Open(conf.file)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", conf.file, err)
		}
		_ = f.Close()
	}

	if callArtifactory {
		serverId := c.GetStringFlagValue("server")
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
		username := c.GetStringFlagValue("user")
		password := c.GetStringFlagValue("password")
		conf.urlMetricsFetcher = provider.NewUrlMetricsFetcher(url, username, password)
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

	flagValue = c.GetStringFlagValue("time")
	intValue, err = strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time window value: %s; cause: %w", flagValue, err)
	}
	if intValue <= 0 {
		return nil, fmt.Errorf("time window value must be positive; got: %d", intValue)
	}
	conf.timeWindow = time.Duration(intValue) * time.Second

	flagValue = c.GetStringFlagValue("metrics")
	if flagValue != "" {
		conf.metrics = strings.Split(flagValue, ",")
	}

	flagValue = c.GetStringFlagValue("aggregate-ignore-labels")
	conf.aggregateIgnoreLabels = provider.StringSet{}
	if flagValue != "" {
		conf.aggregateIgnoreLabels.Add(strings.Split(flagValue, ",")...)
	}

	return &conf, nil
}
