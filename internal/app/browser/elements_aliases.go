package browser

import el "github.com/rkuska/carn/internal/app/elements"

type helpItem = el.HelpItem
type helpSection = el.HelpSection

const (
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
	renderHelpFooter                     = (*el.Theme).RenderHelpFooter
	renderSearchFooter                   = (*el.Theme).RenderSearchFooter
	renderHelpItems                      = (*el.Theme).RenderHelpItems
	renderHelpItem                       = (*el.Theme).RenderHelpItem
	renderHelpOverlay                    = (*el.Theme).RenderHelpOverlay
	joinNonEmpty                         = el.JoinNonEmpty
	withHelpDetail                       = el.WithHelpDetail
	logInfoSection                       = el.LogInfoSection
	versionInfoSection                   = el.VersionInfoSection
	renderFramedPane                     = (*el.Theme).RenderFramedPane
	renderInsetBox                       = el.RenderInsetBox
	framedBodyHeight                     = el.FramedBodyHeight
	appendCmd                            = el.AppendCmd
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
	renderFilterOverlayWithConversations = (*el.Theme).RenderFilterOverlayWithConversations
	filterFooterStatusParts              = el.FilterFooterStatusParts
	filterFooterItems                    = el.FilterFooterItems
	copyBrowserFilterState               = el.CopyFilterState
	fitToWidth                           = el.FitToWidth
	renderWrappedTokens                  = el.RenderWrappedTokens
	renderSingleChip                     = (*el.Theme).RenderSingleChip
)

func resumeErrorNotification(err error, cwd string) notificationMsg {
	return el.FormatResumeErrorNotification(err, cwd, errResumeProviderUnavailable)
}
