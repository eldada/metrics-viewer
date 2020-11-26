package visualization

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/rivo/tview"
	"math/rand"
	"strings"
	"time"
)

type Index interface {
}

type index struct {
	currentMenu *tview.List
	grid        *tview.Grid
	app         *tview.Application
	mainContent *tview.TextView
}

func NewIndex() *index {
	rand.Seed(time.Now().Unix())
	return &index{}
}

func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.app = tview.NewApplication()
	i.mainContent = tview.NewTextView().SetDynamicColors(true)

	i.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(30, 0).
		SetBorders(true).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("JFrog metrics"), 0, 0, 1, 3, 0, 0, false)

	i.generateAndReplaceMenuOnGrid()
	go i.updateMenuOnGrid(ctx, interval)

	i.grid.AddItem(i.mainContent, 1, 1, 1, 2, 0, 100, false)

	if err := i.app.SetRoot(i.grid, true).SetFocus(i.currentMenu).Run(); err != nil {
		panic(err)
	}
}

func (i *index) updateMenuOnGrid(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.generateAndReplaceMenuOnGrid()
		}
	}
}

func (i *index) generateAndReplaceMenuOnGrid() {
	newMenu := i.generateMenu()
	if i.currentMenu == nil {
		i.grid.RemoveItem(i.currentMenu)
	}

	i.currentMenu = newMenu
	i.grid.AddItem(i.currentMenu, 1, 0, 1, 1, 0, 100, false)
}

func (i *index) generateMenu() *tview.List {
	menu := tview.NewList()
	for index, m := range i.getMetrics() {
		i.addItemToMenu(menu, index, m)
	}

	menu.AddItem("Quit", "Press to exit", 'q', func() {
		i.app.Stop()
	})
	return menu
}

func (i *index) addItemToMenu(menu *tview.List, index int, m models.Metrics) *tview.List {
	return menu.AddItem(m.Name, m.Description, rune(index+97), func() {
		_, _, width, height := i.mainContent.GetInnerRect()
		res := NewGraph().SprintOnce(width, height, m)
		i.mainContent.SetText(replaceColors(res))
	})
}

func (*index) getMetrics() []models.Metrics {
	n := rand.Intn(10)

	metrics := make([]models.Metrics, 0, n)
	for i := 0; i < n; i++ {
		metrics = append(metrics, models.Metrics{
			Metrics: []models.Metric{
				{Value: 1.2323 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now()},
				{Value: 1.56443213 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(1 * time.Second)},
				{Value: 1.923491 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(2 * time.Second)},
				{Value: 2.31231 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(3 * time.Second)},
				{Value: 1.223132 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(4 * time.Second)},
				{Value: 3.21321 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(5 * time.Second)},
				{Value: 1.213213 * float64(rand.Intn(10)), Labels: nil, Timestamp: time.Now().Add(6 * time.Second)},
			},
			Name:        fmt.Sprintf("Metric %d", i),
			Description: fmt.Sprintf("Metric %d description", i),
		})
	}

	return metrics
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
