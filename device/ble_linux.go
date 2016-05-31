// +build linux

package device

import "github.com/paypal/gatt"

var gattOption = []gatt.Option{
	gatt.LnxMaxConnections(1),
	gatt.LnxDeviceID(-1, true),
}
