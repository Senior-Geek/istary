package main

// Stories (CONTENT-1A; docs/design/content1/CONTENT-1A.md): a place's
// narrative, shown in its story box (ART-2B), is a Markdown file in
// content/<place>/, built into the application. story.md is the place's
// story; introduction.md, if the place has one, is its story while no game
// is open (NAV-1A). The files are the stories' only text: Go parses them
// here into blocks of plain text, the View carries the one for where the
// player stands, and the frontend draws it with ordinary elements
// (StoryText.svelte), so nothing in a story is ever HTML or script.
//
// The Markdown is the little the stories need: "#", "##" and "###"
// headings (a first "#" heading is the story box's title), paragraphs,
// **bold**, *italic* or _italic_, "-", "*" or "1." lists, and ">"
// quotes. A quote whose first line is "[!challenge]" is a challenge,
// drawn in the story box's gold. A backslash keeps the punctuation
// after it as it is; anything else, HTML included, is text.

import (
	"embed"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

//go:embed content
var storyFiles embed.FS

// Story is a story parsed from its file.
type Story struct {
	Title  string       `json:"title,omitempty"` // its first "#" heading, if any
	Blocks []StoryBlock `json:"blocks"`
}

// StoryBlock is one of a story's blocks. Kind is "heading" (Level 1 to 3),
// "paragraph", "list" (Ordered for "1."), "quote" or "challenge". A
// heading's and a paragraph's text is Text; a list's items and a quote's
// or a challenge's paragraphs are Items.
type StoryBlock struct {
	Kind    string       `json:"kind"`
	Level   int          `json:"level,omitempty"`
	Ordered bool         `json:"ordered,omitempty"`
	Text    []StoryRun   `json:"text,omitempty"`
	Items   [][]StoryRun `json:"items,omitempty"`
}

// StoryRun is a run of text and its emphasis.
type StoryRun struct {
	Text   string `json:"text"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
}

// stories are the built-in stories by file, "castle/story" for
// content/castle/story.md.
var stories = loadStories(storyFiles)

// storyFolders are the places with stories, and their folders in content/.
var storyFolders = map[string]string{
	"castle": "castle",
	"edge":   "edge-of-town",
}

func loadStories(files fs.FS) map[string]*Story {
	m := map[string]*Story{}
	// The files are built in: reading them cannot fail but by a broken
	// build, which the tests find (story_test.go).
	_ = fs.WalkDir(files, "content", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Ext(p) != ".md" {
			return err
		}
		raw, err := fs.ReadFile(files, p)
		if err == nil {
			m[strings.TrimSuffix(strings.TrimPrefix(p, "content/"), ".md")] = parseStory(string(raw))
		}
		return err
	})
	return m
}

// storyFor is the story for location: its introduction with no game open,
// if it has one, otherwise its story; nil where there is none.
func storyFor(location string, game bool) *Story {
	folder, ok := storyFolders[location]
	if !ok {
		return nil
	}
	if st := stories[folder+"/introduction"]; !game && st != nil {
		return st
	}
	return stories[folder+"/story"]
}

var (
	headingLine = regexp.MustCompile(`^(#{1,3})\s+(.*?)(\s+#+)?$`)
	bulletLine  = regexp.MustCompile(`^[-*+]\s+(.*)$`)
	numberLine  = regexp.MustCompile(`^\d+[.)]\s+(.*)$`)
	challenge   = regexp.MustCompile(`(?i)^\[!challenge\]$`)
)

// parseStory parses a story's Markdown.
func parseStory(src string) *Story {
	st := &Story{Blocks: []StoryBlock{}}
	var para, quote, items []string
	ordered := false
	add := func(b StoryBlock) { st.Blocks = append(st.Blocks, b) }
	endPara := func() {
		if len(para) > 0 {
			add(StoryBlock{Kind: "paragraph", Text: inline(strings.Join(para, " "))})
			para = nil
		}
	}
	endList := func() {
		if len(items) > 0 {
			b := StoryBlock{Kind: "list", Ordered: ordered}
			for _, it := range items {
				b.Items = append(b.Items, inline(it))
			}
			add(b)
			items = nil
		}
	}
	endQuote := func() {
		if len(quote) == 0 {
			return
		}
		b := StoryBlock{Kind: "quote"}
		if challenge.MatchString(quote[0]) {
			b.Kind, quote = "challenge", quote[1:]
		}
		var p []string
		for _, l := range append(quote, "") {
			if l != "" {
				p = append(p, l)
			} else if len(p) > 0 {
				b.Items = append(b.Items, inline(strings.Join(p, " ")))
				p = nil
			}
		}
		if len(b.Items) > 0 {
			add(b)
		}
		quote = nil
	}

	for _, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(t, ">"); ok {
			endPara()
			endList()
			quote = append(quote, strings.TrimSpace(rest))
			continue
		}
		endQuote()
		if t == "" {
			endPara()
			endList()
			continue
		}
		if m := headingLine.FindStringSubmatch(t); m != nil {
			endPara()
			endList()
			if text := inline(m[2]); len(m[1]) == 1 && st.Title == "" && len(st.Blocks) == 0 {
				st.Title = plain(text)
			} else {
				add(StoryBlock{Kind: "heading", Level: len(m[1]), Text: text})
			}
			continue
		}
		m, num := bulletLine.FindStringSubmatch(t), false
		if m == nil {
			m, num = numberLine.FindStringSubmatch(t), true
		}
		switch {
		case m != nil:
			endPara()
			if len(items) > 0 && ordered != num {
				endList()
			}
			ordered = num
			items = append(items, m[1])
		case len(items) > 0:
			// A line under an item, indented or not, continues it.
			items[len(items)-1] += " " + t
		default:
			para = append(para, t)
		}
	}
	endPara()
	endList()
	endQuote()
	return st
}

// plain is runs' text without their emphasis.
func plain(runs []StoryRun) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// inline parses a block's text: **bold**, *italic* or _italic_, and a
// backslash before punctuation, which keeps it as it is. A marker that is
// not closed is text.
func inline(s string) []StoryRun {
	var runs []StoryRun
	emit := func(text string, bold, italic bool) {
		if text == "" {
			return
		}
		if n := len(runs); n > 0 && runs[n-1].Bold == bold && runs[n-1].Italic == italic {
			runs[n-1].Text += text
			return
		}
		runs = append(runs, StoryRun{Text: text, Bold: bold, Italic: italic})
	}
	var walk func(s string, bold, italic bool)
	walk = func(s string, bold, italic bool) {
		var b strings.Builder
		for i := 0; i < len(s); {
			c := s[i]
			if c == '\\' && i+1 < len(s) && isPunct(s[i+1]) {
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
			if c == '*' || c == '_' {
				marker, inBold, inItalic := string(c), bold, true
				if c == '*' && strings.HasPrefix(s[i:], "**") {
					marker, inBold, inItalic = "**", true, italic
				}
				if j := closer(s, i, marker); j > 0 && !(marker == "**" && bold) && !(marker != "**" && italic) {
					emit(b.String(), bold, italic)
					b.Reset()
					walk(s[i+len(marker):j], inBold, inItalic)
					i = j + len(marker)
					continue
				}
				if marker == "**" {
					b.WriteString(marker)
					i += 2
					continue
				}
			}
			b.WriteByte(c)
			i++
		}
		emit(b.String(), bold, italic)
	}
	walk(s, false, false)
	return runs
}

// closer is where the emphasis marker opened at s[i] closes, or -1: it
// opens before a letter, not a space, and closes after one; an underscore
// only between words, not inside one.
func closer(s string, i int, marker string) int {
	from := i + len(marker)
	if from >= len(s) || s[from] == ' ' || (marker == "_" && i > 0 && isWord(s[i-1])) {
		return -1
	}
	for k := from; k < len(s); k++ {
		switch {
		case s[k] == '\\':
			k++
		case marker == "*" && strings.HasPrefix(s[k:], "**"):
			k++ // bold inside italic
		case strings.HasPrefix(s[k:], marker) && k > from && s[k-1] != ' ':
			if marker == "_" && k+1 < len(s) && isWord(s[k+1]) {
				continue
			}
			return k
		}
	}
	return -1
}

func isPunct(c byte) bool { return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0 }

func isWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
