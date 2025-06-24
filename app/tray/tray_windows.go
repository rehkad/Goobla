package tray

import (
	"github.com/moogla/moogla/app/tray/commontray"
	"github.com/moogla/moogla/app/tray/wintray"
)

func InitPlatformTray(icon, updateIcon []byte) (commontray.MooglaTray, error) {
	return wintray.InitTray(icon, updateIcon)
}
