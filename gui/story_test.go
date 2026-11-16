package main

// CONTENT-1A: the stories are Markdown files built into the application
// (content/), parsed into plain text and chosen for where the player
// stands.

import (
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestTheStoryFilesLoad(t *testing.T) {
	var files []string
	if err := fs.WalkDir(storyFiles, "content", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, p)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{"content/boltac/story.md", "content/castle/introduction.md", "content/castle/story.md", "content/edge-of-town/story.md", "content/inn/story.md", "content/tavern/story.md"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("the built-in story files: %v, not %v", files, want)
	}
	for _, name := range []string{"boltac/story", "castle/introduction", "castle/story", "edge-of-town/story", "inn/story", "tavern/story"} {
		st := stories[name]
		if st == nil || len(st.Blocks) == 0 {
			t.Fatalf("%s: no story", name)
		}
		for _, b := range st.Blocks {
			if len(b.Text) == 0 && len(b.Items) == 0 {
				t.Fatalf("%s: an empty %s", name, b.Kind)
			}
		}
	}
	// Each place's folder is one with a story.
	for place, folder := range storyFolders {
		if stories[folder+"/story"] == nil {
			t.Fatalf("%s: no %s/story.md", place, folder)
		}
	}
}

// approvedIntroduction is the introductory story's approved text, as NAV-1A
// records it: its paragraphs, the first its title.
func approvedIntroduction(t *testing.T) []string {
	t.Helper()
	return approvedText(t, "../docs/design/nav1/NAV-1A.md")
}

// approvedText is a story's approved text, quoted after "The approved
// text" in its design document: its paragraphs, the first its title and,
// in the stories that have one, the last its challenge.
func approvedText(t *testing.T, file string) []string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	i := strings.Index(doc, "The approved text")
	if i < 0 {
		t.Fatalf("%s: no approved text", file)
	}
	var paras, p []string
	for _, line := range strings.Split(doc[i:], "\n")[1:] {
		line = strings.TrimSpace(line)
		if len(paras) > 0 && !strings.HasPrefix(line, ">") && line != "" {
			break
		}
		if text, ok := strings.CutPrefix(line, ">"); ok && strings.TrimSpace(text) != "" {
			p = append(p, strings.TrimSpace(text))
		} else if len(p) > 0 {
			paras, p = append(paras, strings.Join(p, " ")), nil
		}
	}
	if len(p) > 0 {
		paras = append(paras, strings.Join(p, " "))
	}
	return paras
}

func TestTheApprovedIntroductionIsKept(t *testing.T) {
	approved := approvedIntroduction(t)
	if len(approved) != 12 {
		t.Fatalf("NAV-1A's approved text: %d paragraphs, not a title and 11", len(approved))
	}
	st := stories["castle/introduction"]
	if want := plain(inline(approved[0])); st.Title != want {
		t.Fatalf("title %q, not %q", st.Title, want)
	}
	if len(st.Blocks) != 11 {
		t.Fatalf("%d blocks, not 11", len(st.Blocks))
	}
	// The ten paragraphs word for word, with their emphasis ("Your journey
	// begins at the Edge of Town." bold).
	for i, b := range st.Blocks[:10] {
		if want := inline(approved[i+1]); b.Kind != "paragraph" || !reflect.DeepEqual(b.Text, want) {
			t.Fatalf("paragraph %d: %s %+v, not %+v", i+1, b.Kind, b.Text, want)
		}
	}
	if b := st.Blocks[6]; len(b.Text) != 1 || !b.Text[0].Bold {
		t.Fatalf("the Edge of Town's line is not bold: %+v", b.Text)
	}
	// The challenge, emphasised as a challenge, not an ordinary quote.
	last := st.Blocks[10]
	if want := plain(inline(approved[11])); last.Kind != "challenge" || len(last.Items) != 1 || plain(last.Items[0]) != want {
		t.Fatalf("the closing challenge: %s %+v, not %q", last.Kind, last.Items, want)
	}
}

// ART-3B-1: the Tavern's story is its approved text, word for word: its
// title, eight paragraphs ("Gilgamesh" bold where he is named) and its
// challenge.
func TestTheApprovedTavernStoryIsKept(t *testing.T) {
	approved := approvedText(t, "../docs/design/art3/tavern/README.md")
	if len(approved) != 10 {
		t.Fatalf("ART-3B-1's approved text: %d paragraphs, not a title, 8 and a challenge", len(approved))
	}
	st := storyFor("tavern", true)
	if st != stories["tavern/story"] || storyFor("tavern", false) != st {
		t.Fatal("the Tavern's story is not its own, with or without a game")
	}
	if want := plain(inline(approved[0])); st.Title != want {
		t.Fatalf("title %q, not %q", st.Title, want)
	}
	if len(st.Blocks) != 9 {
		t.Fatalf("%d blocks, not 9", len(st.Blocks))
	}
	for i, b := range st.Blocks[:8] {
		if want := inline(approved[i+1]); b.Kind != "paragraph" || !reflect.DeepEqual(b.Text, want) {
			t.Fatalf("paragraph %d: %s %+v, not %+v", i+1, b.Kind, b.Text, want)
		}
	}
	if b := st.Blocks[3]; len(b.Text) != 3 || !b.Text[1].Bold || b.Text[1].Text != "Gilgamesh" {
		t.Fatalf("Gilgamesh is not named in bold: %+v", b.Text)
	}
	last := st.Blocks[8]
	if want := plain(inline(approved[9])); last.Kind != "challenge" || len(last.Items) != 1 || plain(last.Items[0]) != want {
		t.Fatalf("the closing challenge: %s %+v, not %q", last.Kind, last.Items, want)
	}
}

// ART-3B-2B: the Inn's story is its approved text, word for word: its
// title, eleven paragraphs ("Mistress Elspeth" bold where she is named) and
// its challenge.
func TestTheApprovedInnStoryIsKept(t *testing.T) {
	approved := approvedText(t, "../docs/design/art3/inn/README.md")
	if len(approved) != 13 {
		t.Fatalf("ART-3B-2B's approved text: %d paragraphs, not a title, 11 and a challenge", len(approved))
	}
	st := storyFor("inn", true)
	if st != stories["inn/story"] || storyFor("inn", false) != st {
		t.Fatal("the Inn's story is not its own, with or without a game")
	}
	if want := plain(inline(approved[0])); st.Title != want {
		t.Fatalf("title %q, not %q", st.Title, want)
	}
	if len(st.Blocks) != 12 {
		t.Fatalf("%d blocks, not 12", len(st.Blocks))
	}
	for i, b := range st.Blocks[:11] {
		if want := inline(approved[i+1]); b.Kind != "paragraph" || !reflect.DeepEqual(b.Text, want) {
			t.Fatalf("paragraph %d: %s %+v, not %+v", i+1, b.Kind, b.Text, want)
		}
	}
	if b := st.Blocks[2]; len(b.Text) != 3 || !b.Text[1].Bold || b.Text[1].Text != "Mistress Elspeth" {
		t.Fatalf("Mistress Elspeth is not named in bold: %+v", b.Text)
	}
	last := st.Blocks[11]
	if want := plain(inline(approved[12])); last.Kind != "challenge" || len(last.Items) != 1 || plain(last.Items[0]) != want {
		t.Fatalf("the closing challenge: %s %+v, not %q", last.Kind, last.Items, want)
	}
}

// ART-3B-3B: Boltac's story is its approved text, word for word: its
// title, twelve paragraphs ("Boltac" bold where he is first named) and its
// challenge. It is the Trading Post's, in content/boltac/.
func TestTheApprovedBoltacStoryIsKept(t *testing.T) {
	approved := approvedText(t, "../docs/design/art3/boltac/README.md")
	if len(approved) != 14 {
		t.Fatalf("ART-3B-3B's approved text: %d paragraphs, not a title, 12 and a challenge", len(approved))
	}
	st := storyFor("trading", true)
	if st != stories["boltac/story"] || storyFor("trading", false) != st {
		t.Fatal("the Trading Post's story is not Boltac's, with or without a game")
	}
	if want := plain(inline(approved[0])); st.Title != want {
		t.Fatalf("title %q, not %q", st.Title, want)
	}
	if len(st.Blocks) != 13 {
		t.Fatalf("%d blocks, not 13", len(st.Blocks))
	}
	for i, b := range st.Blocks[:12] {
		if want := inline(approved[i+1]); b.Kind != "paragraph" || !reflect.DeepEqual(b.Text, want) {
			t.Fatalf("paragraph %d: %s %+v, not %+v", i+1, b.Kind, b.Text, want)
		}
	}
	if b := st.Blocks[2]; len(b.Text) != 3 || !b.Text[1].Bold || b.Text[1].Text != "Boltac" {
		t.Fatalf("Boltac is not named in bold: %+v", b.Text)
	}
	last := st.Blocks[12]
	if want := plain(inline(approved[13])); last.Kind != "challenge" || len(last.Items) != 1 || plain(last.Items[0]) != want {
		t.Fatalf("the closing challenge: %s %+v, not %q", last.Kind, last.Items, want)
	}
}

func TestStoryMarkdown(t *testing.T) {
	r := func(text string) StoryRun { return StoryRun{Text: text} }
	b := func(text string) StoryRun { return StoryRun{Text: text, Bold: true} }
	i := func(text string) StoryRun { return StoryRun{Text: text, Italic: true} }
	para := func(runs ...StoryRun) StoryBlock { return StoryBlock{Kind: "paragraph", Text: runs} }
	for _, c := range []struct {
		name, src string
		want      Story
	}{
		{"paragraphs, their lines joined", "One\ntwo.\n\n\nThree.\r\n", Story{Blocks: []StoryBlock{para(r("One two.")), para(r("Three."))}}},
		{"a first # heading is the title", "# The *Title*\n\nText.", Story{Title: "The Title", Blocks: []StoryBlock{para(r("Text."))}}},
		{"other headings", "Text.\n# One\n## Two ##\n### Three\n#### Four", Story{Blocks: []StoryBlock{
			para(r("Text.")),
			{Kind: "heading", Level: 1, Text: []StoryRun{r("One")}},
			{Kind: "heading", Level: 2, Text: []StoryRun{r("Two")}},
			{Kind: "heading", Level: 3, Text: []StoryRun{r("Three")}},
			para(r("#### Four")),
		}}},
		{"bold and italic", "A **bold** and *italic* and _also_ and ***", Story{Blocks: []StoryBlock{
			para(r("A "), b("bold"), r(" and "), i("italic"), r(" and "), i("also"), r(" and ***")),
		}}},
		{"bold in italic, italic in bold", "*a **b** c* **d _e_**", Story{Blocks: []StoryBlock{
			para(i("a "), StoryRun{Text: "b", Bold: true, Italic: true}, i(" c"), r(" "), b("d "), StoryRun{Text: "e", Bold: true, Italic: true}),
		}}},
		{"markers that are not emphasis", "2 * 3 * 4, snake_case_name, ** x**, *unclosed, \\*kept\\*", Story{Blocks: []StoryBlock{
			para(r("2 * 3 * 4, snake_case_name, ** x**, *unclosed, *kept*")),
		}}},
		{"lists", "- one\n* two\n  more\n\n1. first\n2) second", Story{Blocks: []StoryBlock{
			{Kind: "list", Items: [][]StoryRun{{r("one")}, {r("two more")}}},
			{Kind: "list", Ordered: true, Items: [][]StoryRun{{r("first")}, {r("second")}}},
		}}},
		{"quotes, and a challenge", "> A\n> quote.\n>\n> Two.\n\n> [!challenge]\n> **Dare.**\n\n> [!note]\n> Text.", Story{Blocks: []StoryBlock{
			{Kind: "quote", Items: [][]StoryRun{{r("A quote.")}, {r("Two.")}}},
			{Kind: "challenge", Items: [][]StoryRun{{b("Dare.")}}},
			{Kind: "quote", Items: [][]StoryRun{{r("[!note] Text.")}}},
		}}},
		{"nothing", "\n\n", Story{Blocks: []StoryBlock{}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := parseStory(c.src); !reflect.DeepEqual(*got, c.want) {
				t.Fatalf("%q:\n got %+v\nwant %+v", c.src, *got, c.want)
			}
		})
	}
}

// HTML and script in a story are its text, which the frontend draws as
// text (StoryText.svelte).
func TestAStorysHTMLIsText(t *testing.T) {
	src := "# <img src=x onerror=alert(1)>\n\n<script>alert(1)</script> <b>bold</b> [link](javascript:alert(1))\n\n- <i>item</i>"
	st := parseStory(src)
	if st.Title != "<img src=x onerror=alert(1)>" {
		t.Fatalf("title %q", st.Title)
	}
	want := []StoryBlock{
		{Kind: "paragraph", Text: []StoryRun{{Text: "<script>alert(1)</script> <b>bold</b> [link](javascript:alert(1))"}}},
		{Kind: "list", Items: [][]StoryRun{{{Text: "<i>item</i>"}}}},
	}
	if !reflect.DeepEqual(st.Blocks, want) {
		t.Fatalf("%+v", st.Blocks)
	}
}

func TestLoadingStories(t *testing.T) {
	got := loadStories(fstest.MapFS{
		"content/castle/story.md": {Data: []byte("Castle.")},
		"content/inn/story.md":    {Data: []byte("Inn.")},
		"content/inn/notes.txt":   {Data: []byte("not a story")},
	})
	if len(got) != 2 || plain(got["castle/story"].Blocks[0].Text) != "Castle." || plain(got["inn/story"].Blocks[0].Text) != "Inn." {
		t.Fatalf("%+v", got)
	}
}

// The story for where the player stands: the Castle's introduction with no
// game open, its own story in a game, the Edge's in both, and none where
// there is no story or no place.
func TestTheStoryForEachPlace(t *testing.T) {
	s, _ := newLibrary(t)
	defer s.shutdown()
	is := func(v View, name string) {
		t.Helper()
		if want := stories[name]; name == "" && v.Story != nil || name != "" && v.Story != want {
			t.Fatalf("at %q (screen %q, game %d): story %+v, not %s", v.Location, v.Screen, v.Game, v.Story, name)
		}
	}
	is(s.State(), "castle/introduction")
	is(fine(t, s.Navigate("edge")), "edge-of-town/story")
	is(fine(t, s.Navigate("settings")), "")
	is(fine(t, s.ReturnToCastle()), "castle/introduction")
	is(selectGame(t, s), "")

	v := fine(t, s.NewGame(1))
	is(v, "castle/story")
	is(fine(t, s.Navigate("edge")), "edge-of-town/story")
	is(fine(t, s.Navigate("training")), "")
	is(fine(t, s.ReturnToCastle()), "castle/story")
	is(fine(t, s.Navigate("tavern")), "tavern/story") // ART-3B-1: its own
	is(fine(t, s.ReturnToCastle()), "castle/story")
}
