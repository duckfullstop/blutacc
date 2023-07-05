package main

import "tinygo.org/x/bluetooth"

// This whole file is basically a hacky workaround because @duckfullstop's MacBook is very, very old and broken
// and can't properly scan for BTLE devices
// This lets us set our own bluetooth.AdvertisementFields with a stubbed LocalName
// because some code depends on LocalName() for the device SN.

// advertisementFields wraps AdvertisementFields to implement the
// AdvertisementPayload interface. The methods to implement the interface (such
// as LocalName) cannot be implemented on AdvertisementFields because they would
// conflict with field names.
type advertisementFields struct {
	bluetooth.AdvertisementFields
}

// LocalName returns the underlying LocalName field.
func (p advertisementFields) LocalName() string {
	return p.AdvertisementFields.LocalName
}

// HasServiceUUID returns true whether the given UUID is present in the
// advertisement payload as a Service Class UUID.
func (p advertisementFields) HasServiceUUID(uuid bluetooth.UUID) bool {
	for _, u := range p.AdvertisementFields.ServiceUUIDs {
		if u == uuid {
			return true
		}
	}
	return false
}

// Bytes returns nil, as structured advertisement data does not have the
// original raw advertisement data available.
func (p advertisementFields) Bytes() []byte {
	return nil
}
