package browser

import glamouransi "github.com/charmbracelet/glamour/ansi"

const subduedMarkdownMargin = 2

func subduedMarkdownStyleConfig(hasDarkBG bool) glamouransi.StyleConfig {
	bodyColor := markdownThemeValue(hasDarkBG, "240", "249")
	headingColor := markdownThemeValue(hasDarkBG, "232", "255")
	mutedColor := markdownThemeValue(hasDarkBG, "245", "243")
	accentColor := markdownThemeValue(hasDarkBG, "28", "114")
	codeBackground := markdownThemeValue(hasDarkBG, "254", "236")
	codeForeground := markdownThemeValue(hasDarkBG, "238", "252")
	listIndent := uint(2)

	return glamouransi.StyleConfig{
		Document: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       markdownStringPtr(bodyColor),
			},
			Margin: markdownUintPtr(subduedMarkdownMargin),
		},
		BlockQuote: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color:  markdownStringPtr(mutedColor),
				Italic: markdownBoolPtr(true),
			},
			Indent:      markdownUintPtr(1),
			IndentToken: markdownStringPtr("│ "),
		},
		Paragraph: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: markdownStringPtr(bodyColor),
			},
		},
		List: glamouransi.StyleList{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color: markdownStringPtr(bodyColor),
				},
			},
			LevelIndent: 2,
		},
		Heading: markdownHeadingBlock("", headingColor),
		H1:      markdownHeadingBlock("# ", headingColor),
		H2:      markdownHeadingBlock("## ", headingColor),
		H3:      markdownHeadingBlock("### ", headingColor),
		H4:      markdownHeadingBlock("#### ", headingColor),
		H5:      markdownHeadingBlock("##### ", headingColor),
		H6:      markdownHeadingBlock("###### ", mutedColor),
		Text: glamouransi.StylePrimitive{
			Color: markdownStringPtr(bodyColor),
		},
		Strikethrough: glamouransi.StylePrimitive{
			CrossedOut: markdownBoolPtr(true),
		},
		Emph: glamouransi.StylePrimitive{
			Italic: markdownBoolPtr(true),
		},
		Strong: glamouransi.StylePrimitive{
			Bold: markdownBoolPtr(true),
		},
		HorizontalRule: glamouransi.StylePrimitive{
			Color:  markdownStringPtr(mutedColor),
			Format: "\n--------\n",
		},
		Item: glamouransi.StylePrimitive{
			BlockPrefix: "• ",
			Color:       markdownStringPtr(bodyColor),
		},
		Enumeration: glamouransi.StylePrimitive{
			BlockPrefix: ". ",
			Color:       markdownStringPtr(bodyColor),
		},
		Task: glamouransi.StyleTask{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: markdownStringPtr(bodyColor),
			},
			Ticked:   "[x] ",
			Unticked: "[ ] ",
		},
		Link: glamouransi.StylePrimitive{
			Color:     markdownStringPtr(accentColor),
			Underline: markdownBoolPtr(true),
		},
		LinkText: glamouransi.StylePrimitive{
			Color: markdownStringPtr(accentColor),
		},
		Image: glamouransi.StylePrimitive{
			Color: markdownStringPtr(accentColor),
		},
		ImageText: glamouransi.StylePrimitive{
			Color:  markdownStringPtr(mutedColor),
			Format: "Image: {{.Text}} ->",
		},
		Code: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           markdownStringPtr(codeForeground),
				BackgroundColor: markdownStringPtr(codeBackground),
			},
		},
		CodeBlock: glamouransi.StyleCodeBlock{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color:           markdownStringPtr(codeForeground),
					BackgroundColor: markdownStringPtr(codeBackground),
				},
				Margin: &listIndent,
			},
		},
		Table: glamouransi.StyleTable{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color: markdownStringPtr(bodyColor),
				},
			},
			CenterSeparator: markdownStringPtr("|"),
			ColumnSeparator: markdownStringPtr("|"),
			RowSeparator:    markdownStringPtr("-"),
		},
		DefinitionTerm: glamouransi.StylePrimitive{
			Color: markdownStringPtr(headingColor),
			Bold:  markdownBoolPtr(true),
		},
		DefinitionDescription: glamouransi.StylePrimitive{
			BlockPrefix: "\n- ",
			Color:       markdownStringPtr(bodyColor),
		},
		HTMLBlock: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: markdownStringPtr(mutedColor),
			},
		},
		HTMLSpan: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: markdownStringPtr(mutedColor),
			},
		},
	}
}

func markdownHeadingBlock(prefix, color string) glamouransi.StyleBlock {
	return glamouransi.StyleBlock{
		StylePrimitive: glamouransi.StylePrimitive{
			Prefix:      prefix,
			Color:       markdownStringPtr(color),
			Bold:        markdownBoolPtr(true),
			BlockSuffix: "\n",
		},
	}
}

func markdownThemeValue(hasDarkBG bool, light, dark string) string {
	if hasDarkBG {
		return dark
	}
	return light
}

func markdownStringPtr(value string) *string {
	return &value
}

func markdownBoolPtr(value bool) *bool {
	return &value
}

func markdownUintPtr(value uint) *uint {
	return &value
}
