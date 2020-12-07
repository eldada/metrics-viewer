package commands

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/eldada/metrics-viewer/visualization"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"strconv"
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
	return append(
		getCommonFlags(),
		components.StringFlag{
			Name:         "time",
			Description:  "Time window to show in seconds",
			DefaultValue: "300",
		},
	)
}

type graphConfiguration struct {
	commonConfiguration
	timeWindow time.Duration
}

func (c graphConfiguration) TimeWindow() time.Duration {
	return c.timeWindow
}

func (c graphConfiguration) String() string {
	return fmt.Sprintf("%s, time: %s", c.commonConfiguration, c.timeWindow)
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
	commonConfig, err := parseCommonConfig(c)
	if err != nil {
		return nil, err
	}
	conf := graphConfiguration{
		commonConfiguration: *commonConfig,
	}

	flagValue := c.GetStringFlagValue("time")
	intValue, err := strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time window value: %s; cause: %w", flagValue, err)
	}
	if intValue <= 0 {
		return nil, fmt.Errorf("time window value must be positive; got: %d", intValue)
	}
	conf.timeWindow = time.Duration(intValue) * time.Second

	return &conf, nil
}
