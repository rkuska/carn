package app

import (
	"image/color"

	el "github.com/rkuska/carn/internal/app/elements"
)

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
	renderHelpFooter    = el.RenderHelpFooter
	renderHelpOverlay   = el.RenderHelpOverlay
	joinNonEmpty        = el.JoinNonEmpty
	logInfoSection      = el.LogInfoSection
	versionInfoSection  = el.VersionInfoSection
	renderFramedBox     = el.RenderFramedBox
	infoNotification    = el.InfoNotification
	successNotification = el.SuccessNotification
	errorNotification   = el.ErrorNotification
	renderWrappedTokens = el.RenderWrappedTokens
	renderSingleChip    = el.RenderSingleChip
)

var (
	colorPrimary   color.Color
	colorSecondary color.Color
	colorAccent    color.Color
	colorHighlight color.Color
	colorStatusFg  color.Color

	styleSubtitle  = el.StyleSubtitle
	styleMetaLabel = el.StyleMetaLabel
	styleMetaValue = el.StyleMetaValue
)

func initPalette(hasDarkBG bool) {
	el.InitPalette(hasDarkBG)

	colorPrimary = el.ColorPrimary
	colorSecondary = el.ColorSecondary
	colorAccent = el.ColorAccent
	colorHighlight = el.ColorHighlight
	colorStatusFg = el.ColorStatusFg

	styleSubtitle = el.StyleSubtitle
	styleMetaLabel = el.StyleMetaLabel
	styleMetaValue = el.StyleMetaValue
}
