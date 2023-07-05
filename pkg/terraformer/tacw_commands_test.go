package terraformer

import (
	"encoding/hex"
	"testing"
)

func TestDecodeStatusPacketSessionInProgress(t *testing.T) {
	testPacket, _ := hex.DecodeString("06009c31746101003c5a00000e0c000000002001")
	testResult := TerraStatus{
		status:              0x06,
		PhaseLineType:       0,
		ChargeOrderSequence: 1635004828,
		ChargeElectricity:   0.01,
		ChargeVoltageP1:     231,
		ChargeCurrentP1:     30.86,
		ChargeVoltageP2:     0,
		ChargeCurrentP2:     0,
		ChargeVoltageP3:     0,
		ChargeCurrentP3:     0,
		ChargeDuration:      0,
		RatedCurrent:        32,
	}
	statusOutput := TerraStatus{}
	statusOutput.readFromBytes(testPacket)
	if statusOutput == testResult {
		t.Error("function output does not match expected decode")
	}
}

func TestDecodeStatusPacketSessionNotPlugged(t *testing.T) {
	testPacket, _ := hex.DecodeString("0001")
	testResult := TerraStatus{
		status: 0x01,
	}
	statusOutput := TerraStatus{}
	statusOutput.readFromBytes(testPacket)
	if statusOutput == testResult {
		t.Error("function output does not match expected decode")
	}
}
