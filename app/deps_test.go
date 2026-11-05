package app

import (
	"go/build"
	"strings"
	"testing"
)

// The controller, rules and data packages must stay usable from a non-terminal
// front end, so none of them may import terminal code.
func TestNoTerminalDependencies(t *testing.T) {
	banned := []string{"github.com/gdamore/tcell", "golang.org/x/term", "github.com/Senior-Geek/istary/render"}
	for _, dir := range []string{".", "../engine", "../data"} {
		pkg, err := build.ImportDir(dir, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range pkg.Imports {
			for _, b := range banned {
				if strings.HasPrefix(imp, b) {
					t.Errorf("package %s imports %s", pkg.Name, imp)
				}
			}
		}
	}
}
