package browser

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
	items := append([]helpItem{}, m.actionTargetItems(m.actionMode)...)
	items = append(items,
		helpItem{Key: "r", Desc: "raw"},
		helpItem{Key: "?", Desc: "help", Priority: helpPriorityEssential},
		helpItem{Key: "q/esc", Desc: "cancel", Priority: helpPriorityHigh},
	)
	return items
}

func (m viewerModel) planPickerFooterItems() []helpItem {
	return []helpItem{
		{Key: "j/k", Desc: "move"},
		{Key: "enter", Desc: m.planPicker.action.String(), Priority: helpPriorityHigh},
		{Key: "?", Desc: "help", Priority: helpPriorityEssential},
		{Key: "q/esc", Desc: "cancel", Priority: helpPriorityHigh},
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
	items := append([]helpItem{}, m.actionTargetItems(m.actionMode)...)
	items = append(items,
		withHelpDetail(m.rawActionTargetItem(m.actionMode), viewerActionTargetDetail(actionTargetRaw, m.actionMode)),
		helpItem{Key: "q/esc", Desc: "cancel", Detail: "close the target picker without taking action"},
	)
	items = append(items, extraActions...)
	return []helpSection{{Title: "Select Target", Items: items}}
}

func (m viewerModel) planPickerHelpSections(extraActions []helpItem) []helpSection {
	action := m.planPicker.action.String()
	items := []helpItem{
		{Key: "j/k", Desc: "move", Detail: "move between available plans"},
		{Key: "enter", Desc: action, Detail: fmt.Sprintf("%s the selected plan", action)},
		{Key: "q/esc", Desc: "cancel", Detail: "close the plan picker without taking action"},
	}
	items = append(items, extraActions...)
	return []helpSection{{Title: "Select Plan", Items: items}}
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

func (m viewerModel) actionTargetItems(mode viewerActionMode) []helpItem {
	items := []helpItem{
		withHelpDetail(
			helpItem{Key: "c", Desc: "conversation"},
			viewerActionTargetDetail(actionTargetConversation, mode),
		),
	}
	if m.content.hasPlans {
		items = append(items, withHelpDetail(
			helpItem{Key: "p", Desc: "plan"},
			viewerActionTargetDetail(actionTargetPlan, mode),
		))
	}
	return items
}

func (m viewerModel) rawActionTargetItem(mode viewerActionMode) helpItem {
	return withHelpDetail(
		helpItem{Key: "r", Desc: "raw"},
		viewerActionTargetDetail(actionTargetRaw, mode),
	)
}

func viewerActionTargetDetail(target actionTarget, mode viewerActionMode) string {
	switch mode {
	case viewerActionNone:
		return ""
	case viewerActionCopy:
		return viewerCopyTargetDetail(target)
	case viewerActionOpen:
		return viewerOpenTargetDetail(target)
	default:
		return ""
	}
}

func viewerCopyTargetDetail(target actionTarget) string {
	switch target {
	case actionTargetConversation:
		return "copy the visible conversation to the clipboard"
	case actionTargetPlan:
		return "copy the selected plan to the clipboard"
	case actionTargetRaw:
		return "copy the concatenated raw conversation files"
	default:
		return ""
	}
}

func viewerOpenTargetDetail(target actionTarget) string {
	switch target {
	case actionTargetConversation:
		return "open the visible conversation in $EDITOR"
	case actionTargetPlan:
		return "open the selected plan in $EDITOR"
	case actionTargetRaw:
		return "open the concatenated raw conversation files in $EDITOR"
	default:
		return ""
	}
}

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
		return openTextInEditorCmd(text, conversationExportFileName(conversation, session.Meta))
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
