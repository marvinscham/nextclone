package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed sync_512.png
var appIcon []byte

//go:embed globe.svg
var globeIcon []byte

var AppIcon = fyne.NewStaticResource("sync_512.png", appIcon)
var GlobeIcon = fyne.NewStaticResource("globe.svg", globeIcon)
