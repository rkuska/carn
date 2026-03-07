package app

func transcriptToggleItems(opts transcriptOptions, content contentFlags) []helpItem {
	return []helpItem{
		{
			key:    "t",
			desc:   "thinking",
			toggle: true,
			on:     opts.showThinking,
			glow:   !opts.showThinking && content.hasThinking,
		},
		{
			key:    "T",
			desc:   "tools",
			toggle: true,
			on:     opts.showTools,
			glow:   !opts.showTools && content.hasToolCalls,
		},
		{
			key:    "R",
			desc:   "results",
			toggle: true,
			on:     opts.showToolResults,
			glow:   !opts.showToolResults && content.hasToolResults,
		},
		{
			key:    "s",
			desc:   "sidechain",
			toggle: true,
			on:     !opts.hideSidechain,
			glow:   opts.hideSidechain && content.hasSidechain,
		},
	}
}

func transcriptFooterItems(opts transcriptOptions, content contentFlags) []helpItem {
	items := []helpItem{
		{key: "?", desc: "help"},
		{key: "o", desc: "editor"},
		{key: "/", desc: "search"},
		{key: "n/N", desc: "match"},
	}
	items = append(items, transcriptToggleItems(opts, content)...)
	items = append(items,
		helpItem{key: "y", desc: "copy"},
		helpItem{key: "q/esc", desc: "back"},
	)
	return items
}

func transcriptHelpSections(
	opts transcriptOptions,
	content contentFlags,
	extraActions []helpItem,
) []helpSection {
	actions := []helpItem{
		{key: "/", desc: "search transcript"},
		{key: "n / N", desc: "next / previous match"},
		{key: "y", desc: "copy transcript"},
		{key: "e", desc: "export markdown"},
		{key: "o", desc: "open in editor"},
		{key: "r", desc: "resume session"},
		{key: "q/esc", desc: "close transcript"},
	}
	actions = append(actions, extraActions...)

	return []helpSection{
		{
			title: "Navigation",
			items: []helpItem{
				{key: "j/k", desc: "scroll"},
				{key: "gg", desc: "go to top"},
				{key: "G", desc: "go to bottom"},
				{key: "ctrl+f/b", desc: "page down/up"},
			},
		},
		{
			title: "Actions",
			items: actions,
		},
		{
			title: "Toggles",
			items: transcriptToggleItems(opts, content),
		},
	}
}
