package visualization

import (
	"bufio"
	"fmt"
	"sort"
	"strings"

	tm "github.com/buger/goterm"
	"github.com/eldada/metrics-viewer/models"
)

type Graph interface {
	SetPointCharacter(char string)
}

type graph struct {
	pointChar string
}

func NewGraph() *graph {
	return &graph{
		pointChar: "·", // Default to middle dot (softer than bullet)
	}
}

func (g *graph) SetPointCharacter(char string) {
	g.pointChar = char
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

	convertToData(timeData, numberOfGraphs, data)

	stringBuilder := &strings.Builder{}
	tm.Output = bufio.NewWriter(stringBuilder)
	result := chart.Draw(data)

	// Replace the hard-coded bullet points with the selected character
	result = strings.ReplaceAll(result, "•", g.pointChar)

	_, err := tm.Println(result)
	if err != nil {
		fmt.Printf("Error while drawing: %v\n", err)
	}

	tm.Flush()

	return stringBuilder.String()
}

type rowAggregator interface {
	AddRow(elms ...float64)
}

func convertToData(timeData map[float64]map[int]float64, numberOfGraphs int, data rowAggregator) {
	keysSorted := sortKeys(timeData)
	for _, key := range keysSorted {
		all := make([]float64, 0, numberOfGraphs+1)
		all = append(all, key)
		for graphIndex := 0; graphIndex < numberOfGraphs; graphIndex++ {
			graphValue, ok := timeData[key][graphIndex]
			if !ok {
				graphValue = findPrevValue(timeData, graphIndex, key)
			}
			all = append(all, graphValue)
		}
		data.AddRow(all...)
	}
}

func sortKeys(data map[float64]map[int]float64) []float64 {
	allKeys := make([]float64, 0, len(data))
	for k := range data {
		allKeys = append(allKeys, k)
	}

	sort.Float64s(allKeys)

	return allKeys
}

func findPrevValue(timeData map[float64]map[int]float64, graphIndex int, graphValueToSearch float64) float64 {
	valueToReturn := float64(0)
	for key, value := range timeData {
		for currGraphIndex, currVal := range value {
			if currGraphIndex == graphIndex && key < graphValueToSearch {
				valueToReturn = currVal
			}
		}
	}
	return valueToReturn
}
