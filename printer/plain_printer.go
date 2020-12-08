package printer

import (
	"fmt"
	"io"
)

type openMetricsPrinter struct {
	writer io.Writer
}

func (p *openMetricsPrinter) Print(entry string) error {
	_, err := fmt.Fprintln(p.writer, entry)
	return err
}
