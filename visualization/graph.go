package visualization

import (
	"bufio"
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/eldada/metrics-viewer/models"
	"strings"
)

type Graph interface {
}

type graph struct {
}

func NewGraph() *graph {
	return &graph{}
}

func (g *graph) SprintOnce(metrics models.Metrics) string {
	if len(metrics.Metrics) == 0 {
		return ""
	}

	chart := tm.NewLineChart(100, 20)

	data := new(tm.DataTable)
	data.AddColumn("Time")
	data.AddColumn(metrics.Name)

	firstTimestamp := metrics.Metrics[0].Timestamp.Unix()
	for _, metric := range metrics.Metrics {
		data.AddRow(float64(metric.Timestamp.Unix()-firstTimestamp), metric.Value)
	}

	stringBuilder := &strings.Builder{}
	tm.Output = bufio.NewWriter(stringBuilder)
	_, err := tm.Println(chart.Draw(data))
	if err != nil {
		fmt.Printf("Error while drawing: %v\n", err)
	}

	tm.Flush()

	return stringBuilder.String()
}
