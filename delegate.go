package main

import (
	"github.com/charmbracelet/bubbles/list"
)

func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	d.SetSpacing(1)
	d.SetHeight(3)
	return d
}
