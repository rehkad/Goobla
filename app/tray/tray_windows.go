package tray

import (
	"github.com/goobla/goobla/app/tray/commontray"
	"github.com/goobla/goobla/app/tray/wintray"
)

func InitPlatformTray(icon, updateIcon []byte) (commontray.GooblaTray, error) {
	return wintray.InitTray(icon, updateIcon)
}
