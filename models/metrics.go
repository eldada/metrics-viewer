package models

import "time"

type Metric struct {
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type Metrics struct {
	Metrics     []Metric
	Key         string
	Name        string
	Description string
}
