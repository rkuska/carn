package app

func transcriptToggleItems(opts transcriptOptions, content contentFlags) []helpItem {
	return []helpItem{
		{
			key:    "t",
			desc:   "thinking",
			detail: "show or hide assistant thinking blocks",
			toggle: true,
			on:     opts.showThinking,
			glow:   !opts.showThinking && content.hasThinking,
		},
		{
			key:    "T",
			desc:   "tools",
			detail: "show or hide tool call summaries",
			toggle: true,
			on:     opts.showTools,
			glow:   !opts.showTools && content.hasToolCalls,
		},
		{
			key:    "R",
			desc:   "results",
			detail: "show or hide tool result output",
			toggle: true,
			on:     opts.showToolResults,
			glow:   !opts.showToolResults && content.hasToolResults,
		},
		{
			key:    "s",
			desc:   "sidechain",
			detail: "show or hide sidechain messages",
			toggle: true,
			on:     !opts.hideSidechain,
			glow:   opts.hideSidechain && content.hasSidechain,
		},
		{
			key:    "m",
			desc:   "system",
			detail: "show or hide system messages",
			toggle: true,
			on:     opts.showSystem,
			glow:   !opts.showSystem && content.hasSystem,
		},
	}
}

func transcriptFooterItems(opts transcriptOptions, content contentFlags) []helpItem {
	items := []helpItem{
		{key: "/", desc: "search", detail: "search inside the visible transcript"},
		{key: "n/N", desc: "match", detail: "jump to the next or previous search match"},
	}
	items = append(items, transcriptToggleItems(opts, content)...)
	items = append(items,
		helpItem{key: "y", desc: "copy", detail: "choose what to copy from this conversation"},
		helpItem{key: "e", desc: "export", detail: "export the conversation as Markdown", priority: helpPriorityHigh},
		helpItem{key: "?", desc: "help", priority: helpPriorityEssential},
		helpItem{key: "q/esc", desc: "back", priority: helpPriorityHigh},
	)
	return items
}

func transcriptHelpSections(
	opts transcriptOptions,
	content contentFlags,
	extraActions []helpItem,
) []helpSection {
	actions := []helpItem{
		{key: "/", desc: "search", detail: "search inside the visible transcript"},
		{key: "n / N", desc: "match", detail: "jump to the next or previous search match"},
		{key: "y", desc: "copy", detail: "copy the conversation, a plan, or the raw source"},
		{key: "o", desc: "open", detail: "open the conversation, a plan, or the raw source in $EDITOR"},
		{key: "e", desc: "export", detail: "export the conversation as Markdown"},
		{key: "r", desc: "resume", detail: "resume the session with its original provider"},
		{key: "q/esc", desc: "back", detail: "close the conversation"},
	}
	actions = append(actions, extraActions...)

	return []helpSection{
		{
			title: "Navigation",
			items: []helpItem{
				{key: "j/k", desc: "scroll", detail: "scroll the transcript up or down"},
				{key: "gg", desc: "top", detail: "jump to the beginning of the transcript"},
				{key: "G", desc: "bottom", detail: "jump to the end of the transcript"},
				{key: "ctrl+f/b", desc: "page", detail: "move a page down or up"},
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
