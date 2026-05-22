package tui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// highlight renders source code with ANSI terminal colors using chroma. The
// language hint selects the lexer; if empty or unknown, chroma analyses the
// content to guess. On any failure it returns the original content unchanged so
// the preview always shows something.
func highlight(content, language string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	lexer := lexers.Fallback
	if language != "" {
		if l := lexers.Get(language); l != nil {
			lexer = l
		}
	}
	if lexer == lexers.Fallback {
		if l := lexers.Analyse(content); l != nil {
			lexer = l
		}
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}
	var sb strings.Builder
	if err := formatter.Format(&sb, style, iterator); err != nil {
		return content
	}
	return sb.String()
}
