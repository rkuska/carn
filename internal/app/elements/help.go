package elements

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type HelpItem struct {
	Key      string
	Desc     string
	Detail   string
	Toggle   bool
	On       bool
	Glow     bool
	Priority HelpPriority
}

type HelpSection struct {
	Title string
	Items []HelpItem
}

type HelpPriority int

const (
	HelpPriorityLow HelpPriority = iota
	HelpPriorityNormal
	HelpPriorityHigh
	HelpPriorityEssential
)

func (t *Theme) RenderHelpFooter(width int, items []HelpItem, rightParts []string, n Notification) string {
	contentWidth := FramedFooterContentWidth(width)
	minLeftWidth := t.essentialHelpWidth(items)
	right := JoinNonEmpty(rightParts, "  ")
	maxRightWidth := contentWidth
	if minLeftWidth > 0 && contentWidth > minLeftWidth {
		maxRightWidth = contentWidth - minLeftWidth - 1
	}
	right = TruncateFooterText(right, maxRightWidth)
	leftWidth := contentWidth
	if right != "" {
		leftWidth = max(contentWidth-lipgloss.Width(right)-1, 0)
	}

	return RenderFramedFooter(
		width,
		ComposeFooterRow(width, t.RenderFittedHelpItems(items, leftWidth), right),
		t.RenderNotification(n),
	)
}

func (t *Theme) RenderSearchFooter(width int, prompt, right string, n Notification) string {
	return RenderFramedFooter(
		width,
		ComposeFooterRow(width, prompt, right),
		t.RenderNotification(n),
	)
}

func (t *Theme) RenderHelpItems(items []HelpItem) string {
	return t.renderHelpItemsWithKeep(items, nil)
}

func (t *Theme) RenderFittedHelpItems(items []HelpItem, width int) string {
	if width <= 0 {
		return ""
	}

	keep := t.keepHelpItems(items, width)
	return FitToWidth(t.renderHelpItemsWithKeep(items, keep), width)
}

func (t *Theme) renderHelpItemsWithKeep(items []HelpItem, keep []bool) string {
	helpStyle := lipgloss.NewStyle().Foreground(t.ColorSecondary)

	parts := make([]string, 0, len(items))
	for i, item := range items {
		if keep != nil && !keep[i] {
			continue
		}
		if item.Key == "" || item.Desc == "" {
			continue
		}

		keyText := HelpItemKeyText(item)
		parts = append(parts, t.helpItemKeyStyle(item).Render(keyText)+helpStyle.Render(" "+item.Desc))
	}

	return strings.Join(parts, "  ")
}

func (t *Theme) helpItemKeyStyle(item HelpItem) lipgloss.Style {
	if item.Glow {
		return lipgloss.NewStyle().Foreground(t.ColorPrimary)
	}
	return lipgloss.NewStyle().Foreground(t.ColorAccent)
}

func (t *Theme) RenderHelpItem(item HelpItem) string {
	return t.RenderHelpItems([]HelpItem{item})
}

func JoinNonEmpty(items []string, sep string) string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return strings.Join(filtered, sep)
}

func HelpItemKeyText(item HelpItem) string {
	if !item.Toggle {
		return item.Key
	}
	if item.On {
		return "+" + item.Key
	}
	return "-" + item.Key
}

func WithHelpDetail(item HelpItem, detail string) HelpItem {
	item.Detail = detail
	return item
}

func LogInfoSection(logFilePath string) HelpSection {
	return HelpSection{
		Title: "Info",
		Items: []HelpItem{{
			Key:    "Log file",
			Desc:   ShortenPath(logFilePath),
			Detail: "diagnostic logs written here",
		}},
	}
}

func VersionInfoSection() HelpSection {
	return HelpSection{
		Title: "Build",
		Items: []HelpItem{{
			Key:  "Version",
			Desc: VersionInfo(),
		}},
	}
}
