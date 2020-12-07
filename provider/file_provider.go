package provider

import (
	"bytes"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/parser"
	"github.com/hpcloud/tail"
	"strings"
	"time"
)

func newFileProvider(c Config) (*fileProvider, error) {
	t, err := tail.TailFile(c.File(), tail.Config{
		Follow: true,
		ReOpen: true,
	})
	if err != nil {
		return nil, err
	}
	return &fileProvider{
		conf: c,
		tail: t,
	}, nil
}

type fileProvider struct {
	conf          Config
	tail          *tail.Tail
	stagedMetrics []models.Metrics
}

const maxBatchSize = 10240         // no real reason ...
const maxBatchIntervalFactor = 0.9 // use up to 90% of an interval time to fetch and process records

func (p *fileProvider) Get() ([]models.Metrics, error) {
	r := bytes.NewReader([]byte{})
	b := bytes.NewBuffer([]byte{})
	start := now()
	maxBatchIntervalDuration := time.Duration(float64(p.conf.Interval()) * maxBatchIntervalFactor)
	metricsCollection := p.stagedMetrics
	noLinesCounter := 0
	for {
		select {
		case line, ok := <-p.tail.Lines:
			if !ok {
				// channel is closed
				break
			}
			if line == nil {
				continue
			}
			noLinesCounter = 0
			b.WriteString(line.Text)
			b.WriteRune('\n')
			if strings.HasPrefix(line.Text, "#") || line.Text == "" {
				continue
			}
			r.Reset(b.Bytes())
			metrics, err := parser.ParseMetrics(r)
			if err != nil {
				return nil, err
			}
			b.Reset()
			metricsCollection = append(metricsCollection, metrics...)
			p.stagedMetrics = metricsCollection
		default:
			// backoff: 0, ..., 1ms, ..., 10ms ...
			noLinesCounter++
			sleepTime := time.Duration(noLinesCounter/10) * time.Millisecond
			if sleepTime > 0 {
				if sleepTime > 10*time.Millisecond {
					sleepTime = 10 * time.Millisecond
				}
				time.Sleep(sleepTime)
			}
		}
		if len(metricsCollection) >= maxBatchSize || now().Sub(start) >= maxBatchIntervalDuration {
			break
		}
	}
	p.stagedMetrics = nil
	return metricsCollection, nil
}

func (p *fileProvider) Close() error {
	return p.tail.Stop()
}
