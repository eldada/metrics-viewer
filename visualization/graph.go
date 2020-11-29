package visualization

import (
	"bufio"
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/eldada/metrics-viewer/models"
	"sort"
	"strings"
)

type Graph interface {
}

type graph struct {
}

func NewGraph() *graph {
	return &graph{}
}

func (g *graph) SprintOnce(width, height int, multipleMetrics ...models.Metrics) string {
	chart := tm.NewLineChart(width, height)

	if len(multipleMetrics) == 2 {
		chart.Flags = tm.DRAW_INDEPENDENT
	}

	data := new(tm.DataTable)
	data.AddColumn("Time")
	numberOfGraphs := 0
	timeData := map[float64]map[int]float64{} // time data -> graph id -> value
	for i, metrics := range multipleMetrics {
		if len(metrics.Metrics) == 0 {
			continue
		}

		numberOfGraphs++
		data.AddColumn(fmt.Sprintf("Value-%d", i+1))

		firstTimestamp := metrics.Metrics[0].Timestamp.Unix()
		for _, metric := range metrics.Metrics {
			key := float64(metric.Timestamp.Unix() - firstTimestamp)
			if _, ok := timeData[key]; !ok {
				timeData[key] = map[int]float64{}
			}
			timeData[key][i] = metric.Value
		}
	}
	if numberOfGraphs == 0 {
		return ""
	}

	g.convertToData(timeData, numberOfGraphs, data)

	stringBuilder := &strings.Builder{}
	tm.Output = bufio.NewWriter(stringBuilder)
	_, err := tm.Println(chart.Draw(data))
	if err != nil {
		fmt.Printf("Error while drawing: %v\n", err)
	}

	tm.Flush()

	return stringBuilder.String()
}

func (g *graph) convertToData(timeData map[float64]map[int]float64, numberOfGraphs int, data *tm.DataTable) {
	keysSorted := sortKeys(timeData)
	for _, key := range keysSorted {
		all := make([]float64, 0, numberOfGraphs+1)
		all = append(all, key)
		for graphIndex := 0; graphIndex < numberOfGraphs; graphIndex++ {
			graphValue, ok := timeData[key][graphIndex]
			if !ok {
				graphValue = findPrev(timeData, graphIndex, key)
			}
			all = append(all, graphValue)
		}
		data.AddRow(all...)
	}
}

func sortKeys(data map[float64]map[int]float64) []float64 {
	allKeys := make([]float64, 0, len(data))
	for k, _ := range data {
		allKeys = append(allKeys, k)
	}

	sort.Float64s(allKeys)

	return allKeys
}

func findPrev(timeData map[float64]map[int]float64, graphIndex int, graphValueToSearch float64) float64 {
	valueToReturn := float64(0)
	for key, value := range timeData {
		for currGraphIndex, _ := range value {
			if currGraphIndex == graphIndex && key < graphValueToSearch {
				valueToReturn = key
			}
		}
	}
	return valueToReturn
}
