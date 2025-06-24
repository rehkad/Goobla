//go:build !windows

package tray

import (
	"errors"

	"github.com/goobla/goobla/app/tray/commontray"
)

func InitPlatformTray(icon, updateIcon []byte) (commontray.GooblaTray, error) {
	return nil, errors.New("not implemented")
}
