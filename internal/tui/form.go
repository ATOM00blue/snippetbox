package tui

import (
	"errors"
	"strings"

	"github.com/ATOM00blue/snippetbox/internal/snippet"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type formKind int

const (
	formAdd formKind = iota
	formEdit
)

// form field indices.
const (
	fieldTitle = iota
	fieldLang
	fieldTags
	fieldContent
	fieldCount
)

var (
	formLabelStyle = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	formTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	formBox        = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)
)

type formModel struct {
	kind    formKind
	id      string
	created bool // whether we already have a created_at to preserve

	title   textinput.Model
	lang    textinput.Model
	tags    textinput.Model
	content textarea.Model

	focusIdx int
	width    int
	height   int

	orig snippet.Snippet
}

func newForm(kind formKind, sn snippet.Snippet, width, height int) formModel {
	title := textinput.New()
	title.Placeholder = "Title"
	title.CharLimit = 200
	title.Prompt = ""

	lang := textinput.New()
	lang.Placeholder = "language (e.g. go, bash, python)"
	lang.CharLimit = 40
	lang.Prompt = ""

	tags := textinput.New()
	tags.Placeholder = "comma, separated, tags"
	tags.CharLimit = 200
	tags.Prompt = ""

	content := textarea.New()
	content.Placeholder = "Snippet content..."
	content.ShowLineNumbers = true

	innerWidth := width - 6
	if innerWidth < 16 {
		innerWidth = 16
	}
	title.Width = innerWidth
	lang.Width = innerWidth
	tags.Width = innerWidth
	content.SetWidth(innerWidth)
	caHeight := height - 12
	if caHeight < 3 {
		caHeight = 3
	}
	content.SetHeight(caHeight)

	if kind == formEdit {
		title.SetValue(sn.Title)
		lang.SetValue(sn.Language)
		tags.SetValue(sn.TagString())
		content.SetValue(sn.Content)
	}

	return formModel{
		kind:     kind,
		id:       sn.ID,
		created:  kind == formEdit,
		title:    title,
		lang:     lang,
		tags:     tags,
		content:  content,
		focusIdx: fieldTitle,
		width:    width,
		height:   height,
		orig:     sn,
	}
}

func (f formModel) focusFirst() tea.Cmd {
	return f.focusField(fieldTitle)
}

func (f *formModel) focusField(idx int) tea.Cmd {
	f.blurAll()
	f.focusIdx = idx
	switch idx {
	case fieldTitle:
		return f.title.Focus()
	case fieldLang:
		return f.lang.Focus()
	case fieldTags:
		return f.tags.Focus()
	case fieldContent:
		return f.content.Focus()
	}
	return nil
}

func (f *formModel) blurAll() {
	f.title.Blur()
	f.lang.Blur()
	f.tags.Blur()
	f.content.Blur()
}

func (f formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "tab", "down":
			// In the content textarea, "down" should move the cursor, not fields.
			if f.focusIdx == fieldContent && km.String() == "down" {
				break
			}
			next := (f.focusIdx + 1) % fieldCount
			return f, f.focusField(next)
		case "shift+tab", "up":
			if f.focusIdx == fieldContent && km.String() == "up" {
				break
			}
			prev := (f.focusIdx - 1 + fieldCount) % fieldCount
			return f, f.focusField(prev)
		}
	}

	var cmd tea.Cmd
	switch f.focusIdx {
	case fieldTitle:
		f.title, cmd = f.title.Update(msg)
	case fieldLang:
		f.lang, cmd = f.lang.Update(msg)
	case fieldTags:
		f.tags, cmd = f.tags.Update(msg)
	case fieldContent:
		f.content, cmd = f.content.Update(msg)
	}
	return f, cmd
}

func (f formModel) snippet() (snippet.Snippet, error) {
	title := strings.TrimSpace(f.title.Value())
	if title == "" {
		return snippet.Snippet{}, errors.New("title is required")
	}
	body := f.content.Value()
	if strings.TrimSpace(body) == "" {
		return snippet.Snippet{}, errors.New("content is required")
	}
	if f.kind == formAdd {
		return snippet.New(title, f.lang.Value(), body, snippet.ParseTags(f.tags.Value())), nil
	}
	// Edit: preserve ID and created_at; store.Update refreshes updated_at.
	sn := f.orig
	sn.Title = title
	sn.Language = strings.TrimSpace(f.lang.Value())
	sn.Tags = snippet.ParseTags(f.tags.Value())
	sn.Content = body
	return sn, nil
}

func (f formModel) View() string {
	heading := "New snippet"
	if f.kind == formEdit {
		heading = "Edit snippet"
	}

	field := func(label string, m textinput.Model) string {
		return lipgloss.JoinVertical(lipgloss.Left,
			formLabelStyle.Render(label),
			m.View(),
		)
	}

	rows := []string{
		formTitleStyle.Render(heading),
		"",
		field("Title", f.title),
		"",
		field("Language", f.lang),
		"",
		field("Tags", f.tags),
		"",
		formLabelStyle.Render("Content"),
		f.content.View(),
		"",
		helpStyle.Render("tab/↑↓ next field • ctrl+s save • esc cancel"),
	}
	return formBox.Width(f.width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
