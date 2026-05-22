// Command snippetbox is a fast terminal snippet manager with fuzzy search,
// syntax-highlighted preview, and one-key copy to the clipboard.
package main

import (
	"os"

	"github.com/ATOM00blue/snippetbox/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
