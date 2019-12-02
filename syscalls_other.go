// +build !linux,!darwin,!windows,!freebsd,!netbsd,!openbsd,!solaris,!illumos

package water

import "errors"

func openDev(config Config) (*Interface, error) {
	return nil, errors.New("not implemented on this platform")
}
