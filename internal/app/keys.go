package app

import "charm.land/bubbles/v2/key"

type browserKeyMap struct {
	Enter            key.Binding
	Search           key.Binding
	ToggleFullscreen key.Binding
	FocusPane        key.Binding
	DeepSearch       key.Binding
	Resume           key.Binding
	Resync           key.Binding
	Editor           key.Binding
	Help             key.Binding
	Close            key.Binding
	Quit             key.Binding
}

var browserKeys = browserKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open transcript"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	ToggleFullscreen: key.NewBinding(
		key.WithKeys("O"),
		key.WithHelp("O", "toggle fullscreen"),
	),
	FocusPane: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle focus"),
	),
	DeepSearch: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "deep search"),
	),
	Resume: key.NewBinding(
		key.WithKeys("r", "ctrl+r"),
		key.WithHelp("r", "resume session"),
	),
	Resync: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "resync"),
	),
	Editor: key.NewBinding(
		key.WithKeys("o", "ctrl+o"),
		key.WithHelp("o", "open in editor"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Close: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("q/esc", "close transcript"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type importOverviewKeyMap struct {
	Enter     key.Binding
	Configure key.Binding
	Help      key.Binding
	Quit      key.Binding
}

var importOverviewKeys = importOverviewKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	),
	Configure: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "configure"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type viewerKeyMap struct {
	ToggleThinking    key.Binding
	ToggleTools       key.Binding
	ToggleToolResults key.Binding
	TogglePlan        key.Binding
	ToggleSidechain   key.Binding
	ToggleSystem      key.Binding
	Search            key.Binding
	NextMatch         key.Binding
	PrevMatch         key.Binding
	Resume            key.Binding
	Copy              key.Binding
	Export            key.Binding
	Editor            key.Binding
	Back              key.Binding
}

var viewerKeys = viewerKeyMap{
	ToggleThinking: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "thinking"),
	),
	ToggleTools: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "tools"),
	),
	ToggleToolResults: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "results"),
	),
	TogglePlan: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "plan"),
	),
	ToggleSidechain: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sidechain"),
	),
	ToggleSystem: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "system"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	NextMatch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	),
	PrevMatch: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "prev match"),
	),
	Resume: key.NewBinding(
		key.WithKeys("r", "ctrl+r"),
		key.WithHelp("r", "resume session"),
	),
	Copy: key.NewBinding(
		key.WithKeys("y", "ctrl+y"),
		key.WithHelp("y", "copy transcript"),
	),
	Export: key.NewBinding(
		key.WithKeys("e", "ctrl+e"),
		key.WithHelp("e", "export"),
	),
	Editor: key.NewBinding(
		key.WithKeys("o", "ctrl+o"),
		key.WithHelp("o", "open"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("q/esc", "back"),
	),
}
