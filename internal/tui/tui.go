// Package tui implements the interactive Bubble Tea interface for snippetbox:
// a filterable snippet list on the left, a syntax-highlighted preview on the
// right, and one-key copy to the clipboard.
package tui

import (
	"fmt"
	"strings"

	"github.com/ATOM00blue/snippetbox/internal/search"
	"github.com/ATOM00blue/snippetbox/internal/snippet"
	"github.com/ATOM00blue/snippetbox/internal/store"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Run starts the TUI bound to the given store and blocks until the user quits.
func Run(s *store.Store) error {
	m := newModel(s)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ---- styles ---------------------------------------------------------------

var (
	colorAccent  = lipgloss.Color("213")
	colorMuted   = lipgloss.Color("244")
	colorBorder  = lipgloss.Color("240")
	colorWarn    = lipgloss.Color("203")
	colorSuccess = lipgloss.Color("120")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(colorAccent).
			Padding(0, 1)

	previewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	previewHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	metaStyle = lipgloss.NewStyle().Foreground(colorMuted)

	statusStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	errorStyle  = lipgloss.NewStyle().Foreground(colorWarn)

	helpStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

// ---- list item ------------------------------------------------------------

type item struct {
	snip snippet.Snippet
}

func (i item) Title() string { return i.snip.Title }

func (i item) Description() string {
	parts := []string{}
	if i.snip.Language != "" {
		parts = append(parts, i.snip.Language)
	}
	if len(i.snip.Tags) > 0 {
		parts = append(parts, "#"+strings.Join(i.snip.Tags, " #"))
	}
	if len(parts) == 0 {
		return firstLine(i.snip.Content)
	}
	return strings.Join(parts, "  ")
}

// FilterValue powers the list's built-in filtering. We include title, tags,
// language and content so filtering matches across the whole snippet.
func (i item) FilterValue() string { return i.snip.Haystack() }

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// ---- mode -----------------------------------------------------------------

type mode int

const (
	modeBrowse mode = iota
	modeForm
	modeConfirmDelete
)

// ---- keymap ---------------------------------------------------------------

type keyMap struct {
	Copy    key.Binding
	Add     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Focus   key.Binding
	Quit    key.Binding
	PrevTab key.Binding
}

var keys = keyMap{
	Copy:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
	Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
	Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Focus:   key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus preview")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	PrevTab: key.NewBinding(key.WithKeys("shift+tab")),
}

// ---- model ----------------------------------------------------------------

type focus int

const (
	focusList focus = iota
	focusPreview
)

type model struct {
	store   *store.Store
	list    list.Model
	preview viewport.Model
	form    formModel

	mode    mode
	focused focus

	width, height int
	ready         bool

	status    string
	statusErr bool

	deleteID string
}

func newModel(s *store.Store) model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(colorAccent).BorderForeground(colorAccent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(colorAccent).BorderForeground(colorAccent)

	l := list.New(nil, delegate, 0, 0)
	l.Title = "snippetbox"
	l.Styles.Title = titleStyle
	l.SetShowHelp(false)
	l.SetStatusBarItemName("snippet", "snippets")

	m := model{
		store:   s,
		list:    l,
		preview: viewport.New(0, 0),
		mode:    modeBrowse,
		focused: focusList,
	}
	m.reloadItems("")
	return m
}

// reloadItems rebuilds the list from the store, preserving newest-first order.
func (m *model) reloadItems(selectID string) {
	results := search.Search(m.store.All(), "")
	items := make([]list.Item, 0, len(results))
	selectIdx := -1
	for i, r := range results {
		items = append(items, item{snip: r.Snippet})
		if selectID != "" && r.Snippet.ID == selectID {
			selectIdx = i
		}
	}
	m.list.SetItems(items)
	if selectIdx >= 0 {
		m.list.Select(selectIdx)
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) currentSnippet() (snippet.Snippet, bool) {
	it, ok := m.list.SelectedItem().(item)
	if !ok {
		return snippet.Snippet{}, false
	}
	return it.snip, true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		m.refreshPreview()
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeForm:
			return m.updateForm(msg)
		case modeConfirmDelete:
			return m.updateConfirm(msg)
		default:
			return m.updateBrowse(msg)
		}
	}

	// Non-key messages still flow to the active component.
	switch m.mode {
	case modeForm:
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	default:
		return m.propagate(msg)
	}
}

func (m model) updateBrowse(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While the list is actively filtering, let it consume all keys so the
	// user can type a query without our shortcuts hijacking letters.
	if m.list.SettingFilter() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.refreshPreview()
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Focus):
		if m.focused == focusList {
			m.focused = focusPreview
		} else {
			m.focused = focusList
		}
		return m, nil

	case key.Matches(msg, keys.Copy):
		return m.doCopy()

	case key.Matches(msg, keys.Add):
		m.mode = modeForm
		m.form = newForm(formAdd, snippet.Snippet{}, m.formWidth(), m.formHeight())
		return m, m.form.focusFirst()

	case key.Matches(msg, keys.Edit):
		if sn, ok := m.currentSnippet(); ok {
			m.mode = modeForm
			m.form = newForm(formEdit, sn, m.formWidth(), m.formHeight())
			return m, m.form.focusFirst()
		}
		return m, nil

	case key.Matches(msg, keys.Delete):
		if sn, ok := m.currentSnippet(); ok {
			m.mode = modeConfirmDelete
			m.deleteID = sn.ID
			m.status = fmt.Sprintf("delete %q? (y/n)", sn.Title)
			m.statusErr = false
		}
		return m, nil
	}

	// If the preview is focused, scroll it; otherwise drive the list.
	if m.focused == focusPreview {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	}

	prevIdx := m.list.Index()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.list.Index() != prevIdx {
		m.refreshPreview()
	}
	return m, cmd
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		id := m.deleteID
		m.mode = modeBrowse
		if err := m.store.Delete(id); err != nil {
			m.setStatus("delete failed: "+err.Error(), true)
		} else {
			m.reloadItems("")
			m.refreshPreview()
			m.setStatus("deleted", false)
		}
		return m, nil
	default:
		m.mode = modeBrowse
		m.setStatus("", false)
		return m, nil
	}
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBrowse
		m.setStatus("cancelled", false)
		return m, nil
	case "ctrl+s":
		return m.saveForm()
	}
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m model) saveForm() (tea.Model, tea.Cmd) {
	sn, err := m.form.snippet()
	if err != nil {
		m.setStatus(err.Error(), true)
		return m, nil
	}
	if m.form.kind == formAdd {
		saved, aerr := m.store.Add(sn)
		if aerr != nil {
			m.setStatus("save failed: "+aerr.Error(), true)
			return m, nil
		}
		m.mode = modeBrowse
		m.reloadItems(saved.ID)
		m.refreshPreview()
		m.setStatus("added "+saved.Title, false)
		return m, nil
	}
	if uerr := m.store.Update(sn); uerr != nil {
		m.setStatus("save failed: "+uerr.Error(), true)
		return m, nil
	}
	m.mode = modeBrowse
	m.reloadItems(sn.ID)
	m.refreshPreview()
	m.setStatus("saved "+sn.Title, false)
	return m, nil
}

func (m model) doCopy() (tea.Model, tea.Cmd) {
	sn, ok := m.currentSnippet()
	if !ok {
		m.setStatus("nothing to copy", true)
		return m, nil
	}
	if err := clipboard.WriteAll(sn.Content); err != nil {
		m.setStatus("clipboard error: "+err.Error(), true)
		return m, nil
	}
	m.setStatus("copied "+sn.Title+" to clipboard", false)
	return m, nil
}

func (m model) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

// ---- layout & view --------------------------------------------------------

func (m *model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	listWidth := m.width * 2 / 5
	if listWidth < 24 {
		listWidth = 24
	}
	if listWidth > m.width-20 {
		listWidth = m.width - 20
	}
	bodyHeight := m.height - 2 // reserve a line for the status/help bar
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	m.list.SetSize(listWidth, bodyHeight)

	// Preview occupies the remaining width, minus its border + padding (4).
	pvWidth := m.width - listWidth - 4
	if pvWidth < 10 {
		pvWidth = 10
	}
	pvHeight := bodyHeight - 4 // border (2) + header lines (2)
	if pvHeight < 3 {
		pvHeight = 3
	}
	m.preview.Width = pvWidth
	m.preview.Height = pvHeight
}

func (m *model) refreshPreview() {
	sn, ok := m.currentSnippet()
	if !ok {
		m.preview.SetContent(metaStyle.Render("No snippet selected."))
		return
	}
	m.preview.SetContent(highlight(sn.Content, sn.Language))
	m.preview.GotoTop()
}

func (m *model) formWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	if w > 100 {
		w = 100
	}
	return w
}

func (m *model) formHeight() int {
	h := m.height - 4
	if h < 8 {
		h = 8
	}
	return h
}

func (m model) View() string {
	if !m.ready {
		return "loading snippetbox..."
	}
	if m.mode == modeForm {
		return m.form.View()
	}

	left := m.list.View()
	right := m.previewPane()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, body, m.statusBar())
}

func (m model) previewPane() string {
	header := previewHeader.Render("Preview")
	meta := ""
	if sn, ok := m.currentSnippet(); ok {
		header = previewHeader.Render(sn.Title)
		bits := []string{"id " + sn.ID}
		if sn.Language != "" {
			bits = append(bits, "lang "+sn.Language)
		}
		if len(sn.Tags) > 0 {
			bits = append(bits, "#"+strings.Join(sn.Tags, " #"))
		}
		meta = metaStyle.Render(strings.Join(bits, "  •  "))
	}
	inner := lipgloss.JoinVertical(lipgloss.Left, header, meta, m.preview.View())
	return previewBorder.Width(m.preview.Width).Render(inner)
}

func (m model) statusBar() string {
	if m.mode == modeConfirmDelete {
		return errorStyle.Render(m.status)
	}
	if m.status != "" {
		if m.statusErr {
			return errorStyle.Render(m.status)
		}
		return statusStyle.Render(m.status)
	}
	help := "/ filter • a add • e edit • d delete • c copy • tab focus • q quit"
	return helpStyle.Render(help)
}
