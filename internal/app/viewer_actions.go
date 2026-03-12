package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	conv "github.com/rkuska/carn/internal/conversation"
)

type viewerActionMode int

const (
	viewerActionNone viewerActionMode = iota
	viewerActionCopy
	viewerActionOpen
)

type viewerPlanPickerState struct {
	active   bool
	action   viewerActionMode
	selected int
}

func (m viewerActionMode) String() string {
	switch m {
	case viewerActionNone:
		return ""
	case viewerActionCopy:
		return "copy"
	case viewerActionOpen:
		return "open"
	default:
		return ""
	}
}

func (m viewerModel) hasActiveOverlay() bool {
	return m.actionMode != viewerActionNone || m.planPicker.active
}

func (m viewerModel) actionFooterItems() []helpItem {
	items := []helpItem{
		{key: "c", desc: "conversation"},
	}
	if m.content.hasPlans {
		items = append(items, helpItem{key: "p", desc: "plan"})
	}
	items = append(items,
		helpItem{key: "r", desc: "raw"},
		helpItem{key: "?", desc: "help", priority: helpPriorityEssential},
		helpItem{key: "q/esc", desc: "cancel", priority: helpPriorityHigh},
	)
	return items
}

func (m viewerModel) planPickerFooterItems() []helpItem {
	return []helpItem{
		{key: "j/k", desc: "move"},
		{key: "enter", desc: m.planPicker.action.String(), priority: helpPriorityHigh},
		{key: "?", desc: "help", priority: helpPriorityEssential},
		{key: "q/esc", desc: "cancel", priority: helpPriorityHigh},
	}
}

func (m viewerModel) helpSections(extraActions []helpItem) []helpSection {
	switch {
	case m.planPicker.active:
		return m.planPickerHelpSections(extraActions)
	case m.actionMode != viewerActionNone:
		return m.actionHelpSections(extraActions)
	default:
		return transcriptHelpSections(m.opts, m.content, extraActions)
	}
}

func (m viewerModel) actionHelpSections(extraActions []helpItem) []helpSection {
	items := []helpItem{
		{key: "c", desc: "conversation"},
	}
	if m.content.hasPlans {
		items = append(items, helpItem{key: "p", desc: "plan"})
	}
	items = append(items,
		helpItem{key: "r", desc: "raw"},
		helpItem{key: "q/esc", desc: "cancel"},
	)
	items = append(items, extraActions...)
	return []helpSection{{title: "Select Target", items: items}}
}

func (m viewerModel) planPickerHelpSections(extraActions []helpItem) []helpSection {
	items := []helpItem{
		{key: "j/k", desc: "move"},
		{key: "enter", desc: m.planPicker.action.String()},
		{key: "q/esc", desc: "cancel"},
	}
	items = append(items, extraActions...)
	return []helpSection{{title: "Select Plan", items: items}}
}

func (m viewerModel) startActionMode(mode viewerActionMode) viewerModel {
	m.actionMode = mode
	m.planPicker = viewerPlanPickerState{}
	m.pendingGotoTopKey = false
	return m
}

func (m viewerModel) clearActionMode() viewerModel {
	m.actionMode = viewerActionNone
	m.planPicker = viewerPlanPickerState{}
	return m
}

func (m viewerModel) handleActionKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	if m.planPicker.active {
		return m.handlePlanPickerKey(msg)
	}

	if msg.Code == tea.KeyEscape {
		return m.clearActionMode(), nil
	}
	switch msg.Text {
	case "q":
		return m.clearActionMode(), nil
	case "c":
		return m.executeActionTarget(actionTargetConversation)
	case "p":
		if !m.content.hasPlans {
			return m, nil
		}
		return m.executeActionTarget(actionTargetPlan)
	case "r":
		return m.executeActionTarget(actionTargetRaw)
	}
	return m, nil
}

func (m viewerModel) handlePlanPickerKey(msg tea.KeyPressMsg) (viewerModel, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape || msg.Text == "q":
		return m.clearActionMode(), nil
	case msg.Code == tea.KeyUp || msg.Text == "k":
		if m.planPicker.selected > 0 {
			m.planPicker.selected--
		}
		return m, nil
	case msg.Code == tea.KeyDown || msg.Text == "j":
		plans := conv.AllPlans(m.session.Messages)
		if m.planPicker.selected < len(plans)-1 {
			m.planPicker.selected++
		}
		return m, nil
	case msg.Code == tea.KeyEnter:
		return m.executeSelectedPlan()
	}
	return m, nil
}

func (m viewerModel) executeActionTarget(target actionTarget) (viewerModel, tea.Cmd) {
	switch target {
	case actionTargetPlan:
		plans := conv.AllPlans(m.session.Messages)
		switch len(plans) {
		case 0:
			return m, nil
		case 1:
			return m.executePlan(plans[0])
		default:
			m.planPicker = viewerPlanPickerState{
				active: true,
				action: m.actionMode,
			}
			m.actionMode = viewerActionNone
			return m, nil
		}
	case actionTargetConversation:
		cmd := conversationActionCmd(m.actionMode, m.conversation, m.session, m.opts, m.planExpanded)
		return m.clearActionMode(), cmd
	case actionTargetRaw:
		cmd := rawActionCmd(m.actionMode, m.conversation, m.session)
		return m.clearActionMode(), cmd
	default:
		return m, nil
	}
}

func (m viewerModel) executeSelectedPlan() (viewerModel, tea.Cmd) {
	plans := conv.AllPlans(m.session.Messages)
	if len(plans) == 0 {
		return m.clearActionMode(), nil
	}
	selected := min(max(m.planPicker.selected, 0), len(plans)-1)
	return m.executePlan(plans[selected])
}

func (m viewerModel) executePlan(plan conv.Plan) (viewerModel, tea.Cmd) {
	cmd := planActionCmd(m.planPicker.actionOr(m.actionMode), plan)
	return m.clearActionMode(), cmd
}

func (p viewerPlanPickerState) actionOr(fallback viewerActionMode) viewerActionMode {
	if p.action != viewerActionNone {
		return p.action
	}
	return fallback
}

type actionTarget int

const (
	actionTargetConversation actionTarget = iota
	actionTargetPlan
	actionTargetRaw
)

func conversationActionCmd(
	mode viewerActionMode,
	conversation conv.Conversation,
	session conv.Session,
	opts transcriptOptions,
	planExpanded bool,
) tea.Cmd {
	text := renderVisibleConversation(session, opts, planExpanded)
	switch mode {
	case viewerActionNone:
		return nil
	case viewerActionCopy:
		return copyTextCmd(text, "conversation copied to clipboard")
	case viewerActionOpen:
		return openTextInEditorCmd(text, conversationExportFileName(session.Meta))
	default:
		return nil
	}
}

func planActionCmd(mode viewerActionMode, plan conv.Plan) tea.Cmd {
	text := conv.FormatPlan(plan)
	switch mode {
	case viewerActionNone:
		return nil
	case viewerActionCopy:
		return copyTextCmd(text, fmt.Sprintf("plan copied: %s", planFileName(plan)))
	case viewerActionOpen:
		return openTextInEditorCmd(text, planFileName(plan))
	default:
		return nil
	}
}

func rawActionCmd(mode viewerActionMode, conversation conv.Conversation, session conv.Session) tea.Cmd {
	switch mode {
	case viewerActionNone:
		return nil
	case viewerActionCopy:
		return copyRawConversationCmd(conversation, session)
	case viewerActionOpen:
		return openRawConversationCmd(conversation, session)
	default:
		return nil
	}
}
