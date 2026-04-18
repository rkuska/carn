package stats

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

const (
	notificationInfo    = el.NotificationInfo
	notificationSuccess = el.NotificationSuccess
	notificationError   = el.NotificationError
)

type browserFilterState = el.FilterState
type filterDimension = el.FilterDimension
type dimensionFilter = el.DimensionFilter

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

	framedFooterRows       = el.FramedFooterRows
	filterOverlayIndent    = el.FilterOverlayIndent
	filterOverlayCursorOn  = el.FilterOverlayCursorOn
	filterOverlayCursorOff = el.FilterOverlayCursorOff
	filterOverlayCheckOff  = el.FilterOverlayCheckOff
)

type barItem = el.BarItem
type histBucket = el.HistBucket
type tableRow = el.TableRow
type chip = el.Chip
type stackedRowSegment = el.StackedRowSegment
type stackedRowItem = el.StackedRowItem
type dailyRateBucket = el.DailyRateBucket

var (
	renderHelpOverlay      = el.RenderHelpOverlay
	renderHelpItems        = el.RenderHelpItems
	renderFittedHelpItems  = el.RenderFittedHelpItems
	joinNonEmpty           = el.JoinNonEmpty
	renderNotification     = el.RenderNotification
	clearNotificationAfter = el.ClearNotificationAfter
	errorNotification      = el.ErrorNotification

	renderBorderTop          = el.RenderBorderTop
	renderFramedPane         = el.RenderFramedPane
	renderFramedBox          = el.RenderFramedBox
	framedFooterContentWidth = el.FramedFooterContentWidth
	composeFooterRow         = el.ComposeFooterRow

	newBrowserFilterState      = el.NewFilterState
	filterDimensionIsBool      = el.FilterDimensionIsBool
	extractFilterValues        = el.ExtractFilterValues
	applyStructuredFilters     = el.ApplyStructuredFilters
	filterBadges               = el.FilterBadges
	cycleBoolFilter            = el.CycleBoolFilter
	renderFilterDimensionRow   = el.RenderFilterDimensionRow
	renderFilterExpandedValues = el.RenderFilterExpandedValues
	filterDimensionFooterItems = el.FilterDimensionFooterItems
	copyBrowserFilterState     = el.CopyFilterState
	renderSelectionSummary     = el.RenderSelectionSummary

	noDataLabel      = el.NoDataLabel
	fitToWidth       = el.FitToWidth
	splitAndFitLines = el.SplitAndFitLines
	formatFloat      = el.FormatFloat
	scaledWidth      = el.ScaledWidth

	renderHorizontalBars            = el.RenderHorizontalBars
	renderHorizontalBarsBody        = el.RenderHorizontalBarsBody
	renderRankedTable               = el.RenderRankedTable
	renderRankedTableBody           = el.RenderRankedTableBody
	renderSideBySide                = el.RenderSideBySide
	renderColumns                   = el.RenderColumns
	statsColumnWidths               = el.StatsColumnWidths
	renderPreformattedColumns       = el.RenderPreformattedColumns
	renderStatsTitle                = el.RenderStatsTitle
	renderTokenValue                = el.RenderTokenValue
	renderSummaryChips              = el.RenderSummaryChips
	renderSparkline                 = el.RenderSparkline
	renderDailyRateColumnChart      = el.RenderDailyRateColumnChart
	dailyRateBarSlots               = el.DailyRateBarSlots
	renderVerticalHistogram         = el.RenderVerticalHistogram
	renderVerticalHistogramBody     = el.RenderVerticalHistogramBody
	renderActivityHeatmap           = el.RenderActivityHeatmap
	renderActivityHeatmapBody       = el.RenderActivityHeatmapBody
	histogramAxisLabel              = el.HistogramAxisLabel
	histogramAxisLine               = el.HistogramAxisLine
	renderHorizontalStackedBarsBody = el.RenderHorizontalStackedBarsBody
	resolveStackedBarWidths         = el.ResolveStackedBarWidths
)

var (
	colorPrimary     color.Color
	colorSecondary   color.Color
	colorAccent      color.Color
	colorDiffRemove  color.Color
	colorDiffHunk    color.Color
	colorNormalTitle color.Color
	colorNormalDesc  color.Color
	colorTitleFg     color.Color
	colorChartBar    color.Color
	colorChartToken  color.Color
	colorChartTime   color.Color
	colorChartError  color.Color
	colorHeatmap4    color.Color

	styleToolCall  = el.StyleToolCall
	styleMetaLabel = el.StyleMetaLabel
	styleMetaValue = el.StyleMetaValue
	styleRuleHR    = el.StyleRuleHR
)

func syncPaletteFromElements() {
	if el.ColorPrimary == nil {
		el.InitPalette(true)
	}
	colorPrimary = el.ColorPrimary
	colorSecondary = el.ColorSecondary
	colorAccent = el.ColorAccent
	colorDiffRemove = el.ColorDiffRemove
	colorDiffHunk = el.ColorDiffHunk
	colorNormalTitle = el.ColorNormalTitle
	colorNormalDesc = el.ColorNormalDesc
	colorTitleFg = el.ColorTitleFg
	colorChartBar = el.ColorChartBar
	colorChartToken = el.ColorChartToken
	colorChartTime = el.ColorChartTime
	colorChartError = el.ColorChartError
	colorHeatmap4 = el.ColorHeatmap4

	styleToolCall = el.StyleToolCall
	styleMetaLabel = el.StyleMetaLabel
	styleMetaValue = el.StyleMetaValue
	styleRuleHR = el.StyleRuleHR
}
