package printer

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test_openMetricsPrinter_Print(t *testing.T) {
	s := strings.Builder{}
	p := &openMetricsPrinter{
		writer: &s,
	}
	p.Print("entry 1")
	p.Print("entry 2")
	p.Print("entry 3")
	assert.Equal(t, "entry 1\nentry 2\nentry 3\n", s.String())
}
