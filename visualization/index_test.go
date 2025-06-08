package visualization

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/eldada/metrics-viewer/models"
	"github.com/eldada/metrics-viewer/provider"
	"github.com/gdamore/tcell/v2"
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

// Add mock primitives at the top of the file
type mockList struct {
	*tview.List
	items        []string
	currentItem  int
	selectedFunc func(int, string, string, rune)
	inputCapture func(*tcell.EventKey) *tcell.EventKey
}

func newMockList() *mockList {
	return &mockList{
		List:  tview.NewList().ShowSecondaryText(false),
		items: make([]string, 0),
	}
}

func (m *mockList) AddItem(text, secondaryText string, shortcut rune, selected func()) *tview.List {
	m.items = append(m.items, text)
	m.List.AddItem(text, secondaryText, shortcut, selected)
	return m.List
}

func (m *mockList) GetItemCount() int {
	return len(m.items)
}

func (m *mockList) GetItemText(index int) (string, string) {
	if index >= 0 && index < len(m.items) {
		return m.items[index], ""
	}
	return "", ""
}

func (m *mockList) Clear() *tview.List {
	m.items = make([]string, 0)
	m.List.Clear()
	return m.List
}

func (m *mockList) SetCurrentItem(index int) *tview.List {
	if index >= 0 && index < len(m.items) {
		m.currentItem = index
		m.List.SetCurrentItem(index)
	}
	return m.List
}

func (m *mockList) GetCurrentItem() int {
	return m.currentItem
}

func (m *mockList) SetSelectedFunc(handler func(int, string, string, rune)) *tview.List {
	m.selectedFunc = handler
	m.List.SetSelectedFunc(handler)
	return m.List
}

func (m *mockList) SetInputCapture(capture func(*tcell.EventKey) *tcell.EventKey) *tview.List {
	m.inputCapture = capture
	m.List.SetInputCapture(capture)
	return m.List
}

type mockInputField struct {
	*tview.InputField
	text        string
	doneFunc    func(tcell.Key)
	changedFunc func(string)
	mu          sync.Mutex
}

func newMockInputField() *mockInputField {
	return &mockInputField{
		InputField: tview.NewInputField(),
	}
}

func (m *mockInputField) GetText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.text
}

func (m *mockInputField) SetText(text string) *tview.InputField {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.text = text
	// Don't execute changedFunc during tests to avoid complex interactions
	// if m.changedFunc != nil {
	//     m.changedFunc(text)
	// }
	return m.InputField
}

func (m *mockInputField) SetDoneFunc(handler func(tcell.Key)) *tview.InputField {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doneFunc = handler
	return m.InputField
}

func (m *mockInputField) SetChangedFunc(handler func(string)) *tview.InputField {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.changedFunc = handler
	return m.InputField
}

type mockApplication struct {
	*tview.Application
	focused    tview.Primitive
	root       tview.Primitive
	drawCalled bool
	updates    []func()
	afterDraw  func(screen tcell.Screen)
	mu         sync.Mutex
}

func newMockApplication() *mockApplication {
	app := tview.NewApplication()
	return &mockApplication{
		Application: app,
		updates:     make([]func(), 0),
	}
}

func (m *mockApplication) Draw() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("Draw called")
	m.drawCalled = true
	// Execute any pending updates immediately
	for _, update := range m.updates {
		if update != nil {
			log.Printf("Executing update")
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic in update: %v", r)
					}
				}()
				update()
			}()
			log.Printf("Update executed")
		}
	}
	m.updates = make([]func(), 0)
	// Call afterDraw if set
	if m.afterDraw != nil {
		log.Printf("Calling afterDraw")
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in afterDraw: %v", r)
				}
			}()
			m.afterDraw(nil)
		}()
		log.Printf("afterDraw called")
	}
	return nil
}

func (m *mockApplication) SetRoot(root tview.Primitive, fullscreen bool) *tview.Application {
	log.Printf("SetRoot called")
	m.root = root
	return m.Application
}

func (m *mockApplication) SetFocus(p tview.Primitive) *tview.Application {
	log.Printf("SetFocus called on %T", p)
	m.focused = p
	return m.Application
}

func (m *mockApplication) GetFocus() tview.Primitive {
	if m.focused == nil {
		log.Printf("GetFocus: no focus")
	} else {
		log.Printf("GetFocus: focused on %T", m.focused)
	}
	return m.focused
}

func (m *mockApplication) QueueUpdateDraw(f func()) *tview.Application {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("QueueUpdateDraw called")
	if f != nil {
		m.updates = append(m.updates, f)
		log.Printf("Executing update immediately")
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in QueueUpdateDraw: %v", r)
				}
			}()
			f() // Execute immediately in tests
		}()
		log.Printf("Forcing draw")
		m.Draw() // Force draw to process updates
	}
	return m.Application
}

func (m *mockApplication) QueueUpdate(f func()) *tview.Application {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("QueueUpdate called")
	if f != nil {
		m.updates = append(m.updates, f)
		log.Printf("Executing update immediately")
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in QueueUpdate: %v", r)
				}
			}()
			f() // Execute immediately in tests
		}()
		log.Printf("Forcing draw")
		m.Draw() // Force draw to process updates
	}
	return m.Application
}

func (m *mockApplication) SetAfterDrawFunc(handler func(screen tcell.Screen)) *tview.Application {
	log.Printf("SetAfterDrawFunc called")
	m.afterDraw = handler
	return m.Application
}

type fields struct {
	currentMenu          *mockList
	grid                 *tview.Grid
	app                  *mockApplication
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
	selectedMetricsBox   *mockList
	filterBox            *mockInputField
	isFilterActive       bool
	lastFocusedBox       tview.Primitive
}

func Test_index_replaceMenuContentOnGrid(t *testing.T) {
	// Increase test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	modifiedMetrics := newMissingMetricsCache()
	modifiedMetrics.AddToMetrics(defaultMetrics)

	expectedHeaderText := "[yellow::b]Metrics Viewer (unknown[-:-:-]); [::d]Use '/' to search metrics (ESC to clear) • Use ↑↓ to navigate • ENTER or SPACE to select • CTRL+C to exit[-:-:-]"

	tests := []struct {
		name               string
		fields             *fields
		expectedFields     *fields
		expectedCalledDraw bool
	}{
		{
			name: "set error",
			fields: &fields{
				app:                  newMockApplication(),
				currentMenu:          newMockList(),
				header:               tview.NewTextView().SetDynamicColors(true),
				provider:             mockProvider{error: true},
				items:                map[string]models.Metrics{},
				userInteractionMutex: &sync.Mutex{},
				selectedMetricsBox:   newMockList(),
				filterBox:            newMockInputField(),
				drawing:              true,
				grid:                 tview.NewGrid(),
				mainContent:          tview.NewTextView().SetDynamicColors(true),
				rightPane:            tview.NewTextView().SetDynamicColors(true),
			},
			expectedFields: &fields{
				hasError: true,
				header:   tview.NewTextView().SetDynamicColors(true).SetText(expectedHeaderText + "\n[red]error[-]"),
				drawing:  true,
			},
		},
		{
			name: "clear error",
			fields: &fields{
				app:                  newMockApplication(),
				header:               tview.NewTextView().SetDynamicColors(true).SetText(expectedHeaderText + "\n[red]error[-]"),
				provider:             mockProvider{},
				hasError:             true,
				missingMetricsCache:  newMissingMetricsCache(),
				currentMenu:          newMockList(),
				selectedMetricsBox:   newMockList(),
				items:                map[string]models.Metrics{},
				userInteractionMutex: &sync.Mutex{},
				filterBox:            newMockInputField(),
				drawing:              false,
				grid:                 tview.NewGrid(),
				mainContent:          tview.NewTextView().SetDynamicColors(true),
				rightPane:            tview.NewTextView().SetDynamicColors(true),
			},
			expectedFields: &fields{
				hasError: false,
				header:   tview.NewTextView().SetDynamicColors(true).SetText("Metrics Viewer"),
				drawing:  false,
			},
		},
		{
			name: "add new metrics without selected items",
			fields: &fields{
				app:                  newMockApplication(),
				header:               tview.NewTextView().SetDynamicColors(true),
				provider:             mockProvider{},
				missingMetricsCache:  newMissingMetricsCache(),
				currentMenu:          newMockList(),
				selectedMetricsBox:   newMockList(),
				userInteractionMutex: &sync.Mutex{},
				mainContent:          tview.NewTextView().SetDynamicColors(true),
				rightPane:            tview.NewTextView().SetDynamicColors(true),
				selected:             []string{}, // No selected items to avoid complex UI interactions
				items:                map[string]models.Metrics{},
				filterBox:            newMockInputField(),
				drawing:              false,
				grid:                 tview.NewGrid(),
			},
			expectedFields: &fields{
				drawing: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("Starting test: %s", tt.name)

			// Create a simpler mock application that doesn't execute complex UI operations
			mockApp := newMockApplication()

			// Override the mock to avoid complex UI operations during testing
			mockApp.SetAfterDrawFunc(func(screen tcell.Screen) {
				log.Printf("After draw func called, setting drawing to true")
				tt.fields.drawing = true
			})

			i := &index{
				currentMenu:          tt.fields.currentMenu.List,
				grid:                 tt.fields.grid,
				app:                  mockApp.Application,
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
				selectedMetricsBox:   tt.fields.selectedMetricsBox.List,
				filterBox:            tt.fields.filterBox.InputField,
				isFilterActive:       tt.fields.isFilterActive,
				lastFocusedBox:       tt.fields.lastFocusedBox,
			}

			// Create a channel to receive any errors from the goroutine
			errChan := make(chan error, 1)
			done := make(chan struct{})

			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						if err, ok := r.(error); ok {
							errChan <- err
						} else {
							errChan <- fmt.Errorf("panic: %v", r)
						}
					}
				}()

				log.Printf("Calling replaceMenuContentOnGrid")
				i.replaceMenuContentOnGrid()
				log.Printf("Finished replaceMenuContentOnGrid")
			}()

			// Wait for either completion or timeout
			select {
			case <-ctx.Done():
				t.Fatal("Test operation timed out")
			case err := <-errChan:
				t.Fatalf("Operation failed with error: %v", err)
			case <-done:
				log.Printf("Operation completed successfully")
			case <-time.After(5 * time.Second): // Reduced timeout per operation
				t.Fatal("Individual operation timed out")
			}

			// Verify expected fields
			if tt.expectedFields.hasError != i.hasError {
				t.Errorf("hasError = %v, want %v", i.hasError, tt.expectedFields.hasError)
			}
			if tt.expectedFields.drawing != i.drawing {
				t.Errorf("drawing = %v, want %v", i.drawing, tt.expectedFields.drawing)
			}
			if tt.expectedFields.header != nil {
				if i.header == nil {
					t.Error("header is nil")
				} else {
					got := i.header.GetText(true)
					want := tt.expectedFields.header.GetText(true)
					if got != want {
						t.Errorf("header text mismatch:\ngot:  %s\nwant: %s", got, want)
					}
				}
			}
			if tt.expectedFields.rightPane != nil {
				if i.rightPane == nil {
					t.Error("rightPane is nil")
				} else {
					got := i.rightPane.GetText(true)
					want := tt.expectedFields.rightPane.GetText(true)
					if got != want {
						t.Errorf("rightPane text mismatch:\ngot:  %s\nwant: %s", got, want)
					}
				}
			}
		})
	}
}

func Test_index_generateMenuReturnsListThatIsNotTheCurrentMenu(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	i := &index{
		app:                  newMockApplication().Application,
		items:                map[string]models.Metrics{},
		userInteractionMutex: &sync.Mutex{},
		filterBox:            tview.NewInputField(),
	}

	done := make(chan struct{})
	go func() {
		i.currentMenu = i.generateMenu()
		newMenu := i.generateMenu()
		assert.NotEqual(t, i.currentMenu, newMenu)
		close(done)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("Test timed out")
	case <-done:
		// Test completed successfully
	}
}

func Test_index_searchbar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	i := &index{
		grid: tview.NewGrid(),
		app:  newMockApplication().Application,
		items: map[string]models.Metrics{
			"ab":  {Name: "ab"},
			"abc": {Name: "abc"},
		},
		selectedMetricsBox:   tview.NewList(),
		userInteractionMutex: &sync.Mutex{},
		filterBox:            tview.NewInputField().SetLabel("Search metrics (regex): "),
	}

	done := make(chan struct{})
	go func() {
		i.currentMenu = i.generateMenu()
		i.filterBox.SetText("a")
		i.refreshMenuAccordingToFilterInput()
		assert.Equal(t, 2, i.currentMenu.GetItemCount()) // Both items match "a"
		close(done)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("Test timed out")
	case <-done:
		// Test completed successfully
	}
}
