package visualization

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	error bool
}

var defaultMetrics = []models.Metrics{{
	Metrics: []models.Metric{{
		Value:     1,
		Labels:    nil,
		Timestamp: time.Unix(12345, 0),
	}},
	Description: "abc",
	Key:         "hello",
	Name:        "hello_abc",
}}

func (m mockProvider) Get() ([]models.Metrics, error) {
	if m.error {
		return nil, errors.New("error")
	}

	return defaultMetrics, nil
}

func Test_index_replaceMenuContentOnGrid(t *testing.T) {
	modifiedMetrics := newMissingMetricsCache()
	modifiedMetrics.AddToMetrics(defaultMetrics)
	type fields struct {
		currentMenu          *tview.List
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
		selectedMetricsBox   *tview.List
	}
	tests := []struct {
		name               string
		fields             *fields
		expectedFields     *fields
		expectedCalledDraw bool
	}{
		{
			name: "set error",
			fields: &fields{
				header:   tview.NewTextView(),
				provider: mockProvider{error: true},
			},
			expectedFields: &fields{
				hasError: true,
				header:   tview.NewTextView().SetText("[yellow::b]Metrics Viewer (unknown)[-:-:-]\n[::d]Use '/' to search metrics (supports regex) • Use ↑↓ to navigate • ENTER to select • ESC to clear • Quit or CTRL+C to exit[-:-:-]\n[red]error[-]"),
			},
		},
		{
			name: "clear error",
			fields: &fields{
				header:              tview.NewTextView().SetText("[yellow::b]Metrics Viewer (unknown)[-:-:-]\n[::d]Use '/' to search metrics (supports regex) • Use ↑↓ to navigate • ENTER to select • ESC to clear • Quit or CTRL+C to exit[-:-:-]\n[red]error[-]"),
				provider:            mockProvider{},
				hasError:            true,
				missingMetricsCache: newMissingMetricsCache(),
				currentMenu:         tview.NewList(),
				selectedMetricsBox:  tview.NewList(),
				items:               map[string]models.Metrics{},
			},
			expectedFields: &fields{
				hasError: false,
				header:   tview.NewTextView().SetText("[yellow::b]Metrics Viewer (unknown)[-:-:-]\n[::d]Use '/' to search metrics (supports regex) • Use ↑↓ to navigate • ENTER to select • ESC to clear • Quit or CTRL+C to exit[-:-:-]"),
			},
		},
		{
			name: "add metrics to cache",
			fields: &fields{
				header:              tview.NewTextView(),
				provider:            mockProvider{},
				missingMetricsCache: newMissingMetricsCache(),
				currentMenu:         tview.NewList(),
				selectedMetricsBox:  tview.NewList(),
				items:               map[string]models.Metrics{},
			},
			expectedFields: &fields{
				missingMetricsCache: modifiedMetrics,
			},
		},
		{
			name: "set description",
			fields: &fields{
				header:              tview.NewTextView(),
				provider:            mockProvider{},
				missingMetricsCache: newMissingMetricsCache(),
				currentMenu:         tview.NewList(),
				selectedMetricsBox:  tview.NewList(),
				items: map[string]models.Metrics{
					"hello_abc": {Metrics: defaultMetrics[0].Metrics, Name: "hello", Key: "hello_abc"},
				},
			},
			expectedFields: &fields{
				items: map[string]models.Metrics{
					"hello_abc": defaultMetrics[0], // including description
				},
			},
		},
		{
			name: "keep selection on update",
			fields: &fields{
				header:               tview.NewTextView(),
				provider:             mockProvider{},
				missingMetricsCache:  newMissingMetricsCache(),
				currentMenu:          tview.NewList(),
				selectedMetricsBox:   tview.NewList(),
				userInteractionMutex: &sync.Mutex{},
				mainContent:          tview.NewTextView(),
				rightPane:            tview.NewTextView(),
				selected:             []string{"hello_abc"},
				items: map[string]models.Metrics{
					"hello_abc": {Metrics: defaultMetrics[0].Metrics, Name: "hello", Key: "hello_abc"},
				},
			},
			expectedFields: &fields{
				selected: []string{"hello_abc"},
			},
		},
		{
			name: "set right pane on selected item",
			fields: &fields{
				header:               tview.NewTextView(),
				provider:             mockProvider{},
				missingMetricsCache:  newMissingMetricsCache(),
				currentMenu:          tview.NewList(),
				selectedMetricsBox:   tview.NewList(),
				userInteractionMutex: &sync.Mutex{},
				mainContent:          tview.NewTextView(),
				rightPane:            tview.NewTextView(),
				selected:             []string{"hello_abc"},
				items: map[string]models.Metrics{
					"hello_abc": {Metrics: defaultMetrics[0].Metrics, Name: "hello", Key: "hello_abc"},
				},
			},
			expectedFields: &fields{
				rightPane: tview.NewTextView().SetText("[green][green]hello_abc[-]\n[green]No description[-]\n[green]Max:     1.000000[-]\n[green]Min:     1.000000[-]\n[green]Current: 1.000000[-]\n[-]"),
			},
		},
		{
			name: "add new item to existing menu",
			fields: &fields{
				header:              tview.NewTextView(),
				provider:            mockProvider{},
				missingMetricsCache: newMissingMetricsCache(),
				currentMenu:         tview.NewList(),
				selectedMetricsBox:  tview.NewList(),
				items: map[string]models.Metrics{
					"new_item": {Metrics: defaultMetrics[0].Metrics, Name: "new", Key: "new_item"},
				},
			},
			expectedFields: &fields{
				items: map[string]models.Metrics{
					"new_item":  {Metrics: defaultMetrics[0].Metrics, Name: "new", Key: "new_item"},
					"hello_abc": defaultMetrics[0],
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &index{
				currentMenu:          tt.fields.currentMenu,
				grid:                 tt.fields.grid,
				app:                  tt.fields.app,
				mainContent:          tt.fields.mainContent,
				provider:             tt.fields.provider,
				missingMetricsCache:  tt.fields.missingMetricsCache,
				header:               tt.fields.header,
				selected:             tt.fields.selected,
				items:                tt.fields.items,
				hasError:             tt.fields.hasError,
				drawing:              tt.fields.drawing,
				userInteractionMutex: tt.fields.userInteractionMutex,
				rightPane:            tt.fields.rightPane,
				selectedMetricsBox:   tt.fields.selectedMetricsBox,
			}

			i.replaceMenuContentOnGrid()

			assertEqualIfNotNil(t, tt.expectedFields.currentMenu, i.currentMenu, "currentMenu")
			assertEqualIfNotNil(t, tt.expectedFields.selectedMetricsBox, i.selectedMetricsBox, "selectedMetricsBox")
			assertEqualIfNotNil(t, tt.expectedFields.grid, i.grid, "grid")
			assertEqualIfNotNil(t, tt.expectedFields.app, i.app, "app")
			assertEqualIfNotNil(t, tt.expectedFields.mainContent, i.mainContent, "mainContent")
			assertEqualIfNotNil(t, tt.expectedFields.provider, i.provider, "provider")
			assertEqualIfNotNil(t, tt.expectedFields.missingMetricsCache, i.missingMetricsCache, "missingMetricsCache")
			assertEqualIfNotNil(t, tt.expectedFields.header, i.header, "header")
			assertEqualIfNotNil(t, tt.expectedFields.selected, i.selected, "selected")
			assertEqualIfNotNil(t, tt.expectedFields.items, i.items, "items")
			assertEqualIfNotNil(t, tt.expectedFields.hasError, i.hasError, "hasError")
			assertEqualIfNotNil(t, tt.expectedFields.drawing, i.drawing, "drawing")
			assertEqualIfNotNil(t, tt.expectedFields.userInteractionMutex, i.userInteractionMutex, "userInteractionMutex")
			assertEqualIfNotNil(t, tt.expectedFields.rightPane, i.rightPane, "rightPane")
		})
	}
}

func Test_index_generateMenuReturnsListThatIsNotTheCurrentMenu(t *testing.T) {
	i := &index{}
	i.currentMenu = i.generateMenu()
	newMenu := i.generateMenu()
	assert.NotEqual(t, i.currentMenu, newMenu)
}

func Test_index_searchbar(t *testing.T) {
	i := &index{
		grid: tview.NewGrid(),
		app:  tview.NewApplication(),
		items: map[string]models.Metrics{
			"ab":  {Name: "ab"},
			"abc": {Name: "abc"},
		},
		selectedMetricsBox:   tview.NewList(),
		userInteractionMutex: &sync.Mutex{},
		filterBox:            tview.NewInputField().SetLabel("Search metrics (regex): "),
	}

	i.currentMenu = i.generateMenu()
	i.filterBox.SetText("a")
	i.refreshMenuAccordingToFilterInput()
	assert.Equal(t, 2, i.currentMenu.GetItemCount()) // Both items match "a"

	i.filterBox.SetText("ab")
	i.refreshMenuAccordingToFilterInput()
	assert.Equal(t, 2, i.currentMenu.GetItemCount()) // Both items match "ab"

	i.filterBox.SetText("abc")
	i.refreshMenuAccordingToFilterInput()
	assert.Equal(t, 1, i.currentMenu.GetItemCount()) // Only one item matches "abc"

	i.filterBox.SetText("ab")
	i.refreshMenuAccordingToFilterInput()
	assert.Equal(t, 2, i.currentMenu.GetItemCount()) // Back to both items matching "ab"

	i.filterBox.SetText("")
	i.refreshMenuAccordingToFilterInput()
	assert.Equal(t, 2, i.currentMenu.GetItemCount()) // All items shown when filter is empty
}

func assertEqualIfNotNil(t *testing.T, expected interface{}, actual interface{}, fieldName string) {
	if expected != nil {
		switch expected.(type) {
		case bool:
			assert.Equal(t, expected, actual, fieldName)
		case *tview.TextView:
			if !reflect.ValueOf(expected).IsNil() {
				assert.Equal(t, expected.(*tview.TextView).GetText(false), actual.(*tview.TextView).GetText(false), fieldName)
			}
		case missingMetricsCache:
			if !reflect.ValueOf(expected).IsNil() {
				assert.Equal(t, len(expected.(missingMetricsCache)), len(actual.(missingMetricsCache)), fieldName)
			}
		default:
			if !reflect.ValueOf(expected).IsNil() {
				assert.Equal(t, expected, actual, fieldName)
			}
		}
	}
}
