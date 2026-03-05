package main

import (
	"charm.land/bubbles/v2/list"
)

func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	d.SetSpacing(1)
	d.SetHeight(3)
	return d
}
