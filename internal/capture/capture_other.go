//go:build !linux && !darwin && !windows

package capture

import "fmt"

// OpenLive is not implemented on this platform.
func OpenLive(iface string, filter string) (Source, error) {
	return nil, fmt.Errorf("capture: live capture is not supported on this platform")
}
