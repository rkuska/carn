package main

import "charm.land/bubbles/v2/key"

type browserKeyMap struct {
	Enter      key.Binding
	Tab        key.Binding
	DeepSearch key.Binding
	Resume     key.Binding
	Copy       key.Binding
	Export     key.Binding
	Editor     key.Binding
	Help       key.Binding
	Quit       key.Binding
}

var browserKeys = browserKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open transcript"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle focus"),
	),
	DeepSearch: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "deep search"),
	),
	Resume: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "resume session"),
	),
	Copy: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy transcript"),
	),
	Export: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "export markdown"),
	),
	Editor: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open in editor"),
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
	ToggleSidechain   key.Binding
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
		key.WithHelp("t", "toggle thinking"),
	),
	ToggleTools: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "toggle tools"),
	),
	ToggleToolResults: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "toggle tool results"),
	),
	ToggleSidechain: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle sidechain"),
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
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "resume session"),
	),
	Copy: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy transcript"),
	),
	Export: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "export"),
	),
	Editor: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open in editor"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "back"),
	),
}
