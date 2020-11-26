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
	drawing     bool
	provider    provider.Provider
	header      *tview.TextView
}

func NewIndex() *index {
	rand.Seed(time.Now().Unix())
	return &index{}
}

const defaultHeader = "JFrog metrics"

func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.provider = prov
	i.app = tview.NewApplication()
	i.mainContent = tview.NewTextView().SetDynamicColors(true)

	i.header = tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true).SetText(defaultHeader)
	i.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(30, 0).
		SetBorders(true).
		AddItem(i.header, 0, 0, 1, 3, 0, 0, false)

	i.generateAndReplaceMenuOnGrid()

	i.grid.AddItem(i.mainContent, 1, 1, 1, 2, 0, 100, false)

	i.app = i.app.SetRoot(i.grid, true).SetFocus(i.currentMenu)
	go i.updateMenuOnGrid(ctx, interval)
	i.drawing = true
	if err := i.app.Run(); err != nil {
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
	if i.currentMenu != nil {
		i.grid.RemoveItem(i.currentMenu)
	}

	i.currentMenu = newMenu
	i.grid.AddItem(i.currentMenu, 1, 0, 1, 1, 0, 100, false)
	if i.drawing {
		i.app.Draw()
		i.app.SetFocus(i.currentMenu)
	}
}

func (i *index) generateMenu() *tview.List {
	menu := tview.NewList()
	metrics, err := i.provider.Get()
	if err != nil {
		i.header.SetText(fmt.Sprintf("%s\n[red]%s[-]", defaultHeader, err.Error()))
		return menu
	} else {
		i.header.SetText(defaultHeader)
	}

	for index, m := range metrics {
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
