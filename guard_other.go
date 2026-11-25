//go:build !windows

package main

import "errors"

// The safeguard is defined in terms of a Windows drive letter and volume
// label, which have no equivalent here. Fail closed: no writes on this OS
// until an equivalent rule is designed.
func platformLookupVolume(path string) (volumeInfo, error) {
	return volumeInfo{}, errors.New("writes are only supported on Windows (drive H: labelled UNDELETE)")
}
