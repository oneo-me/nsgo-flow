package main

//go:generate go run ./tools/iconsgen

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"

	"github.com/oneo-me/nsgo-scenra/flowfile"
)

// iconNamed is the icon a design names, by its Lucide name or an alias;
// one NSGo lacks draws as a circle.
func iconNamed(name string) func() *core.SVG {
	name, _ = flowfile.IconName(name)
	if icon, ok := lucideIcons[name]; ok {
		return icon
	}
	return lucide.Circle
}
