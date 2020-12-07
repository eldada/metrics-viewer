package visualization

import (
	"context"
	"fmt"
	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type Index interface {
}

type index struct {
	currentMenu          *tview.List
	allItemsMenu         *tview.List
	grid                 *tview.Grid
	app                  *tview.Application
	mainContent          *tview.TextView
	provider             provider.Provider
	missingMetricsCache  missingMetricsCache
	header               *tview.TextView
	metricName           *tview.TextView
	selected             []string
	items                map[string]models.Metrics
	hasError             bool
	drawing              bool
	userInteractionMutex sync.Locker
	rightPane            *tview.TextView
}

func NewIndex() *index {
	rand.Seed(time.Now().Unix())
	return &index{
		missingMetricsCache:  newMissingMetricsCache(),
		items:                map[string]models.Metrics{},
		userInteractionMutex: &sync.Mutex{},
	}
}

const maximumSelectedItems = 5
const defaultHeader = "JFrog metrics"
const ignoreSecondaryText = "---N/A---"
const highlightColor = "[lightgray::b](x) "
const filterColor = "[darkgray:gray:b]Filter: "

var colors = []string{"[green]", "[yellow]", "[blue]", "[teal]", "[gray]", "[gold]", "[indigo]", "[lavender]"}

const filterItemIndex = 0
const quitItemIndex = 1

// Main function to create the application
func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.provider = prov
	i.app = tview.NewApplication()
	i.mainContent = tview.NewTextView().SetDynamicColors(true)
	i.rightPane = tview.NewTextView().SetDynamicColors(true)
	i.header = tview.NewTextView().SetTextAlign(tview.AlignCenter).SetDynamicColors(true).SetText(defaultHeader)
	i.grid = tview.NewGrid().
		SetRows(3, 0).
		SetColumns(30, 0, 50).
		SetBorders(true).
		SetBordersColor(tcell.ColorDarkSeaGreen).
		AddItem(i.header, 0, 0, 1, 3, 0, 0, false)

	i.grid.SetBackgroundColor(tcell.ColorBlack)

	newMenu := i.generateMenu()
	i.allItemsMenu = newMenu
	i.currentMenu = newMenu

	i.grid.AddItem(i.currentMenu, 1, 0, 1, 1, 0, 100, false)
	i.grid.AddItem(i.mainContent, 1, 1, 1, 1, 0, 100, false)
	i.grid.AddItem(i.rightPane, 1, 2, 1, 1, 0, 100, false)

	i.app = i.app.SetRoot(i.grid, true).SetFocus(i.currentMenu)
	go i.updateMenuOnGrid(ctx, interval)
	i.replaceMenuContentOnGrid()
	i.app.SetAfterDrawFunc(func(screen tcell.Screen) {
		i.drawing = true
	})
	if err := i.app.Run(); err != nil {
		panic(err)
	}
}

// A ticker to update the metrics from the source
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

// Recreating the menu every time an update has arrived
func (i *index) replaceMenuContentOnGrid() {
	metrics, err := i.provider.Get()

	if err != nil {
		i.setSecondHeader(fmt.Sprintf("[red]%s[-]", err.Error()))
		i.hasError = true
		return
	} else {
		i.setSecondHeader("")
	}

	if i.hasError {
		i.hasError = false
		i.header.SetText(defaultHeader)
	}

	metrics = i.missingMetricsCache.AddToMetrics(metrics)

	i.upsertMetricsOnMenu(metrics)

	if i.drawing {
		i.app.Draw()
	}
}

// Adding new metrics or updating the current ones
func (i *index) upsertMetricsOnMenu(metrics []models.Metrics) {
	for _, m := range metrics {
		_, ok := i.items[m.Name]
		if ok {
			i.setDescriptionOnItems(i.currentMenu, m)
			if i.allItemsMenu != i.currentMenu {
				i.setDescriptionOnItems(i.allItemsMenu, m)
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
}

func (i *index) setDescriptionOnItems(menu *tview.List, m models.Metrics) {
	items := i.findExactMatch(menu, m.Name)
	for _, item := range items {
		menu.SetItemText(item, m.Name, m.Description)
	}
}

// Generating a new menu to replace, useful for the searchbar capability
func (i *index) generateMenu() *tview.List {
	menu := tview.NewList()

	menu.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) { i.selectedFunc(name) })
	menu.SetWrapAround(false)
	menu.SetBorderPadding(0, 0, 1, 0)
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			i.searchbarInput(event.Rune())
		} else if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			i.searchbarDelete()
		} else if event.Key() == tcell.KeyEscape {
			i.searchbarClear()
		}
		return event
	})
	i.addFilterMenuItem(menu)
	i.addQuitMenuItem(menu)

	menu.SetCurrentItem(quitItemIndex)
	return menu
}

// Reacting to the user selection
func (i *index) selectedFunc(name string) {
	name = clearColor(name, highlightColor)
	i.toggleSelected(name)

	_, _, width, height := i.mainContent.GetInnerRect()
	summary, selectedMetrics := i.selectedToList()
	res := NewGraph().SprintOnce(width, height, selectedMetrics...)
	i.mainContent.SetText(replaceColors(res))
	i.setRightPane(summary)
	i.hasError = false
}

// Inverting a menu item selection
func (i *index) toggleSelected(name string) {
	i.userInteractionMutex.Lock()
	defer i.userInteractionMutex.Unlock()
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
	converter := func(main string) string { return addColor(main, highlightColor) }
	i.findAndUpdateItemText(i.currentMenu, name, converter)
	if i.allItemsMenu != i.currentMenu {
		i.findAndUpdateItemText(i.allItemsMenu, name, converter)
	}
}

func (i *index) removeSelectedItemColor(name string) {
	converter := func(main string) string { return clearColor(main, highlightColor) }
	modifiedName := addColor(clearColor(name, highlightColor), highlightColor)
	i.findAndUpdateItemText(i.currentMenu, modifiedName, converter)
	if i.allItemsMenu != i.currentMenu {
		i.findAndUpdateItemText(i.allItemsMenu, modifiedName, converter)
	}
}

func (i *index) findAndUpdateItemText(menu *tview.List, name string, converter func(string) string) {
	items := i.findExactMatch(menu, name)
	for _, itemIndex := range items {
		main, sec := menu.GetItemText(itemIndex)
		menu.SetItemText(itemIndex, converter(main), sec)
	}
}

func (i *index) findExactMatch(menu *tview.List, name string) []int {
	candidateItems := menu.FindItems(name, ignoreSecondaryText, false, false)
	items := make([]int, 0, len(candidateItems))
	for _, item := range candidateItems {
		main, _ := i.currentMenu.GetItemText(item)

		// Protecting vs prefix matching
		if main == name {
			items = append(items, item)
		}
	}

	return items
}

func addColor(main string, color string) string {
	return fmt.Sprintf("%s%s[-]", color, main)
}

func clearColor(name string, color string) string {
	return strings.TrimSuffix(strings.TrimPrefix(name, color), "[-]")
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

// Converting the selected items into a list of metrics
func (i *index) selectedToList() (string, []models.Metrics) {
	selectedList := make([]models.Metrics, 0, len(i.selected))
	selectedSummary := make([]string, 0, len(i.selected))

	for selectedIndex, val := range i.selected {
		item := i.items[val]
		selectedList = append(selectedList, item)
		summaryToAdd := fmt.Sprintf("%s%s[-]\n", colors[selectedIndex], val)
		desc := item.Description
		if desc == "" {
			desc = "No description"
		}
		summaryToAdd += fmt.Sprintf("%s%s[-]\n", colors[selectedIndex], desc)
		summaryToAdd += fmt.Sprintf("%sMax: %v[-]\n", colors[selectedIndex], findMaxMetricValue(item.Metrics))
		summaryToAdd += fmt.Sprintf("%sCurrent: %v[-]\n", colors[selectedIndex], findCurrentMetricValue(item.Metrics))
		selectedSummary = append(selectedSummary, fmt.Sprintf("%s%s[-]", colors[selectedIndex], summaryToAdd))
	}

	return strings.Join(selectedSummary, "[white]\n[-]"), selectedList
}

func findMaxMetricValue(metrics []models.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	maxValue := metrics[0].Value
	for _, m := range metrics {
		if m.Value > maxValue {
			maxValue = m.Value
		}
	}
	return maxValue
}

func findCurrentMetricValue(metrics []models.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	return metrics[len(metrics)-1].Value
}

func (i *index) addFilterMenuItem(menu *tview.List) {
	menu.AddItem("", "", 0, nil)
}

func (i *index) addQuitMenuItem(menu *tview.List) {
	menu.AddItem("Quit", "Choose to exit", 0, func() {
		if closer, ok := i.provider.(io.Closer); ok {
			_ = closer.Close()
		}
		i.app.Stop()
	})
}

func (i *index) addItemToMenu(m models.Metrics) {
	i.allItemsMenu.AddItem(m.Name, m.Description, 0, nil)
	if i.currentMenu != i.allItemsMenu {
		filterText, _ := i.allItemsMenu.GetItemText(filterItemIndex)
		clearedText := clearColor(filterText, filterColor)
		if textContains(m.Name, clearedText) || textContains(m.Description, clearedText) {
			i.currentMenu.AddItem(m.Name, m.Description, 0, nil)
		}
	}
}

func textContains(text string, filterText string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(filterText))
}

func (i *index) setSecondHeader(secondHeader string) *tview.TextView {
	return i.header.SetText(fmt.Sprintf("%s\n%s", defaultHeader, secondHeader))
}

func (i *index) setRightPane(secondHeader string) *tview.TextView {
	return i.rightPane.SetText(secondHeader)
}

func (i *index) searchbarInput(r rune) {
	if r >= '!' && r <= '~' {
		text, _ := i.allItemsMenu.GetItemText(filterItemIndex)
		i.setFilterText(clearColor(text, filterColor) + string(r))
	}
}

func (i *index) searchbarDelete() {
	text, _ := i.allItemsMenu.GetItemText(filterItemIndex)
	clearedText := clearColor(text, filterColor)
	if len(clearedText) == 0 {
		return
	}
	i.setFilterText(clearedText[:len(clearedText)-1])
}

func (i *index) searchbarClear() {
	i.setFilterText("")
}

func (i *index) setFilterText(newFilterText string) {
	newFilterColored := newFilterText
	if len(newFilterText) > 0 {
		newFilterColored = addColor(newFilterText, filterColor)
	}
	i.allItemsMenu.SetItemText(filterItemIndex, newFilterColored, "")
	if i.allItemsMenu != i.currentMenu {
		i.currentMenu.SetItemText(filterItemIndex, newFilterColored, "")
	}
	i.refreshMenuAccordingToFilterInput(newFilterText)
}

// This function should be used as part of the searchbar functionality to show only relevant menu items
func (i *index) refreshMenuAccordingToFilterInput(input string) {
	i.userInteractionMutex.Lock()
	defer i.userInteractionMutex.Unlock()
	newMenu := i.allItemsMenu
	alreadySetCurrentItem := false
	if input != "" {
		itemsIndexes := i.allItemsMenu.FindItems(input, input, false, true)
		newMenu = i.generateMenu()
		selectedMainText, _ := i.currentMenu.GetItemText(filterItemIndex)
		clearedFilterText := clearColor(selectedMainText, filterColor)
		for _, itemIndex := range itemsIndexes {
			if itemIndex == quitItemIndex {
				continue
			}
			text, secondary := i.allItemsMenu.GetItemText(itemIndex)
			if itemIndex == filterItemIndex {
				newMenu.SetItemText(filterItemIndex, text, secondary)
				continue
			}
			newMenu.AddItem(text, secondary, 0, nil)
			if !alreadySetCurrentItem && textContains(text, clearedFilterText) {
				alreadySetCurrentItem = true
				newMenu.SetCurrentItem(newMenu.GetItemCount() - 1)
			}
		}
	}

	i.grid.RemoveItem(i.currentMenu)
	i.grid.AddItem(newMenu, 1, 0, 1, 1, 0, 100, false)
	i.currentMenu = newMenu
	i.app.SetFocus(i.currentMenu)
}

func replaceColors(res string) string {
	colorsReplacement := map[string]string{
		"\033[31m":  colors[0],
		"\033[32m":  colors[1],
		"\033[33m":  colors[2],
		"\033[34m":  colors[3],
		"\033[35m":  colors[4],
		"\033[36m":  colors[5],
		"\033[37m":  colors[6],
		"\033[38m":  colors[7],
		"\u001B[0m": "[-]", // reset
	}
	for orig, newColor := range colorsReplacement {
		res = strings.ReplaceAll(res, orig, newColor)
	}

	return res
}
