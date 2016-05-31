// +build darwin

package device

import "github.com/paypal/gatt"

var gattOption = []gatt.Option{
	gatt.MacDeviceRole(gatt.CentralManager),
}
