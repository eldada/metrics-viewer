package visualization

import (
	tm "github.com/buger/goterm"
	"github.com/eldada/metrics-viewer/models"
)

type Graph interface {

}

type graph struct {

}

func NewGraph() *graph {
	return &graph{}
}

func (g *graph) PrintOnce([]models.Metrics) {
	tm.Printf("")
}