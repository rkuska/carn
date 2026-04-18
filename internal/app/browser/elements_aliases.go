package browser

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
)

type helpItem = el.HelpItem
type helpSection = el.HelpSection

const (
	helpPriorityLow       = el.HelpPriorityLow
	helpPriorityNormal    = el.HelpPriorityNormal
	helpPriorityHigh      = el.HelpPriorityHigh
	helpPriorityEssential = el.HelpPriorityEssential
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

type browserFilterState = el.FilterState
type filterDimension = el.FilterDimension
type dimensionFilter = el.DimensionFilter
type boolFilterState = el.BoolFilterState

const (
	filterDimProvider  = el.FilterDimProvider
	filterDimProject   = el.FilterDimProject
	filterDimModel     = el.FilterDimModel
	filterDimGitBranch = el.FilterDimGitBranch
	filterDimHasPlans  = el.FilterDimHasPlans
	filterDimMultiPart = el.FilterDimMultiPart
	filterDimCount     = el.FilterDimCount

	boolFilterAny = el.BoolFilterAny
	boolFilterYes = el.BoolFilterYes
	boolFilterNo  = el.BoolFilterNo
)

var (
	renderHelpFooter                     = el.RenderHelpFooter
	renderSearchFooter                   = el.RenderSearchFooter
	renderHelpItems                      = el.RenderHelpItems
	renderHelpItem                       = el.RenderHelpItem
	renderHelpOverlay                    = el.RenderHelpOverlay
	joinNonEmpty                         = el.JoinNonEmpty
	withHelpDetail                       = el.WithHelpDetail
	logInfoSection                       = el.LogInfoSection
	versionInfoSection                   = el.VersionInfoSection
	renderFramedPane                     = el.RenderFramedPane
	renderInsetBox                       = el.RenderInsetBox
	framedBodyHeight                     = el.FramedBodyHeight
	infoNotification                     = el.InfoNotification
	successNotification                  = el.SuccessNotification
	errorNotification                    = el.ErrorNotification
	notificationDuration                 = el.NotificationDuration
	clearNotificationAfter               = el.ClearNotificationAfter
	notificationCmd                      = el.NotificationCmd
	newBrowserFilterState                = el.NewFilterState
	filterDimensionLabel                 = el.FilterDimensionLabel
	filterDimensionIsBool                = el.FilterDimensionIsBool
	extractFilterValues                  = el.ExtractFilterValues
	applyStructuredFilters               = el.ApplyStructuredFilters
	filterBadges                         = el.FilterBadges
	cycleBoolFilter                      = el.CycleBoolFilter
	renderFilterOverlayWithConversations = el.RenderFilterOverlayWithConversations
	filterFooterStatusParts              = el.FilterFooterStatusParts
	filterFooterItems                    = el.FilterFooterItems
	copyBrowserFilterState               = el.CopyFilterState
	fitToWidth                           = el.FitToWidth
)

func resumeErrorNotification(err error, cwd string) notificationMsg {
	return el.FormatResumeErrorNotification(err, cwd, errResumeProviderUnavailable)
}

var (
	colorPrimary    color.Color
	colorSecondary  color.Color
	colorAccent     color.Color
	colorHighlight  color.Color
	colorSelectedFg color.Color
	colorDiffRemove color.Color
	colorStatusFg   color.Color
	colorNormalDesc color.Color

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
)

func syncPaletteFromElements() {
	if el.ColorPrimary == nil {
		el.InitPalette(true)
	}
	colorPrimary = el.ColorPrimary
	colorSecondary = el.ColorSecondary
	colorAccent = el.ColorAccent
	colorHighlight = el.ColorHighlight
	colorSelectedFg = el.ColorSelectedFg
	colorDiffRemove = el.ColorDiffRemove
	colorStatusFg = el.ColorStatusFg
	colorNormalDesc = el.ColorNormalDesc

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
}
