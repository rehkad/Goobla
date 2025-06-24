package tray

import (
	"fmt"
	"runtime"

	"github.com/goobla/goobla/app/assets"
	"github.com/goobla/goobla/app/tray/commontray"
)

func NewTray() (commontray.GooblaTray, error) {
	extension := ".png"
	if runtime.GOOS == "windows" {
		extension = ".ico"
	}
	iconName := commontray.UpdateIconName + extension
	updateIcon, err := assets.GetIcon(iconName)
	if err != nil {
		return nil, fmt.Errorf("failed to load icon %s: %w", iconName, err)
	}
	iconName = commontray.IconName + extension
	icon, err := assets.GetIcon(iconName)
	if err != nil {
		return nil, fmt.Errorf("failed to load icon %s: %w", iconName, err)
	}

	return InitPlatformTray(icon, updateIcon)
}
