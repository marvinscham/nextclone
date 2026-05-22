package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed sync_512.png
var appIcon []byte

var AppIcon = fyne.NewStaticResource("sync_512.png", appIcon)
