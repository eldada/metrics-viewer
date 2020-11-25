package models

import "time"

type Metric struct {
	Value float64
	Labels map[string]interface{}
	Timestamp time.Time
}

type Metrics struct {
	Metrics []Metric
	Name string
	Description string
}