package app

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
)

type helpItem = el.HelpItem
type helpSection = el.HelpSection
type helpPriority = el.HelpPriority

const (
	helpPriorityLow       = el.HelpPriorityLow
	helpPriorityNormal    = el.HelpPriorityNormal
	helpPriorityHigh      = el.HelpPriorityHigh
	helpPriorityEssential = el.HelpPriorityEssential

	framedFooterRows = el.FramedFooterRows
)

type notification = el.Notification
type notificationMsg = el.NotificationMsg
type clearNotificationMsg = el.ClearNotificationMsg
type notificationKind = el.NotificationKind

const (
	notificationInfo    = el.NotificationInfo
	notificationSuccess = el.NotificationSuccess
	notificationError   = el.NotificationError
)

var (
	renderHelpFooter         = el.RenderHelpFooter
	renderSearchFooter       = el.RenderSearchFooter
	renderHelpItems          = el.RenderHelpItems
	renderFittedHelpItems    = el.RenderFittedHelpItems
	renderHelpItem           = el.RenderHelpItem
	renderHelpOverlay        = el.RenderHelpOverlay
	joinNonEmpty             = el.JoinNonEmpty
	logInfoSection           = el.LogInfoSection
	versionInfoSection       = el.VersionInfoSection
	renderFramedPane         = el.RenderFramedPane
	renderFramedBox          = el.RenderFramedBox
	renderBorderTop          = el.RenderBorderTop
	renderInsetBox           = el.RenderInsetBox
	framedBodyHeight         = el.FramedBodyHeight
	framedFooterContentWidth = el.FramedFooterContentWidth
	composeFooterRow         = el.ComposeFooterRow
	renderNotification       = el.RenderNotification
	infoNotification         = el.InfoNotification
	successNotification      = el.SuccessNotification
	errorNotification        = el.ErrorNotification
	clearNotificationAfter   = el.ClearNotificationAfter
	notificationDuration     = el.NotificationDuration
	fitToWidth               = el.FitToWidth
	renderWrappedTokens      = el.RenderWrappedTokens
	renderSingleChip         = el.RenderSingleChip
)

var (
	colorPrimary     color.Color
	colorSecondary   color.Color
	colorAccent      color.Color
	colorHighlight   color.Color
	colorSelectedFg  color.Color
	colorDiffRemove  color.Color
	colorDiffHunk    color.Color
	colorToolBg      color.Color
	colorFgOnBg      color.Color
	colorStatusFg    color.Color
	colorNormalTitle color.Color
	colorNormalDesc  color.Color
	colorTitleFg     color.Color
	colorChartBar    color.Color
	colorChartToken  color.Color
	colorChartTime   color.Color
	colorChartError  color.Color
	colorHeatmap0    color.Color
	colorHeatmap1    color.Color
	colorHeatmap2    color.Color
	colorHeatmap3    color.Color
	colorHeatmap4    color.Color

	styleSubtitle             = el.StyleSubtitle
	styleToolCall             = el.StyleToolCall
	styleToolCallItalic       = el.StyleToolCallItalic
	styleMetaLabel            = el.StyleMetaLabel
	styleMetaValue            = el.StyleMetaValue
	styleSearchMatch          = el.StyleSearchMatch
	styleCurrentMatch         = el.StyleCurrentMatch
	styleRuleHR               = el.StyleRuleHR
	styleBadgeUser            = el.StyleBadgeUser
	styleBadgeAssistant       = el.StyleBadgeAssistant
	styleBadgeSystem          = el.StyleBadgeSystem
	styleThinkLabel           = el.StyleThinkLabel
	styleThinkBorder          = el.StyleThinkBorder
	styleThinkLine            = el.StyleThinkLine
	styleSelectedPreview      = el.StyleSelectedPreview
	styleNormalPreview        = el.StyleNormalPreview
	styleDimmedPreview        = el.StyleDimmedPreview
	styleDiffBg               = el.StyleDiffBg
	styleDiffAdd              = el.StyleDiffAdd
	styleDiffRemoveLine       = el.StyleDiffRemoveLine
	styleDiffHunkLine         = el.StyleDiffHunkLine
	styleToolResultBadge      = el.StyleToolResultBadge
	styleToolResultErrorBadge = el.StyleToolResultErrorBadge
	stylePaneTitle            = el.StylePaneTitle
)

func initPalette(hasDarkBG bool) {
	el.InitPalette(hasDarkBG)

	colorPrimary = el.ColorPrimary
	colorSecondary = el.ColorSecondary
	colorAccent = el.ColorAccent
	colorHighlight = el.ColorHighlight
	colorSelectedFg = el.ColorSelectedFg
	colorDiffRemove = el.ColorDiffRemove
	colorDiffHunk = el.ColorDiffHunk
	colorToolBg = el.ColorToolBg
	colorFgOnBg = el.ColorFgOnBg
	colorStatusFg = el.ColorStatusFg
	colorNormalTitle = el.ColorNormalTitle
	colorNormalDesc = el.ColorNormalDesc
	colorTitleFg = el.ColorTitleFg
	colorChartBar = el.ColorChartBar
	colorChartToken = el.ColorChartToken
	colorChartTime = el.ColorChartTime
	colorChartError = el.ColorChartError
	colorHeatmap0 = el.ColorHeatmap0
	colorHeatmap1 = el.ColorHeatmap1
	colorHeatmap2 = el.ColorHeatmap2
	colorHeatmap3 = el.ColorHeatmap3
	colorHeatmap4 = el.ColorHeatmap4

	styleSubtitle = el.StyleSubtitle
	styleToolCall = el.StyleToolCall
	styleToolCallItalic = el.StyleToolCallItalic
	styleMetaLabel = el.StyleMetaLabel
	styleMetaValue = el.StyleMetaValue
	styleSearchMatch = el.StyleSearchMatch
	styleCurrentMatch = el.StyleCurrentMatch
	styleRuleHR = el.StyleRuleHR
	styleBadgeUser = el.StyleBadgeUser
	styleBadgeAssistant = el.StyleBadgeAssistant
	styleBadgeSystem = el.StyleBadgeSystem
	styleThinkLabel = el.StyleThinkLabel
	styleThinkBorder = el.StyleThinkBorder
	styleThinkLine = el.StyleThinkLine
	styleSelectedPreview = el.StyleSelectedPreview
	styleNormalPreview = el.StyleNormalPreview
	styleDimmedPreview = el.StyleDimmedPreview
	styleDiffBg = el.StyleDiffBg
	styleDiffAdd = el.StyleDiffAdd
	styleDiffRemoveLine = el.StyleDiffRemoveLine
	styleDiffHunkLine = el.StyleDiffHunkLine
	styleToolResultBadge = el.StyleToolResultBadge
	styleToolResultErrorBadge = el.StyleToolResultErrorBadge
	stylePaneTitle = el.StylePaneTitle
}
