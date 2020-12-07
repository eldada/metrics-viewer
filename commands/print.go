package commands

import (
	"fmt"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/eldada/metrics-viewer/printer"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/jfrog/jfrog-cli-core/plugins/components"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"io"
	"os"
	"strings"
)

func GetPrintCommand() components.Command {
	return components.Command{
		Name:        "print",
		Description: "Easily print and tail Open Metrics data in terminal",
		Aliases:     []string{"p"},
		Flags:       getPrintFlags(),
		Action: func(c *components.Context) error {
			return printCmd(c)
		},
	}
}

func getPrintFlags() []components.Flag {
	return append(
		getCommonFlags(),
		components.StringFlag{
			Name:         "format",
			Description:  "format in which to print the metrics (available: open-metrics, csv)",
			DefaultValue: "open-metrics",
		},
		components.StringFlag{
			Name:        "metrics",
			Description: "comma separated list of metrics to collect. This is required when the output format is csv",
		},
	)
}

type printConfiguration struct {
	commonConfiguration
	format  printer.OutputFormat
	metrics []string
}

func (c printConfiguration) Format() printer.OutputFormat {
	return c.format
}

func (c printConfiguration) Metrics() []string {
	return c.metrics
}

func (c printConfiguration) Writer() io.Writer {
	return os.Stdout
}

func (c printConfiguration) String() string {
	return fmt.Sprintf("%s, format: %s", c.commonConfiguration, c.format)
}

func printCmd(c *components.Context) error {
	conf, err := parsePrintCmdConfig(c)
	if err != nil {
		return err
	}
	log.Debug("command config:", conf)

	fetcher, err := printer.NewFetcher(conf)
	if err != nil {
		return err
	}
	defer fetcher.Close()
	p, err := printer.NewPrinter(conf)
	if err != nil {
		return err
	}
	shouldPrintEntry := getFilterFunc(conf)

	for entry := range fetcher.Entries() {
		if shouldPrintEntry(entry) {
			_ = p.Print(entry)
		}
	}

	return nil
}

func parsePrintCmdConfig(c *components.Context) (*printConfiguration, error) {
	commonConfig, err := parseCommonConfig(c)
	if err != nil {
		return nil, err
	}
	conf := printConfiguration{
		commonConfiguration: *commonConfig,
	}

	flagValue := c.GetStringFlagValue("format")
	if format, ok := printer.SupportedOutputFormats[flagValue]; ok {
		conf.format = format
	} else {
		return nil, fmt.Errorf("unknown output format: %s", flagValue)
	}

	flagValue = c.GetStringFlagValue("metrics")
	if flagValue == "" && conf.format == printer.CSVFormat {
		return nil, fmt.Errorf("--metrics is required when output format is csv")
	}
	conf.metrics = strings.Split(flagValue, ",")

	return &conf, nil
}

func getFilterFunc(conf printer.Config) func(entry string) bool {
	filter := conf.Filter()
	if filter == nil {
		return func(entry string) bool {
			return true
		}
	}
	mapMetrics := provider.NewLabelsMetricsMapper(conf.AggregateIgnoreLabels(), "|")
	return func(entry string) bool {
		metrics, err := parser.ParseMetrics(strings.NewReader(entry))
		if err != nil {
			return false
		}
		metrics = mapMetrics(metrics)
		for _, m := range metrics {
			if filter.MatchString(m.Name) {
				return true
			}
		}
		return false
	}
}
