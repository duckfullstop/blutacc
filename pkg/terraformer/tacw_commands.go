package terraformer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/duckfullstop/blutacc/pkg/tripledesECB"
	"log"
	"strconv"
	"strings"
	"time"
)

const terraEncryptKey string = "ucserver"

type TerraStatus struct {
	// 0x00 "In the free"
	// 0x01 "The gun is not charged"
	// 0x02 "Wait charging"
	// 0x06 "In the charging"
	// 0x08 "Undrawn gun after charging"
	// 0x0F "In the fault"
	status byte

	// Supplied only if charger is in fault - NOT IMPLEMENTED, DO NOT READ THESE
	FaultCode byte
	Fault     string

	// Following are all supplied if charger is status 0x06
	PhaseLineType       int
	ChargeOrderSequence int
	// Charge delivered (in KWh)
	ChargeElectricity float32

	// First phase
	ChargeVoltageP1 float32 // In Volts
	ChargeCurrentP1 float32 // In Amps

	// Second phase (if present)
	ChargeVoltageP2 float32
	ChargeCurrentP2 float32

	// Third phase (if present)
	ChargeVoltageP3 float32
	ChargeCurrentP3 float32

	// Duration of the active charge session (in seconds)
	ChargeDuration int
	// Current this session is allowed to pull (in Amps)
	RatedCurrent int
}

func (cc *TerraStatus) readFromBytes(data []byte) {
	var ptr int
	// read status
	cc.status = data[ptr]
	ptr++
	if cc.status == byte(0x06) {
		// charging, get extra data
		if ptr < len(data) {
			cc.PhaseLineType = int(data[ptr])
			ptr++
		}
		if ptr+4 < len(data) {
			cc.ChargeOrderSequence = int(binary.LittleEndian.Uint32(data[ptr : ptr+4]))
			ptr = ptr + 4
		}
		if ptr+2 < len(data) {
			cc.ChargeElectricity = float32(binary.LittleEndian.Uint16(data[ptr:ptr+2])) / 100
			ptr = ptr + 2
		}
		if ptr+4 < len(data) {
			cc.ChargeVoltageP1 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
			ptr = ptr + 4
		}
		if ptr+4 < len(data) {
			cc.ChargeCurrentP1 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
			ptr = ptr + 4
		}
		if cc.PhaseLineType != 0 {
			// Multiple phases
			if ptr+4 < len(data) {
				cc.ChargeVoltageP2 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
				ptr = ptr + 4
			}
			if ptr+4 < len(data) {
				cc.ChargeCurrentP2 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
				ptr = ptr + 4
			}
			if ptr+4 < len(data) {
				cc.ChargeVoltageP3 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
				ptr = ptr + 4
			}
			if ptr+4 < len(data) {
				cc.ChargeCurrentP3 = float32(binary.LittleEndian.Uint32(data[ptr:ptr+4])) / 100
				ptr = ptr + 4
			}
		}
		if ptr+4 < len(data) {
			cc.ChargeDuration = int(binary.LittleEndian.Uint32(data[ptr : ptr+4]))
			ptr = ptr + 4
		}
		if ptr < len(data) {
			cc.RatedCurrent = int(data[ptr])
			ptr++
		}
	}
}

type TerraCapacityConfig struct {
	// Bit1 through bit5 are unused as far as we can ascertain
	bit1 bool
	bit2 bool
	bit3 bool
	bit4 bool
	bit5 bool

	// FreeVend sets the CP to charge immediately, regardless of schedule or RFID authentication
	FreeVend bool
	// ExternalCards permits the CP to register cards that are not ABB-certified (anything the reader can read)
	ExternalCards bool
	// ExternalAccess sets the CP to use the registered OCPP server
	ExternalAccess bool
}

func (cc *TerraCapacityConfig) readFromByte(data byte) {
	cc.bit1 = data&128 == 1
	cc.bit2 = data&64 == 1
	cc.bit3 = data&32 == 1
	cc.bit4 = data&16 == 1
	cc.bit5 = data&8 == 1
	cc.FreeVend = data&4 == 1
	cc.ExternalCards = data&2 == 1
	cc.ExternalAccess = data&1 == 1
}

func (cc *TerraCapacityConfig) getByte() (data byte) {
	// really stupid function and there's better ways to do bitmaps but I hate them with a passion
	var n int64
	if cc.bit1 {
		n |= (1 << 0)
	}
	if cc.bit2 {
		n |= (1 << 1)
	}
	if cc.bit3 {
		n |= (1 << 2)
	}
	if cc.bit4 {
		n |= (1 << 3)
	}
	if cc.bit5 {
		n |= (1 << 4)
	}
	if cc.FreeVend {
		n |= (1 << 5)
	}
	if cc.ExternalCards {
		n |= (1 << 6)
	}
	if cc.ExternalAccess {
		n |= (1 << 7)
	}
	return []byte(strconv.FormatInt(n, 16))[0]
}

func (wb *TerraACWallbox) cmdAuthenticate() (data []byte, err error) {
	// auth packet thoughts
	// in java call (terraconfig android):
	// str is chargerSN
	// str2 is "ucserver" because that's very secure
	// mUserId in func is the ChargerSync / TerraConfig AUTO_INCREMENT user ID (e.g 12699)

	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	// per buildIdentityAuthenticationRequestBody
	rArr := make([]byte, 48)
	// remove dashes from serial because they are shown in some literature
	copy(rArr[0:], strings.Replace(wb.SerialNumber, "-", "", -1))

	// everything from the end of the serial to byte 20 should be 0

	// wonder if these packets set authorization level?
	// on TerraConfig iOS packet dump: [00 02]
	// on TerraConfig Android disassembly: [02 01]
	rArr[20] = 2
	rArr[21] = 1

	// this can be literally anything you want, there's no authentication
	const terraUserID int = 1337

	copy(rArr[22:], fmt.Sprint(terraUserID))

	// everything from the end of the User ID to byte 48 should be 0

	// this data now gets encrypted in an extraordinarily secure fashion
	eData, err := tripledesECB.TripleEcbDesEncrypt(rArr, []byte(terraEncryptKey))

	if err != nil {
		return nil, err
	}

	// some magic
	pArr := make([]byte, 130)
	pArr[0] = 128
	pArr[1] = 128
	copy(pArr[2:], eData)

	// wb.token is unpopulated when this command fires, but terraWrap will handle that
	request, err := terraWrap(254, pArr, wb.token)
	if err != nil {
		return nil, err
	}

	err = wb.write(request)
	if err != nil {
		return nil, err
	}

	rtn, err := wb.read(5 * time.Second)
	if err != nil {
		return nil, err
	}

	if len(rtn.data) == 0 {
		return nil, errors.New("likely patched - empty auth response")
	}
	if len(rtn.data) < 34 {
		return nil, errors.New("auth response not long enough")
	}

	// token is 8 bytes at the end of the data
	// more data can be pulled from the response but it's not particularly interesting
	// private boolean authSuccess = true;
	// private String communicationVersion;
	// private String hardwareVersion = "";
	// private int softwareVersion;
	// private int startCode = 255;
	// private String token;

	return rtn.data[26:34], nil
}

func (wb *TerraACWallbox) CmdRequestWifiInfo() (data []byte, err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x01}
	request, err := terraWrap(199, pArr, wb.token)
	if err != nil {
		return nil, err
	}

	err = wb.write(request)
	if err != nil {
		return nil, err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return nil, err
	}

	return rtn.data, nil

}

func (wb *TerraACWallbox) CmdRequestLog() (data []byte, err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x00, 0x00, 0x00, 0x00, 0x0A}
	request, err := terraWrap(189, pArr, wb.token)
	if err != nil {
		return nil, err
	}

	err = wb.write(request)
	if err != nil {
		return nil, err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return nil, err
	}

	return rtn.data, nil
}

func (wb *TerraACWallbox) CmdSetCapacities(capacities TerraCapacityConfig) (err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	// pArr := []byte{capacities.getByte(), 0x00}
	pArr := []byte{0x03, 0x00}

	request, err := terraWrap(220, pArr, wb.token)
	if err != nil {
		return err
	}

	err = wb.write(request)
	if err != nil {
		return err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return err
	}

	switch rtn.data[0] {
	case 0x00:
		return nil
	default:
		return errors.New(fmt.Sprintf("tacw reported failure %x", rtn.data[0]))
	}
}

func (wb *TerraACWallbox) CmdCheckCapacities() (capConfig TerraCapacityConfig, err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{}
	request, err := terraWrap(221, pArr, wb.token)
	if err != nil {
		return capConfig, err
	}

	err = wb.write(request)
	if err != nil {
		return capConfig, err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return capConfig, err
	}

	capConfig.readFromByte(rtn.data[0])

	log.Printf("Caps: 0x%x / %v received, calculate %v", rtn.data[0], rtn.data[0], capConfig)
	return capConfig, nil
}

func (wb *TerraACWallbox) CmdStartCharge() (err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x00, 0x00}
	request, err := terraWrap(180, pArr, wb.token)
	if err != nil {
		return err
	}

	err = wb.write(request)
	if err != nil {
		return err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return err
	}

	switch rtn.data[0] {
	case 0x00:
		return nil
	default:
		return errors.New(fmt.Sprintf("tacw reported failure %x", rtn.data[0]))
	}
}

func (wb *TerraACWallbox) CmdStopCharge() (err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x00}
	request, err := terraWrap(182, pArr, wb.token)
	if err != nil {
		return err
	}

	err = wb.write(request)
	if err != nil {
		return err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return err
	}

	switch rtn.data[0] {
	case 0x00:
		return nil
	default:
		return errors.New(fmt.Sprintf("tacw reported failure %x", rtn.data[0]))
	}
}

func (wb *TerraACWallbox) CmdRequestStatus() (data []byte, err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x00}
	request, err := terraWrap(181, pArr, wb.token)
	if err != nil {
		return nil, err
	}

	err = wb.write(request)
	if err != nil {
		return nil, err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return nil, err
	}

	// not plugged: 0001
	// plugged: 0301
	// charging: 06009c31746101003c5a00000e0c000000002001

	status := TerraStatus{}
	status.readFromBytes(rtn.data)
	return rtn.data, nil
}

func (wb *TerraACWallbox) CmdRequestOcppConfig() (data []byte, err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	pArr := []byte{0x01}
	request, err := terraWrap(197, pArr, wb.token)
	if err != nil {
		return nil, err
	}

	err = wb.write(request)
	if err != nil {
		return nil, err
	}

	rtn, err := wb.read(1 * time.Second)
	if err != nil {
		return nil, err
	}

	return rtn.data, nil
}

func (wb *TerraACWallbox) CmdWriteOcppConfig(conf string) (err error) {
	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	// 0x02 to write
	pArr := make([]byte, len(conf)+1)
	pArr[0] = 0x02
	copy(pArr[1:], conf)

	request, err := terraWrap(197, pArr, wb.token)
	if err != nil {
		return err
	}

	err = wb.write(request)
	if err != nil {
		return err
	}

	data, err := wb.read(5 * time.Second)
	if err != nil {
		return err
	}

	switch data.data[0] {
	case 0x00:
		return errors.New("tacw reported failure")
	case 0x01:
		return nil
	default:
		return errors.New("tacw reported unknown result code")
	}
}

func (wb *TerraACWallbox) CmdWriteOcppData() (err error) {
	// this is all stubbed and not working, don't use yet
	cServerEnable := 1
	cDomainUrl := "http://ocpp/ocpp"
	var cPort uint16 = 6277
	cProtocolType := 1
	cProtocolVersion := "ocpp16j"
	cSecurityKey := "00"
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

	// take the mutex until we have all the data we need
	wb.btMtx.Lock()
	defer wb.btMtx.Unlock()

	request, err := terraWrap(214, pArr, wb.token)
	if err != nil {
		return err
	}

	err = wb.write(request)
	if err != nil {
		return err
	}

	data, err := wb.read(5 * time.Second)
	if err != nil {
		return err
	}

	switch data.data[0] {
	case 0x00:
		return errors.New("tacw reported failure")
	case 0x01:
		return nil
	case 0x02:
		return errors.New("tacw reported security identify failure")
	default:
		return errors.New("tacw reported unknown result code")
	}
}
