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
	i.allItemsMenu = newMenu
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
		i.setSecondHeader(fmt.Sprintf("[red]%s[-]", err.Error()), "")
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

	if i.drawing {
		i.app.Draw()
	}
}

func (i *index) setDescriptionOnItems(menu *tview.List, m models.Metrics) {
	items := menu.FindItems(m.Name, ignoreSecondaryText, false, false)
	for _, item := range items {
		menu.SetItemText(item, m.Name, m.Description)
	}
}

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
		}
		return event
	})
	i.addFilterMenuItem(menu)
	i.addQuitMenuItem(menu)

	menu.SetCurrentItem(quitItemIndex)
	return menu
}

func (i *index) selectedFunc(name string) {
	name = clearColor(name, highlightColor)
	i.toggleSelected(name)

	_, _, width, height := i.mainContent.GetInnerRect()
	summary, descriptionSummary, selectedMetrics := i.selectedToList()
	res := NewGraph().SprintOnce(width, height, selectedMetrics...)
	i.mainContent.SetText(replaceColors(res))
	i.setSecondHeader(summary, descriptionSummary)
	i.hasError = false
}

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
	items := menu.FindItems(name, ignoreSecondaryText, false, false)
	for _, itemIndex := range items {
		main, sec := menu.GetItemText(itemIndex)
		menu.SetItemText(itemIndex, converter(main), sec)
	}
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

func (i *index) selectedToList() (string, string, []models.Metrics) {
	selectedList := make([]models.Metrics, 0, len(i.selected))
	selectedSummary := make([]string, 0, len(i.selected))
	selectedDescSummary := make([]string, 0, len(i.selected))

	for selectedIndex, val := range i.selected {
		item := i.items[val]
		selectedList = append(selectedList, item)
		selectedSummary = append(selectedSummary, fmt.Sprintf("%s%s[-]", colors[selectedIndex], val))
		desc := item.Description
		if desc == "" {
			desc = "No description"
		}
		selectedDescSummary = append(selectedDescSummary, fmt.Sprintf("%s%s[-]", colors[selectedIndex], desc))
	}

	return strings.Join(selectedSummary, "[white] | [-]"), strings.Join(selectedDescSummary, "[white] | [-]"), selectedList
}

func (i *index) addFilterMenuItem(menu *tview.List) {
	menu.AddItem("", "", 0, nil)
}

func (i *index) addQuitMenuItem(menu *tview.List) {
	menu.AddItem("Quit", "Choose to exit", 0, func() { i.app.Stop() })
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

func (i *index) setSecondHeader(secondHeader string, thirdHeader string) *tview.TextView {
	return i.header.SetText(fmt.Sprintf("%s\n%s\n%s", defaultHeader, secondHeader, thirdHeader))
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
