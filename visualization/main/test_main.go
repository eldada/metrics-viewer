package main

import (
	"context"
	"github.com/eldada/metrics-viewer/visualization"
	"time"
)

func main() {
	visualization.NewIndex().Present(context.Background(), 1 * time.Second)
}