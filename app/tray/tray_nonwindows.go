//go:build !windows

package tray

import (
	"errors"

	"github.com/moogla/moogla/app/tray/commontray"
)

func InitPlatformTray(icon, updateIcon []byte) (commontray.MooglaTray, error) {
	return nil, errors.New("not implemented")
}
