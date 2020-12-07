package visualization

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type rowCollector struct {
	rows [][]float64
}

func (rc *rowCollector) AddRow(row ...float64) {
	rc.rows = append(rc.rows, row)
}

func Test_convertToData(t *testing.T) {
	testRowAggregator := &rowCollector{}
	type args struct {
		timeData       map[float64]map[int]float64
		numberOfGraphs int
		data           rowAggregator
	}
	tests := []struct {
		name     string
		args     args
		expected [][]float64
	}{
		{
			name: "should convert to [][]float64",
			args: args{
				timeData:       map[float64]map[int]float64{1.12: {0: 11.0}, 10.1: {0: 16.53}, 15.32: {0: 13.53}},
				numberOfGraphs: 1,
				data:           testRowAggregator,
			},
			expected: [][]float64{{1.12, 11}, {10.1, 16.53}, {15.32, 13.53}},
		},
		{
			name: "should sort keys",
			args: args{
				timeData:       map[float64]map[int]float64{10.1: {0: 16.53}, 1.12: {0: 11.0}, 15.32: {0: 13.53}},
				numberOfGraphs: 1,
				data:           testRowAggregator,
			},
			expected: [][]float64{{1.12, 11}, {10.1, 16.53}, {15.32, 13.53}},
		},
		{
			name: "should use prev if missing",
			args: args{
				timeData:       map[float64]map[int]float64{1.12: {0: 11.0}, 10.1: {3: 16.53}, 15.32: {0: 13.53}},
				numberOfGraphs: 1,
				data:           testRowAggregator,
			},
			expected: [][]float64{{1.12, 11}, {10.1, 11}, {15.32, 13.53}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRowAggregator.rows = [][]float64{}

			convertToData(tt.args.timeData, tt.args.numberOfGraphs, tt.args.data)
			assert.Equal(t, tt.expected, testRowAggregator.rows)
		})
	}
}
