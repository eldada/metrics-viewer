package visualization

import (
	"github.com/eldada/metrics-viewer/models"
	"github.com/rivo/tview"
	"strings"
	"time"
)

type Index interface {
}

type index struct {
}

func NewIndex() *index {
	return &index{}
}

func (*index) Present() {
	app := tview.NewApplication()
	main := tview.NewTextView().SetDynamicColors(true)
	menu := tview.NewList().
		AddItem("metric 1", "metric 1 description", 'a', func() {
			res := NewGraph().SprintOnce(models.Metrics{
				Metrics: []models.Metric{
					{Value: 1.2323, Labels: nil, Timestamp: time.Now()},
					{Value: 1.56443213, Labels: nil, Timestamp: time.Now().Add(1 * time.Second)},
					{Value: 1.923491, Labels: nil, Timestamp: time.Now().Add(2 * time.Second)},
					{Value: 2.31231, Labels: nil, Timestamp: time.Now().Add(3 * time.Second)},
					{Value: 1.223132, Labels: nil, Timestamp: time.Now().Add(4 * time.Second)},
					{Value: 3.21321, Labels: nil, Timestamp: time.Now().Add(5 * time.Second)},
					{Value: 1.213213, Labels: nil, Timestamp: time.Now().Add(6 * time.Second)},
				},
				Name:        "Test",
				Description: "1234",
			})

			main.SetText(replaceColors(res))
		}).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})

	grid := tview.NewGrid().
		SetRows(3, 0).
		SetColumns(30, 0).
		SetBorders(true).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("JFrog metrics"), 0, 0, 1, 3, 0, 0, false)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false).
		AddItem(main, 1, 0, 1, 1, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false).
		AddItem(main, 1, 1, 1, 2, 0, 100, false)

	if err := app.SetRoot(grid, true).SetFocus(menu).Run(); err != nil {
		panic(err)
	}
}

func replaceColors(res string) string {
	colorsReplacement := map[string]string{
		"\033[31m":  "[red]",
		"\033[32m":  "[green]",
		"\033[33m":  "[yellow]",
		"\033[34m":  "[blue]",
		"\u001B[0m": "[-]", // reset
	}
	for orig, newColor := range colorsReplacement {
		res = strings.ReplaceAll(res, orig, newColor)
	}

	return res
}
