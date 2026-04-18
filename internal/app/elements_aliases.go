package app

import el "github.com/rkuska/carn/internal/app/elements"

type helpItem = el.HelpItem
type helpSection = el.HelpSection

const (
	helpPriorityHigh      = el.HelpPriorityHigh
	helpPriorityEssential = el.HelpPriorityEssential

	framedFooterRows = el.FramedFooterRows
)

type notification = el.Notification

const (
	notificationInfo = el.NotificationInfo
)

var (
	appendCmd           = el.AppendCmd
	renderHelpFooter    = (*el.Theme).RenderHelpFooter
	renderHelpOverlay   = (*el.Theme).RenderHelpOverlay
	joinNonEmpty        = el.JoinNonEmpty
	logInfoSection      = el.LogInfoSection
	versionInfoSection  = el.VersionInfoSection
	renderFramedBox     = (*el.Theme).RenderFramedBox
	infoNotification    = el.InfoNotification
	successNotification = el.SuccessNotification
	errorNotification   = el.ErrorNotification
	renderWrappedTokens = el.RenderWrappedTokens
	renderSingleChip    = (*el.Theme).RenderSingleChip
)
