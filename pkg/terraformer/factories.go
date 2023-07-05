package terraformer

import (
	"encoding/binary"
	"fmt"
)

func ocppServerConfigureFactory() (packet []byte, cmd int) {
	// conf
	cServerEnable := 1
	cDomainUrl := "wss://ocpp.partyparrot.moe:6277/ocpp"
	var cPort uint16 = 6277
	cProtocolType := 1
	cProtocolVersion := "ocpp16j"
	cSecurityKey := "1zysdonl"
	cTlsEnable := 1
	cCertificateNo := 0
	var cCertificateSize uint32 = 0
	var cDownloadBytes uint16 = 512 // static? never set by app, unsure

	// actual factory bs

	domainUrlLength := len(cDomainUrl)
	securityKeyLength := len(cSecurityKey)
	pArr := make([]byte, domainUrlLength+24+securityKeyLength)

	pArr[0] = byte(cServerEnable)
	pArr[1] = byte(domainUrlLength)
	copy(pArr[2:], cDomainUrl)

	binary.LittleEndian.PutUint16(pArr[domainUrlLength+2:], cPort)
	pArr[domainUrlLength+4] = byte(cProtocolType)

	// pad to 10 characters
	copy(pArr[domainUrlLength+5:], fmt.Sprintf("%-10v", cProtocolVersion))

	pArr[domainUrlLength+15] = byte(securityKeyLength)

	if securityKeyLength > 0 {
		copy(pArr[domainUrlLength+16:], cSecurityKey)
	}

	pArr[domainUrlLength+16+securityKeyLength] = byte(cTlsEnable)
	if cTlsEnable != 0 {
		pArr[domainUrlLength+17+securityKeyLength] = byte(cCertificateNo)
		binary.LittleEndian.PutUint32(pArr[domainUrlLength+18+securityKeyLength:], cCertificateSize)
		binary.LittleEndian.PutUint16(pArr[domainUrlLength+22+securityKeyLength:], cDownloadBytes)
	}

	return pArr, 214

}

func wifiInfoFactory() (packet []byte) {
	return make([]byte, 0)
}

func historyChangeRecordFactory() (packet []byte) {
	return make([]byte, 1)
}

func readDebugErrorLogFactory() (packet []byte, cmd int) {
	return make([]byte, 0), 52
}

func readSysInfoFactory() (packet []byte, cmd int) {
	pArr := make([]byte, 1)
	pArr[0] = 0x01
	return pArr, 194
}

func readConfigurationFactory() (packet []byte, cmd int) {
	pArr := make([]byte, 1)
	pArr[0] = 0x01
	return pArr, 197
}

func writeConfigurationFactory() (packet []byte, cmd int) {
	// jsonString := "{\"socket_a_enable\": [17, 1], \"server_a_ip_address\": [17, \"10.10.0.156\"], \"server_a_port\": [17, 6277], \"server_a_protocol\": [17, \"OCPP\"], \"protocol_a_version\": [17, \"1.6\"], \"encrypted_a\": [17, 0]}"
	jsonString := "{\"socket_a_ip_address\": [51, \"10.10.0.156\"]}"
	// jsonString := "{\"socket_a_enable\": [51, 1], " +
	//     "\"server_a_ip_address\": [51, \"10.10.0.156\"], " +
	//     "\"server_a_port\": [51, 6277], " +
	//     "\"server_a_protocol\": [51, \"OCPP\"]}"
	packet = make([]byte, len(jsonString)+1)

	packet[0] = 0x02
	copy(packet[1:], jsonString)
	return packet, 197
}
