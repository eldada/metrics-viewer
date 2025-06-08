package visualization

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var version = "unknown"

// SetVersion sets the version for display in the UI
func SetVersion(v string) {
	version = v
}

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
const defaultHeader = "Metrics Viewer"
const usageInstructions = "Use '/' to search metrics (supports regex) • Use ↑↓ to navigate • ENTER to select • ESC to clear • Quit or CTRL+C to exit"
const ignoreSecondaryText = "---N/A---"
const highlightColor = "[lightgray::b](x) "
const filterColor = "[white:blue:b]"
const cursorChar = "█"
const selectedItemColor = "[white:blue:b]"
const separatorLine = "───────────────────────"
const separatorItemIndex = 1
const filterSeparatorIndex = 3

var colors = []string{"[green]", "[yellow]", "[blue]", "[teal]", "[gray]", "[gold]", "[indigo]", "[lavender]"}

const filterItemIndex = 2
const quitItemIndex = 0

// Main function to create the application
func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.provider = prov
	i.app = tview.NewApplication()
	i.mainContent = tview.NewTextView().SetDynamicColors(true)
	i.rightPane = tview.NewTextView().SetDynamicColors(true)
	i.header = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetText("[yellow::b]" + defaultHeader + " (" + version + "[-:-:-])\n[::d]" + usageInstructions + "[-:-:-]")

	newMenu := i.generateMenu()
	i.allItemsMenu = newMenu
	i.currentMenu = newMenu

	i.grid = tview.NewGrid().
		SetRows(2, 0).
		SetColumns(-3, -10, -3).
		SetMinSize(0, 30).
		SetBorders(true).
		SetBordersColor(tcell.ColorGreen).
		AddItem(i.header, 0, 0, 1, 3, 0, 0, false)

	i.grid.SetBackgroundColor(tcell.ColorBlack)

	i.grid.AddItem(i.currentMenu, 1, 0, 1, 1, 0, 100, false)
	i.grid.AddItem(i.mainContent, 1, 1, 1, 1, 0, 100, false)
	i.grid.AddItem(i.rightPane, 1, 2, 1, 1, 0, 100, false)

	i.searchbarClear()

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

	// Store current selection before update
	currentItem := i.currentMenu.GetCurrentItem()

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
		// Restore selection after update
		i.currentMenu.SetCurrentItem(currentItem)
		i.app.Draw()
	}
}

// Adding new metrics or updating the current ones
func (i *index) upsertMetricsOnMenu(metrics []models.Metrics) {
	for _, m := range metrics {
		_, ok := i.items[m.Name]
		if ok {
			i.items[m.Name] = m

			if selectedIndex := i.findSelectedIndex(m.Name); selectedIndex >= 0 {
				i.removeSelected(selectedIndex)
				i.selectedFunc(m.Name) // reselect to redraw
			}
		} else {
			i.addItemToMenu(m)
		}
	}

	// Regenerate the menu to show updated items
	newMenu := i.generateMenu()
	i.grid.RemoveItem(i.currentMenu)
	i.grid.AddItem(newMenu, 1, 0, 1, 1, 0, 100, false)
	i.currentMenu = newMenu
	i.app.SetFocus(i.currentMenu)
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

	menu.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) {
		if name == separatorLine {
			return
		}
		i.selectedFunc(name)
	})
	menu.SetWrapAround(false)
	menu.SetBorderPadding(0, 0, 0, 0)
	menu.ShowSecondaryText(false)
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyUp {
			currentItem := menu.GetCurrentItem()
			if currentItem == separatorItemIndex || currentItem == filterSeparatorIndex {
				if currentItem == separatorItemIndex {
					menu.SetCurrentItem(quitItemIndex)
				} else {
					menu.SetCurrentItem(filterItemIndex)
				}
				return nil
			}
		} else if event.Key() == tcell.KeyDown {
			currentItem := menu.GetCurrentItem()
			if currentItem == separatorItemIndex || currentItem == filterSeparatorIndex {
				if currentItem == separatorItemIndex {
					menu.SetCurrentItem(filterItemIndex)
				} else {
					menu.SetCurrentItem(4)
				}
				return nil
			}
		}

		if event.Key() == tcell.KeyRune {
			if event.Rune() == '/' {
				i.searchbarClear()
				i.currentMenu.SetCurrentItem(filterItemIndex)
				i.app.SetFocus(i.currentMenu)
				return nil
			}
			i.searchbarInput(event.Rune())
		} else if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			i.searchbarDelete()
		} else if event.Key() == tcell.KeyEscape {
			i.searchbarClear()
		} else if event.Key() == tcell.KeyEnter {
			if menu.GetCurrentItem() == filterItemIndex {
				if menu.GetItemCount() > 4 {
					menu.SetCurrentItem(4)
				} else {
					menu.SetCurrentItem(quitItemIndex)
				}
			}
		}
		return event
	})

	// Add Quit at the top
	menu.AddItem("Quit", "Choose to exit", 0, func() {
		if closer, ok := i.provider.(io.Closer); ok {
			_ = closer.Close()
		}
		i.app.Stop()
	})

	// Add separator after Quit
	menu.AddItem(separatorLine, "", 0, func() {
		menu.SetCurrentItem(quitItemIndex)
	})

	// Add Filter
	menu.AddItem(addColor("Filter: "+cursorChar, filterColor), "", 0, nil)

	// Add separator after Filter
	menu.AddItem(separatorLine, "", 0, func() {
		menu.SetCurrentItem(filterItemIndex)
	})

	// Sort selected items
	sortedSelected := make([]string, len(i.selected))
	copy(sortedSelected, i.selected)
	sort.Strings(sortedSelected)

	// Add selected items at the top
	for _, selectedName := range sortedSelected {
		menu.AddItem(addColor("(*) "+selectedName, selectedItemColor), "", 0, nil)
	}

	// Get all non-selected items and sort them
	var nonSelectedItems []string
	for name := range i.items {
		isSelected := false
		for _, selectedName := range i.selected {
			if name == selectedName {
				isSelected = true
				break
			}
		}
		if !isSelected {
			nonSelectedItems = append(nonSelectedItems, name)
		}
	}
	sort.Strings(nonSelectedItems)

	// Add all non-selected items
	for _, name := range nonSelectedItems {
		menu.AddItem(name, "", 0, nil)
	}

	// Ensure the menu starts with no offset to show the top items
	menu.SetOffset(0, 0)
	menu.SetCurrentItem(filterItemIndex)
	return menu
}

// Helper function to clean item names from all formatting and prefixes
func (i *index) cleanItemName(name string) string {
	// Remove all instances of "(*) " prefix
	for strings.HasPrefix(name, "(*) ") {
		name = strings.TrimPrefix(name, "(*) ")
	}
	// Remove any color formatting
	name = clearColor(name, selectedItemColor)
	name = clearColor(name, highlightColor)
	return name
}

// Reacting to the user selection
func (i *index) selectedFunc(name string) {
	// Clean the name from any existing formatting and prefixes
	name = i.cleanItemName(name)

	// Store the current index and get the next item's name before toggling selection
	currentIndex := i.currentMenu.GetCurrentItem()
	var nextItemName string
	if currentIndex+1 < i.currentMenu.GetItemCount() {
		nextName, _ := i.currentMenu.GetItemText(currentIndex + 1)
		nextItemName = i.cleanItemName(nextName)
	}

	i.toggleSelected(name)

	_, _, width, height := i.mainContent.GetInnerRect()
	summary, selectedMetrics := i.selectedToList()
	res := NewGraph().SprintOnce(width, height, selectedMetrics...)
	i.mainContent.SetText(replaceColors(res))
	i.setRightPane(summary)
	i.hasError = false

	// Regenerate the menu to reorder items
	newMenu := i.generateMenu()
	i.grid.RemoveItem(i.currentMenu)
	i.grid.AddItem(newMenu, 1, 0, 1, 1, 0, 100, false)
	i.currentMenu = newMenu

	// Find and focus on the next item in the new menu
	if nextItemName != "" {
		for idx := 2; idx < newMenu.GetItemCount(); idx++ {
			itemText, _ := newMenu.GetItemText(idx)
			itemName := i.cleanItemName(itemText)
			if itemName == nextItemName {
				newMenu.SetCurrentItem(idx)
				break
			}
		}
	}

	i.app.SetFocus(i.currentMenu)
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
	converter := func(main string) string {
		return addColor("(*) "+i.cleanItemName(main), selectedItemColor)
	}
	i.findAndUpdateItemText(i.currentMenu, name, converter)
	if i.allItemsMenu != i.currentMenu {
		i.findAndUpdateItemText(i.allItemsMenu, name, converter)
	}
}

func (i *index) removeSelectedItemColor(name string) {
	converter := func(main string) string {
		return i.cleanItemName(main)
	}
	modifiedName := i.cleanItemName(name)
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
		main, _ := menu.GetItemText(item)

		// Protecting vs prefix matching
		if i.cleanItemName(main) == i.cleanItemName(name) {
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
	cleanedName := i.cleanItemName(name)
	for selectedIndex, candidate := range i.selected {
		if cleanedName == candidate {
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
		summaryToAdd += fmt.Sprintf("%sMax:     %f[-]\n", colors[selectedIndex], findMaxMetricValue(item.Metrics))
		summaryToAdd += fmt.Sprintf("%sMin:     %f[-]\n", colors[selectedIndex], findMinMetricValue(item.Metrics))
		summaryToAdd += fmt.Sprintf("%sCurrent: %f[-]\n", colors[selectedIndex], findCurrentMetricValue(item.Metrics))
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

func findMinMetricValue(metrics []models.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	minValue := metrics[0].Value
	for _, m := range metrics {
		if m.Value < minValue {
			minValue = m.Value
		}
	}

	return minValue
}

func findCurrentMetricValue(metrics []models.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	return metrics[len(metrics)-1].Value
}

func (i *index) addItemToMenu(m models.Metrics) {
	// Store the metric in items map
	i.items[m.Name] = m

	// Regenerate the menu to show the new item
	newMenu := i.generateMenu()
	i.grid.RemoveItem(i.currentMenu)
	i.grid.AddItem(newMenu, 1, 0, 1, 1, 0, 100, false)
	i.currentMenu = newMenu
	i.app.SetFocus(i.currentMenu)
}

func textContains(text string, filterText string) bool {
	if filterText == "" {
		return true
	}
	matched, err := regexp.MatchString(filterText, strings.ToLower(text))
	if err != nil {
		return strings.Contains(strings.ToLower(text), strings.ToLower(filterText))
	}
	return matched
}

func (i *index) setSecondHeader(secondHeader string) *tview.TextView {
	headerText := fmt.Sprintf("[yellow::b]%s %s[-:-:-]\n[::d]%s[-:-:-]", defaultHeader, version, usageInstructions)
	if secondHeader != "" {
		headerText += "\n" + secondHeader
	}
	return i.header.SetText(headerText)
}

func (i *index) setRightPane(secondHeader string) *tview.TextView {
	return i.rightPane.SetText(secondHeader)
}

func (i *index) searchbarInput(r rune) {
	if r >= '!' && r <= '~' {
		text, _ := i.allItemsMenu.GetItemText(filterItemIndex)
		clearedText := strings.TrimPrefix(clearColor(text, filterColor), "Filter: ")
		clearedText = strings.TrimSuffix(clearedText, cursorChar)
		i.setFilterText(clearedText + string(r))
	}
}

func (i *index) searchbarDelete() {
	text, _ := i.allItemsMenu.GetItemText(filterItemIndex)
	clearedText := strings.TrimPrefix(clearColor(text, filterColor), "Filter: ")
	clearedText = strings.TrimSuffix(clearedText, cursorChar)
	if len(clearedText) == 0 {
		return
	}
	i.setFilterText(clearedText[:len(clearedText)-1])
}

func (i *index) searchbarClear() {
	i.currentMenu.SetItemText(filterItemIndex, addColor("Filter: "+cursorChar, filterColor), "")
	i.refreshMenuAccordingToFilterInput("")
	i.currentMenu.SetCurrentItem(filterItemIndex)
	i.app.SetFocus(i.currentMenu)
}

func (i *index) setFilterText(newFilterText string) {
	newFilterColored := newFilterText
	if len(newFilterText) > 0 {
		newFilterColored = addColor("Filter: "+newFilterText+cursorChar, filterColor)
	} else {
		newFilterColored = addColor("Filter: "+cursorChar, filterColor)
	}
	i.currentMenu.SetItemText(filterItemIndex, newFilterColored, "")
	i.refreshMenuAccordingToFilterInput(newFilterText)
}

// This function should be used as part of the searchbar functionality to show only relevant menu items
func (i *index) refreshMenuAccordingToFilterInput(input string) {
	i.userInteractionMutex.Lock()
	defer i.userInteractionMutex.Unlock()
	newMenu := i.generateMenu()

	// First add selected items at the top
	for _, selectedName := range i.selected {
		if input == "" || textContains(selectedName, input) {
			newMenu.AddItem(addColor("(*) "+selectedName, selectedItemColor), "", 0, nil)
		}
	}

	// Then add non-selected items
	if input != "" {
		itemsIndexes := i.allItemsMenu.FindItems(input, input, false, true)
		for _, itemIndex := range itemsIndexes {
			if itemIndex <= filterSeparatorIndex {
				continue
			}
			text, secondary := i.allItemsMenu.GetItemText(itemIndex)
			// Skip if this item is already added as selected
			isSelected := false
			for _, selectedName := range i.selected {
				if i.cleanItemName(text) == i.cleanItemName(selectedName) {
					isSelected = true
					break
				}
			}
			if !isSelected {
				newMenu.AddItem(text, secondary, 0, nil)
			}
		}
	}

	i.grid.RemoveItem(i.currentMenu)
	i.grid.AddItem(newMenu, 1, 0, 1, 1, 0, 100, false)
	i.currentMenu = newMenu
	newMenu.SetCurrentItem(filterItemIndex)
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
