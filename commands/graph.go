package commands

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/eldada/metrics-viewer/visualization"
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
			Name:         "interval",
			Description:  "scraping interval in seconds",
			DefaultValue: "5",
		},
		components.StringFlag{
			Name:         "time",
			Description:  "time window to show in seconds",
			DefaultValue: "60",
		},
		components.StringFlag{
			Name:         "metric",
			Description:  "comma delimited list of metrics to show (default: all)",
			DefaultValue: "",
		},
	}
}

type graphConfiguration struct {
	file       string
	url        string
	interval   time.Duration
	timeWindow time.Duration
	metrics    []string
}

func (c graphConfiguration) Url() string {
	return c.url
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

func graphCmd(c *components.Context) error {
	conf, err := parseGraphCmdConfig(c)
	if err != nil {
		return err
	}
	//TODO Change to debug
	log.Info(fmt.Sprintf("file: '%s', url: '%s', interval: %s, time: %s, metrics: %s",
		conf.file, conf.url, conf.interval, conf.timeWindow, conf.metrics))

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
		url:  c.GetStringFlagValue("url"),
	}

	if conf.file == "" && conf.url == "" && os.Getenv("MOCK_METRICS_DATA") == "" {
		return nil, fmt.Errorf("one flag is required: file | url")
	}
	if conf.file != "" && conf.url != "" {
		return nil, fmt.Errorf("only one flag is required: file | url")
	}

	if conf.file != "" {
		f, err := os.Open(conf.file)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", conf.file, err)
		}
		_ = f.Close()
	}

	flagValue := c.GetStringFlagValue("interval")
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

	return &conf, nil
}
