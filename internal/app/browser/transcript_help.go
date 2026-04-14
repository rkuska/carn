package browser

func transcriptToggleItems(opts transcriptOptions, content contentFlags) []helpItem {
	return []helpItem{
		{
			Key:    "t",
			Desc:   "thinking",
			Detail: "show or hide assistant thinking blocks",
			Toggle: true,
			On:     opts.showThinking,
			Glow:   !opts.showThinking && content.hasThinking,
		},
		{
			Key:    "T",
			Desc:   "tools",
			Detail: "show or hide tool call summaries",
			Toggle: true,
			On:     opts.showTools,
			Glow:   !opts.showTools && content.hasToolCalls,
		},
		{
			Key:    "R",
			Desc:   "results",
			Detail: "show or hide tool result output",
			Toggle: true,
			On:     opts.showToolResults,
			Glow:   !opts.showToolResults && content.hasToolResults,
		},
		{
			Key:    "s",
			Desc:   "sidechain",
			Detail: "show or hide sidechain messages",
			Toggle: true,
			On:     !opts.hideSidechain,
			Glow:   opts.hideSidechain && content.hasSidechain,
		},
		{
			Key:    "m",
			Desc:   "system",
			Detail: "show or hide system messages",
			Toggle: true,
			On:     opts.showSystem,
			Glow:   !opts.showSystem && content.hasSystem,
		},
	}
}

func transcriptFooterItems(opts transcriptOptions, content contentFlags) []helpItem {
	items := []helpItem{
		{Key: "/", Desc: "search", Detail: "search inside the visible transcript"},
		{Key: "n/N", Desc: "match", Detail: "jump to the next or previous search match"},
	}
	items = append(items, transcriptToggleItems(opts, content)...)
	items = append(items,
		helpItem{Key: "y", Desc: "copy", Detail: "choose what to copy from this conversation"},
		helpItem{Key: "e", Desc: "export", Detail: "export the conversation as Markdown", Priority: helpPriorityHigh},
		helpItem{Key: "?", Desc: "help", Priority: helpPriorityEssential},
		helpItem{Key: "q/esc", Desc: "back", Priority: helpPriorityHigh},
	)
	return items
}

func transcriptHelpSections(
	opts transcriptOptions,
	content contentFlags,
	extraActions []helpItem,
) []helpSection {
	actions := []helpItem{
		{Key: "/", Desc: "search", Detail: "search inside the visible transcript"},
		{Key: "n / N", Desc: "match", Detail: "jump to the next or previous search match"},
		{Key: "y", Desc: "copy", Detail: "copy the conversation, a plan, or the raw source"},
		{Key: "o", Desc: "open", Detail: "open the conversation, a plan, or the raw source in $EDITOR"},
		{Key: "e", Desc: "export", Detail: "export the conversation as Markdown"},
		{Key: "r", Desc: "resume", Detail: "resume the session with its original provider"},
		{Key: "q/esc", Desc: "back", Detail: "close the conversation"},
	}
	actions = append(actions, extraActions...)

	return []helpSection{
		{
			Title: "Navigation",
			Items: []helpItem{
				{Key: "j/k", Desc: "scroll", Detail: "scroll the transcript up or down"},
				{Key: "gg", Desc: "top", Detail: "jump to the beginning of the transcript"},
				{Key: "G", Desc: "bottom", Detail: "jump to the end of the transcript"},
				{Key: "ctrl+f/b", Desc: "page", Detail: "move a page down or up"},
			},
		},
		{
			Title: "Actions",
			Items: actions,
		},
		{
			Title: "Toggles",
			Items: transcriptToggleItems(opts, content),
		},
	}
}
