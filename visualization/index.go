package visualization

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Index interface {
}

type index struct {
	currentMenu         *tview.List
	grid                *tview.Grid
	app                 *tview.Application
	mainContent         *tview.TextView
	provider            provider.Provider
	missingMetricsCache missingMetricsCache
	header              *tview.TextView
	metricName          *tview.TextView
	selected            []string
	items               map[string]models.Metrics
	hasError            bool
	drawing             bool
	selectedMutex       sync.Locker
}

func NewIndex() *index {
	rand.Seed(time.Now().Unix())
	return &index{
		missingMetricsCache: newMissingMetricsCache(),
		items:               map[string]models.Metrics{},
		selectedMutex:       &sync.Mutex{},
	}
}

const maximumSelectedItems = 10
const defaultHeader = "JFrog metrics"
const ignoreSecondaryText = "---N/A---"
const highlightColor = "[darkgray::b](x) "

func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.provider = prov
	i.app = tview.NewApplication()
	i.mainContent = tview.NewTextView().SetDynamicColors(true)

	i.header = tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true).SetText(defaultHeader)
	i.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(30, 0).
		SetBorders(true).
		SetBordersColor(tcell.ColorDarkSeaGreen).
		AddItem(i.header, 0, 0, 1, 3, 0, 0, false)

	i.grid.SetBackgroundColor(tcell.ColorBlack)

	newMenu := i.generateMenu()
	i.currentMenu = newMenu

	i.grid.AddItem(i.mainContent, 1, 1, 1, 2, 0, 100, false)
	i.grid.AddItem(i.currentMenu, 1, 0, 1, 1, 0, 100, false)

	i.app = i.app.SetRoot(i.grid, true).SetFocus(i.currentMenu)
	go i.updateMenuOnGrid(ctx, interval)
	i.replaceMenuContentOnGrid()
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
			i.replaceMenuContentOnGrid()
		}
	}
}

func (i *index) replaceMenuContentOnGrid() {
	metrics, err := i.provider.Get()

	if err != nil {
		i.setSecondHeader(fmt.Sprintf("[red]%s[-]", err.Error()))
		i.hasError = true
		return
	}

	if i.hasError {
		i.hasError = false
		i.header.SetText(defaultHeader)
	}

	metrics = i.missingMetricsCache.AddToMetrics(metrics)

	for _, m := range metrics {
		_, ok := i.items[m.Name]
		if ok {
			items := i.currentMenu.FindItems(m.Name, ignoreSecondaryText, false, false)
			for _, item := range items {
				i.currentMenu.SetItemText(item, m.Name, m.Description)
			}

			if selectedIndex := i.findSelectedIndex(m.Name); selectedIndex >= 0 {
				i.removeSelected(selectedIndex)
				i.selectedFunc(m.Name) // reselect to redraw
			}
		} else {
			i.addItemToMenu(m)
		}

		i.items[m.Name] = m
	}

	if i.drawing {
		i.app.Draw()
	}
}

func (i *index) generateMenu() *tview.List {
	menu := tview.NewList()

	menu.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) { i.selectedFunc(name) })

	i.addQuitMenuItem(menu)
	return menu
}

func (i *index) selectedFunc(name string) {
	name = clearColor(name)
	i.toggleSelected(name)

	_, _, width, height := i.mainContent.GetInnerRect()
	summary, selectedMetrics := i.selectedToList()
	res := NewGraph().SprintOnce(width, height, selectedMetrics...)
	i.mainContent.SetText(replaceColors(res))
	i.setSecondHeader(fmt.Sprintf("[teal]%s[-]", summary))
	i.hasError = false
}

func (i *index) toggleSelected(name string) {
	i.selectedMutex.Lock()
	defer i.selectedMutex.Unlock()
	selectedIndex := i.findSelectedIndex(name)
	if selectedIndex >= 0 {
		i.removeSelected(selectedIndex)
	} else {
		if len(i.selected) >= maximumSelectedItems {
			i.removeSelected(0)
		}
		i.selected = append(i.selected, name)
		i.setSelectedItemColor(name)
	}
}

func (i *index) setSelectedItemColor(name string) {
	items := i.currentMenu.FindItems(name, ignoreSecondaryText, false, false)
	for _, itemIndex := range items {
		main, sec := i.currentMenu.GetItemText(itemIndex)
		i.currentMenu.SetItemText(itemIndex, addSelectedColor(main), sec)
	}
}

func (i *index) removeSelectedItemColor(name string) {
	items := i.currentMenu.FindItems(addSelectedColor(clearColor(name)), ignoreSecondaryText, false, false)
	for _, itemIndex := range items {
		main, sec := i.currentMenu.GetItemText(itemIndex)
		i.currentMenu.SetItemText(itemIndex, fmt.Sprintf("%s", clearColor(main)), sec)
	}
}

func addSelectedColor(main string) string {
	return fmt.Sprintf("%s%s[-]", highlightColor, main)
}

func clearColor(name string) string {
	return strings.TrimSuffix(strings.TrimPrefix(name, highlightColor), "[-]")
}

func (i *index) findSelectedIndex(name string) int {
	for selectedIndex, candidate := range i.selected {
		if name == candidate {
			return selectedIndex
		}
	}

	return -1
}

func (i *index) removeSelected(selectedIndex int) {
	i.removeSelectedItemColor(i.selected[selectedIndex])

	if selectedIndex >= 0 {
		copy(i.selected[selectedIndex:], i.selected[selectedIndex+1:])
		i.selected[len(i.selected)-1] = ""
		i.selected = i.selected[:len(i.selected)-1]
	}
}

func (i *index) selectedToList() (string, []models.Metrics) {
	selectedList := make([]models.Metrics, 0, len(i.selected))
	selectedSummary := make([]string, 0, len(i.selected))
	for _, val := range i.selected {
		selectedList = append(selectedList, i.items[val])
		selectedSummary = append(selectedSummary, val)
	}

	return strings.Join(selectedSummary, ", "), selectedList
}

func (i *index) addQuitMenuItem(menu *tview.List) {
	menu.AddItem("Quit", "Press to exit", 'q', func() { i.app.Stop() })
}

func (i *index) addItemToMenu(m models.Metrics) *tview.List {
	return i.currentMenu.AddItem(m.Name, m.Description, 0, nil)
}

func (i *index) setSecondHeader(secondHeader string) *tview.TextView {
	return i.header.SetText(fmt.Sprintf("%s\n\n%s", defaultHeader, secondHeader))
}

func (i *index) findShortcut(index int) rune {
	shortcut := rune(index + 97)
	if shortcut >= 'q' {
		shortcut++
	}
	if shortcut > 'z' {
		shortcut = 0
	}
	return shortcut
}

func replaceColors(res string) string {
	colorsReplacement := map[string]string{
		"\033[31m":  "[red]",
		"\033[32m":  "[green]",
		"\033[33m":  "[yellow]",
		"\033[34m":  "[blue]",
		"\033[35m":  "[teal]",
		"\033[36m":  "[gray]",
		"\033[37m":  "[gold]",
		"\033[38m":  "[indigo]",
		"\033[39m":  "[lavender]",
		"\033[310m": "[olive]",
		"\033[311m": "[ivory]",
		"\u001B[0m": "[-]", // reset
	}
	for orig, newColor := range colorsReplacement {
		res = strings.ReplaceAll(res, orig, newColor)
	}

	return res
}
