package app

import "charm.land/bubbles/v2/key"

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
