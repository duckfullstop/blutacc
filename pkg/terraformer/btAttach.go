package terraformer

import (
	"errors"
	"fmt"
	"log"
	"tinygo.org/x/bluetooth"
)

// stubby AF, probably pass this through at a later point
// or just refactor this entire package to not touch bluetooth directly maybe?
var adapter = bluetooth.DefaultAdapter

func BluetoothResultToTacw(target bluetooth.ScanResult) (acw *TerraACWallbox, err error) {
	log.Printf("🔌 Connecting to %s", target.Address)
	device, err := adapter.Connect(target.Address, bluetooth.ConnectionParams{})
	if err != nil {
		if device != nil {
			device.Disconnect()
		}
		return nil, errors.New(fmt.Sprintf("error connecting to CP: %s", err.Error()))
	}

	acwb, err := NewTerraACWallbox(device, target.LocalName())
	if err != nil {
		return acwb, err
	}

	log.Printf("✅ Connected to %s", target.LocalName())

	err = acwb.Auth()
	if err != nil {
		acwb.Disconnect()
		return acwb, errors.New(fmt.Sprintf("failed to authenticate with TACW: %s", err))
	}
	return acwb, nil
}
