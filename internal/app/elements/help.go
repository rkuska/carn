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

func RenderHelpFooter(width int, items []HelpItem, rightParts []string, n Notification) string {
	contentWidth := FramedFooterContentWidth(width)
	minLeftWidth := essentialHelpWidth(items)
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
		ComposeFooterRow(width, RenderFittedHelpItems(items, leftWidth), right),
		RenderNotification(n),
	)
}

func RenderSearchFooter(width int, prompt, right string, n Notification) string {
	return RenderFramedFooter(
		width,
		ComposeFooterRow(width, prompt, right),
		RenderNotification(n),
	)
}

func RenderHelpItems(items []HelpItem) string {
	return renderHelpItemsWithKeep(items, nil)
}

func RenderFittedHelpItems(items []HelpItem, width int) string {
	if width <= 0 {
		return ""
	}

	keep := keepHelpItems(items, width)
	return FitToWidth(renderHelpItemsWithKeep(items, keep), width)
}

func renderHelpItemsWithKeep(items []HelpItem, keep []bool) string {
	helpStyle := lipgloss.NewStyle().Foreground(ColorSecondary)

	parts := make([]string, 0, len(items))
	for i, item := range items {
		if keep != nil && !keep[i] {
			continue
		}
		if item.Key == "" || item.Desc == "" {
			continue
		}

		keyText := HelpItemKeyText(item)
		parts = append(parts, helpItemKeyStyle(item).Render(keyText)+helpStyle.Render(" "+item.Desc))
	}

	return strings.Join(parts, "  ")
}

func helpItemKeyStyle(item HelpItem) lipgloss.Style {
	if item.Glow {
		return lipgloss.NewStyle().Foreground(ColorPrimary)
	}
	return lipgloss.NewStyle().Foreground(ColorAccent)
}

func RenderHelpItem(item HelpItem) string {
	return RenderHelpItems([]HelpItem{item})
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
