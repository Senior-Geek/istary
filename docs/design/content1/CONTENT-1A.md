# CONTENT-1A: stories in Markdown files

**Baseline:** v0.14.1-castle-startup (`911cb82`). **Branch:**
`content-1a-markdown-stories`.

A place's story, the narrative in its story box (ART-2B), is a Markdown
file in the repository, built into Istary. The files are the stories' only
text: there is none in Svelte or Go. Changing a story means editing its
file and building Istary again; Istary reads no story from anywhere else.
No rule, navigation, startup, save or setting changes.

## The files

```text
gui/content/
├── castle/
│   ├── introduction.md   the Castle with no game open (NAV-1A)
│   └── story.md          the Castle in a game
└── edge-of-town/
    └── story.md          the Edge of Town, with or without a game
```

Each place with stories has a folder. `story.md` is its story;
`introduction.md`, if it has one, is its story while no game is open.

## How a story is chosen and shown

- **Go** (`gui/story.go`) embeds `gui/content` (`go:embed`) and parses
  every file once, when Istary starts, into blocks of plain text
  (`Story`, `StoryBlock`, `StoryRun`).
- **The View** carries the story for where the player stands
  (`View.Story`, `storyFor`): `storyFolders` maps a location to its
  folder (`castle`, `edge` → `edge-of-town`); with no game open the
  folder's introduction if it has one, otherwise its story. Only the
  town's places have one (`Screen` "roster"): none in game selection,
  creation, the maze or combat, or where there is no file.
- **The frontend:** `LocationFrame` opens the story box where the View has
  a story and the place has its picture; `StoryText.svelte` draws it with
  ordinary elements. Every run is text, which Svelte escapes: no
  `{@html}`, and nothing in a file is ever HTML or script. The box's
  title is the story's first `#` heading, or the place's name.
- The story box itself (moving, resizing, scrolling, ×, the Story toggle,
  the appearances) is unchanged.

## Supported Markdown

| Write | Shown as |
|---|---|
| `# Title` as the first line | the story box's title |
| `#`, `##`, `###` elsewhere | a heading in the story |
| blank-line-separated text | paragraphs; a paragraph's lines are joined |
| `**bold**`, `*italic*` or `_italic_` | bold, italic (`**_both_**`) |
| `- item`, `* item`, `1. item` | a list; lines under an item continue it |
| `> text` | a quote; `>` alone separates its paragraphs |
| `> [!challenge]` then `> text` | a challenge: bold, in the story box's gold |
| `\*` (backslash, punctuation) | the character itself |

Anything else, HTML, links and images included, is shown as the text it
is. An ordinary quote is never a challenge: only `[!challenge]` makes one.

## Changing or adding a story

- **Change one:** edit its file, then build (`wails build`) or run
  (`wails dev`) Istary. The fast tests check the files load, and
  `TestTheApprovedIntroductionIsKept` checks the Castle's introduction
  against the approved text in `docs/design/nav1/NAV-1A.md`: change that
  record too if the introduction is meant to change.
- **Add a place's story:** create `gui/content/<place>/story.md`, add the
  location and folder to `storyFolders` (`gui/story.go`), and update
  `TestTheStoryFilesLoad`'s list of files and
  `TestTheStoryForEachPlace`. The place shows it once it has a picture
  (`scenes.js`); no Svelte changes.
- **ART-3:** a location is its illustration, its Markdown story and its
  existing interface. Neither the picture nor the story changes a rule.

## Tests

- **Go** (`gui/story_test.go`): the three files load; the introduction is
  the approved text, its title, its ten paragraphs word for word with the
  bold Edge of Town line, and its challenge; the Markdown forms, and
  emphasis markers that are not emphasis; HTML is text; the story for
  each place, with and without a game. The frame fixture
  (`frame-views.json`) records each view's story.
- **Frontend** (`content1a.test.js`): `StoryText` rendered with Svelte's
  own server renderer, for every kind of block, for HTML and script
  (escaped, never elements), and for the three stories as Go recorded
  them; the challenge's style; no `{@html}` anywhere; the Castle and the
  Edge hold no story of their own. The NAV-1A and layout tests read the
  stories from the fixture, not from the Svelte source.
- **Live:** the scenes and startup suites compare the story box's text
  with the View's story, not with quoted text.
- **The test plan** (`scripts/lib/classify.mjs`): a story file is the
  GUI's embedded data, not documentation, so changing one runs the GUI's
  tests, the frontend's and the town's live checks.
