package stats

import el "github.com/rkuska/carn/internal/app/elements"

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

const notificationError = el.NotificationError

type browserFilterState = el.FilterState
type filterDimension = el.FilterDimension
type dimensionFilter = el.DimensionFilter

const (
	filterDimProvider = el.FilterDimProvider
	filterDimProject  = el.FilterDimProject
	filterDimModel    = el.FilterDimModel
	filterDimVersion  = el.FilterDimVersion
	filterDimHasPlans = el.FilterDimHasPlans
	filterDimCount    = el.FilterDimCount

	boolFilterYes = el.BoolFilterYes

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
	renderHelpOverlay      = (*el.Theme).RenderHelpOverlay
	renderHelpItems        = (*el.Theme).RenderHelpItems
	renderFittedHelpItems  = (*el.Theme).RenderFittedHelpItems
	joinNonEmpty           = el.JoinNonEmpty
	renderNotification     = (*el.Theme).RenderNotification
	clearNotificationAfter = el.ClearNotificationAfter
	errorNotification      = el.ErrorNotification

	renderBorderTop          = (*el.Theme).RenderBorderTop
	renderFramedPane         = (*el.Theme).RenderFramedPane
	renderFramedBox          = (*el.Theme).RenderFramedBox
	renderInlineTitledRule   = (*el.Theme).RenderInlineTitledRule
	framedFooterContentWidth = el.FramedFooterContentWidth
	composeFooterRow         = el.ComposeFooterRow

	newBrowserFilterState      = el.NewFilterState
	filterDimensionIsBool      = el.FilterDimensionIsBool
	extractFilterValues        = el.ExtractFilterValues
	applyStructuredFilters     = el.ApplyStructuredFilters
	filterBadges               = el.FilterBadges
	cycleBoolFilter            = el.CycleBoolFilter
	renderFilterDimensionRow   = (*el.Theme).RenderFilterDimensionRow
	renderFilterExpandedValues = (*el.Theme).RenderFilterExpandedValues
	filterDimensionFooterItems = el.FilterDimensionFooterItems
	copyBrowserFilterState     = el.CopyFilterState

	noDataLabel      = el.NoDataLabel
	fitToWidth       = el.FitToWidth
	splitAndFitLines = el.SplitAndFitLines
	formatFloat      = el.FormatFloat
	scaledWidth      = el.ScaledWidth

	renderHorizontalBars            = (*el.Theme).RenderHorizontalBars
	renderHorizontalBarsBody        = (*el.Theme).RenderHorizontalBarsBody
	renderRankedTable               = (*el.Theme).RenderRankedTable
	renderRankedTableBody           = (*el.Theme).RenderRankedTableBody
	renderSideBySide                = (*el.Theme).RenderSideBySide
	renderColumns                   = (*el.Theme).RenderColumns
	statsColumnWidths               = el.StatsColumnWidths
	renderPreformattedColumns       = (*el.Theme).RenderPreformattedColumns
	renderStatsTitle                = (*el.Theme).RenderStatsTitle
	renderTokenValue                = (*el.Theme).RenderTokenValue
	renderSummaryChips              = (*el.Theme).RenderSummaryChips
	renderSparkline                 = el.RenderSparkline
	renderDailyRateColumnChart      = (*el.Theme).RenderDailyRateColumnChart
	dailyRateBarSlots               = el.DailyRateBarSlots
	renderVerticalHistogram         = (*el.Theme).RenderVerticalHistogram
	renderVerticalHistogramBody     = (*el.Theme).RenderVerticalHistogramBody
	renderActivityHeatmap           = (*el.Theme).RenderActivityHeatmap
	renderActivityHeatmapBody       = (*el.Theme).RenderActivityHeatmapBody
	renderHorizontalStackedBarsBody = el.RenderHorizontalStackedBarsBody
	resolveStackedBarWidths         = el.ResolveStackedBarWidths
)
