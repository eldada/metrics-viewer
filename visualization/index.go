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
	selectedMetricsBox   *tview.List
	grid                 *tview.Grid
	app                  *tview.Application
	mainContent          *tview.TextView
	provider             provider.Provider
	missingMetricsCache  missingMetricsCache
	header               *tview.TextView
	selected             []string
	items                map[string]models.Metrics
	hasError             bool
	drawing              bool
	userInteractionMutex sync.Locker
	rightPane            *tview.TextView
	filterBox            *tview.InputField
	isFilterActive       bool
	lastFocusedBox       tview.Primitive // Track which box had focus
	lastSelectedBoxIndex int             // Track selected item in Selected Metrics box
	updatingSelectedBox  bool            // Guard against recursive updates
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
const usageInstructions = "Use '/' to search metrics (supports regex) • Use ↑↓ to navigate • ENTER to select • ESC to clear • CTRL+C to exit"
const ignoreSecondaryText = "---N/A---"
const highlightColor = "[lightgray::b](x) "
const selectedItemColor = "[green::b]"
const thinSeparatorLine = "───────────────────────────────────────────────────────────────"

var colors = []string{"[green]", "[yellow]", "[blue]", "[teal]", "[gray]", "[gold]", "[indigo]", "[lavender]"}

// Main function to create the application
func (i *index) Present(ctx context.Context, interval time.Duration, prov provider.Provider) {
	i.provider = prov
	i.app = tview.NewApplication()
	// To customize the selection background color, you could add:
	// tview.Styles.PrimitiveBackgroundColor = tcell.ColorDarkBlue
	// tview.Styles.ContrastBackgroundColor = tcell.ColorBlue
	// tview.Styles.MoreContrastBackgroundColor = tcell.ColorNavy
	i.mainContent = tview.NewTextView().SetDynamicColors(true)
	i.rightPane = tview.NewTextView().SetDynamicColors(true)
	i.header = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetText("[yellow::b]" + defaultHeader + " (" + version + "[-:-:-]); [::d]" + usageInstructions + "[-:-:-]")

	// Create the filter box
	i.filterBox = tview.NewInputField().
		SetLabel("Search metrics (regex): ").
		SetFieldWidth(0). // Allow full width
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				i.isFilterActive = false
				i.app.SetFocus(i.currentMenu)
			} else if key == tcell.KeyEscape {
				i.isFilterActive = false
				i.filterBox.SetText("")
				i.refreshMenuAccordingToFilterInput()
				i.app.SetFocus(i.currentMenu)
			}
		}).
		SetChangedFunc(func(text string) {
			i.refreshMenuAccordingToFilterInput()
		})

	// Create the selected metrics box
	i.selectedMetricsBox = tview.NewList()
	i.selectedMetricsBox.ShowSecondaryText(false)
	i.selectedMetricsBox.SetBorderPadding(0, 0, 1, 1)
	i.selectedMetricsBox.SetTitle("Selected Metrics")
	i.selectedMetricsBox.SetTitleAlign(tview.AlignLeft)
	i.selectedMetricsBox.SetBorder(true)
	i.selectedMetricsBox.SetHighlightFullLine(true)

	i.selectedMetricsBox.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) {
		if name != thinSeparatorLine {
			name = i.cleanItemName(name)
			i.selectedFunc(name)
		}
	})

	// Set up navigation handler once
	i.selectedMetricsBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentItem := i.selectedMetricsBox.GetCurrentItem()
		itemCount := i.selectedMetricsBox.GetItemCount()

		if event.Key() == tcell.KeyDown {
			// If at the bottom, move to available metrics box
			if currentItem == itemCount-1 {
				i.app.SetFocus(i.currentMenu)
				i.currentMenu.SetCurrentItem(0)
				return nil
			}
			// Let tview handle normal down navigation
			return event
		} else if event.Key() == tcell.KeyUp {
			// If we're in the available metrics and at the top, try to move to selected metrics
			if i.app.GetFocus() == i.currentMenu &&
				i.currentMenu.GetCurrentItem() == 0 &&
				i.selectedMetricsBox.GetItemCount() > 0 {
				i.app.SetFocus(i.selectedMetricsBox)
				i.selectedMetricsBox.SetCurrentItem(i.selectedMetricsBox.GetItemCount() - 1)
				return nil
			}
			// Let tview handle normal up navigation
			return event
		}

		return event
	})

	// Create the available metrics menu
	i.currentMenu = tview.NewList()
	i.currentMenu.ShowSecondaryText(false)
	i.currentMenu.SetBorderPadding(0, 0, 1, 1)
	i.currentMenu.SetTitle("Available Metrics")
	i.currentMenu.SetTitleAlign(tview.AlignLeft)
	i.currentMenu.SetBorder(true)
	i.currentMenu.SetHighlightFullLine(true)

	i.currentMenu.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) {
		if name != thinSeparatorLine {
			i.selectedFunc(name)
		}
	})

	i.grid = tview.NewGrid().
		SetRows(3, 0). // Header height for title and filter only
		SetColumns(-4, -10, -3).
		SetMinSize(0, 30).
		SetBorders(true).
		SetBordersColor(tcell.ColorGreen)

	// Create a flex layout for header and filter
	headerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(i.header, 1, 0, false).
		AddItem(i.filterBox, 1, 0, false)

	i.grid.AddItem(headerFlex, 0, 0, 1, 3, 0, 0, false)

	i.grid.SetBackgroundColor(tcell.ColorBlack)

	// Create a flex layout for the left panel with fixed height for selected metrics
	leftPanel := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(i.selectedMetricsBox, 7, 0, false). // Height of 7 shows 5 items (5 + border + title)
		AddItem(i.currentMenu, 0, 1, true)          // Available metrics takes remaining space

	i.grid.AddItem(leftPanel, 1, 0, 1, 1, 0, 100, false)
	i.grid.AddItem(i.mainContent, 1, 1, 1, 1, 0, 100, false)
	i.grid.AddItem(i.rightPane, 1, 2, 1, 1, 0, 100, false)

	i.app = i.app.SetRoot(i.grid, true).SetFocus(i.currentMenu)
	go i.updateMenuOnGrid(ctx, interval)
	i.replaceMenuContentOnGrid()
	i.app.SetAfterDrawFunc(func(screen tcell.Screen) {
		i.drawing = true
	})

	// Set up global keyboard handling
	i.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			if event.Rune() == '/' {
				i.isFilterActive = true
				i.app.SetFocus(i.filterBox)
				return nil
			}
		case tcell.KeyCtrlC:
			if closer, ok := i.provider.(io.Closer); ok {
				_ = closer.Close()
			}
			i.app.Stop()
			return nil
		case tcell.KeyUp:
			// If we're in the available metrics and at the top, try to move to selected metrics
			if i.app.GetFocus() == i.currentMenu &&
				i.currentMenu.GetCurrentItem() == 0 &&
				i.selectedMetricsBox.GetItemCount() > 0 {
				i.app.SetFocus(i.selectedMetricsBox)
				i.selectedMetricsBox.SetCurrentItem(i.selectedMetricsBox.GetItemCount() - 1)
				return nil
			}
		case tcell.KeyDown:
			// If we're in the selected metrics and at the bottom, move to available metrics
			if i.app.GetFocus() == i.selectedMetricsBox &&
				i.selectedMetricsBox.GetCurrentItem() == i.selectedMetricsBox.GetItemCount()-1 {
				i.app.SetFocus(i.currentMenu)
				i.currentMenu.SetCurrentItem(0)
				return nil
			}
		}
		return event
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

	// Store current focus and selection states
	i.lastFocusedBox = i.app.GetFocus()
	if i.lastFocusedBox == i.selectedMetricsBox {
		i.lastSelectedBoxIndex = i.selectedMetricsBox.GetCurrentItem()
	}

	// Store current selection before update
	var currentText string
	if i.currentMenu != nil && i.currentMenu.GetItemCount() > 0 {
		currentItem := i.currentMenu.GetCurrentItem()
		currentText, _ = i.currentMenu.GetItemText(currentItem)
	}

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
		// Try to find the same item in the new menu
		cleanedText := i.cleanItemName(currentText)
		found := false
		if cleanedText != "" {
			for idx := 0; idx < i.currentMenu.GetItemCount(); idx++ {
				text, _ := i.currentMenu.GetItemText(idx)
				if i.cleanItemName(text) == cleanedText {
					i.currentMenu.SetCurrentItem(idx)
					found = true
					break
				}
			}
		}
		// If item not found (e.g. was removed), stay at current position if valid
		if !found && i.currentMenu.GetItemCount() > 0 {
			i.currentMenu.SetCurrentItem(0)
		}

		// Restore selected item position in Selected Metrics box if it was focused
		if i.lastFocusedBox == i.selectedMetricsBox && i.selectedMetricsBox.GetItemCount() > 0 {
			// Ensure the index is valid for the current number of items
			if i.lastSelectedBoxIndex >= i.selectedMetricsBox.GetItemCount() {
				i.lastSelectedBoxIndex = i.selectedMetricsBox.GetItemCount() - 1
			}
			i.selectedMetricsBox.SetCurrentItem(i.lastSelectedBoxIndex)
		}

		// Restore focus to filter box if it was active
		if i.isFilterActive {
			i.app.SetFocus(i.filterBox)
		} else if i.lastFocusedBox != nil {
			// Restore focus to the last focused box
			i.app.SetFocus(i.lastFocusedBox)
		}

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

	// Update both the selected metrics box and available metrics menu
	i.updateSelectedMetricsBox()

	// Only refresh the available metrics if we're not filtering
	if !i.isFilterActive {
		i.refreshMenuAccordingToFilterInput()
	}
}

// Generating a new menu to replace, useful for the searchbar capability
func (i *index) generateMenu() *tview.List {
	menu := tview.NewList()
	menu.ShowSecondaryText(false)
	menu.SetBorderPadding(0, 0, 1, 1)
	menu.SetHighlightFullLine(true)

	menu.SetSelectedFunc(func(index int, name string, secondaryName string, shortcut rune) {
		if name != thinSeparatorLine {
			i.selectedFunc(name)
		}
	})

	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentItem := menu.GetCurrentItem()

		// Handle navigation
		if event.Key() == tcell.KeyUp {
			// If at the top and there are selected items, move to selected box
			if currentItem == 0 && i.selectedMetricsBox.GetItemCount() > 0 {
				// Move focus to selected box and select the last item
				i.app.SetFocus(i.selectedMetricsBox)
				i.selectedMetricsBox.SetCurrentItem(i.selectedMetricsBox.GetItemCount() - 1)
				return nil
			}

			// Move up if not at top
			if currentItem > 0 {
				menu.SetCurrentItem(currentItem - 1)
			}
			return nil

		} else if event.Key() == tcell.KeyDown {
			itemCount := menu.GetItemCount()

			// Move down if not at bottom
			if currentItem < itemCount-1 {
				menu.SetCurrentItem(currentItem + 1)
			}
			return nil
		}

		return event
	})

	// Sort non-selected items
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

	// Get current filter text
	filterText := i.filterBox.GetText()

	// Add non-selected items (filtered)
	for _, name := range nonSelectedItems {
		if filterText == "" || textContains(name, filterText) {
			menu.AddItem(name, "", 0, nil)
		}
	}

	return menu
}

// Helper function to clean item names from all formatting and prefixes
func (i *index) cleanItemName(name string) string {
	// Remove all instances of "(*) " prefix - keep removing until none left
	for strings.Contains(name, "(*) ") {
		name = strings.Replace(name, "(*) ", "", 1)
	}
	// Remove any remaining prefix patterns
	name = strings.TrimPrefix(name, "(*)")
	name = strings.TrimSpace(name)

	// Remove any color formatting - remove all tview color tags
	// Pattern: [color:background:attributes]
	re := regexp.MustCompile(`\[[^\]]*\]`)
	name = re.ReplaceAllString(name, "")

	// Also remove specific color patterns we know about
	name = clearColor(name, selectedItemColor)
	name = clearColor(name, highlightColor)

	return strings.TrimSpace(name)
}

// Reacting to the user selection
func (i *index) selectedFunc(name string) {
	// Clean the name from any existing formatting and prefixes
	name = i.cleanItemName(name)

	// Store current focus before updates
	i.lastFocusedBox = i.app.GetFocus()

	i.toggleSelected(name)

	_, _, width, height := i.mainContent.GetInnerRect()
	summary, selectedMetrics := i.selectedToList()
	res := NewGraph().SprintOnce(width, height, selectedMetrics...)
	i.mainContent.SetText(replaceColors(res))
	i.setRightPane(summary)
	i.hasError = false

	// Update both boxes
	// Use goroutine to prevent hang but without delay for responsiveness
	go func() {
		i.updateSelectedMetricsBox()
	}()

	// Only refresh the available metrics if we're not filtering
	if !i.isFilterActive {
		i.refreshMenuAccordingToFilterInput()
	} else {
		// If we're filtering, just refresh with current filter
		i.refreshMenuAccordingToFilterInput()
	}

	// If no selected items remain and we were in the selected box, move to available metrics
	if len(i.selected) == 0 && i.lastFocusedBox == i.selectedMetricsBox {
		i.app.SetFocus(i.currentMenu)
		if i.currentMenu.GetItemCount() > 0 {
			i.currentMenu.SetCurrentItem(0)
		}
		i.lastFocusedBox = i.currentMenu
	} else if i.isFilterActive {
		i.app.SetFocus(i.filterBox)
		i.lastFocusedBox = i.filterBox
	} else {
		// Restore focus to where it was
		i.app.SetFocus(i.lastFocusedBox)
	}
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
	if i.currentMenu != i.selectedMetricsBox {
		i.findAndUpdateItemText(i.selectedMetricsBox, name, converter)
	}
}

func (i *index) removeSelectedItemColor(name string) {
	converter := func(main string) string {
		return i.cleanItemName(main)
	}
	modifiedName := i.cleanItemName(name)
	i.findAndUpdateItemText(i.currentMenu, modifiedName, converter)
	if i.selectedMetricsBox != i.currentMenu {
		i.findAndUpdateItemText(i.selectedMetricsBox, modifiedName, converter)
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

	// Only refresh the menu if we're not filtering
	if !i.isFilterActive {
		i.refreshMenuAccordingToFilterInput()
	}
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
	headerText := fmt.Sprintf("[yellow::b]%s (%s[-:-:-]); [::d]%s[-:-:-]", defaultHeader, version, usageInstructions)
	if secondHeader != "" {
		headerText += "\n" + secondHeader
	}
	return i.header.SetText(headerText)
}

func (i *index) setRightPane(secondHeader string) *tview.TextView {
	return i.rightPane.SetText(secondHeader)
}

func (i *index) updateSelectedMetricsBox() {
	// Guard against recursive calls
	if i.updatingSelectedBox {
		return
	}
	i.updatingSelectedBox = true
	defer func() {
		i.updatingSelectedBox = false
	}()

	i.selectedMetricsBox.Clear()

	// Sort selected items
	sortedSelected := make([]string, len(i.selected))
	copy(sortedSelected, i.selected)
	sort.Strings(sortedSelected)

	// Add selected items with consistent green formatting
	for _, selectedName := range sortedSelected {
		// Always use green bold formatting for consistency
		displayText := fmt.Sprintf("[green::b](*) %s[-]", selectedName)
		i.selectedMetricsBox.AddItem(displayText, "", 0, nil)
	}
}

func (i *index) refreshMenuAccordingToFilterInput() {
	i.userInteractionMutex.Lock()
	defer i.userInteractionMutex.Unlock()

	// Generate new menu (filter is now handled in generateMenu)
	newMenu := i.generateMenu()

	// Replace the menu content without recreating the grid item
	i.currentMenu.Clear()
	for idx := 0; idx < newMenu.GetItemCount(); idx++ {
		text, secondary := newMenu.GetItemText(idx)
		i.currentMenu.AddItem(text, secondary, 0, nil)
	}

	// Keep focus on filter box while filtering
	if i.isFilterActive {
		i.app.SetFocus(i.filterBox)
	} else {
		i.app.SetFocus(i.currentMenu)
	}
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
