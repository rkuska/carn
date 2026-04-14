package browser

import (
	"fmt"

	"charm.land/bubbles/v2/viewport"
)

func viewerLineRangeStatus(v viewport.Model) string {
	total := v.TotalLineCount()
	if total == 0 {
		return ""
	}

	first := v.YOffset() + 1
	visible := max(v.VisibleLineCount(), 1)
	last := min(v.YOffset()+visible, total)
	return fmt.Sprintf("L%d-%d/%d", first, last, total)
}
