package terraformer

import (
	"errors"
)

type terraACBTPacket struct {
	typeSend byte
	cmd      byte

	// 15 byte auth token
	token []byte
	data  []byte
}

func terraACBTPacketAssemble(data []byte) (packet *terraACBTPacket, err error) {
	pkt := new(terraACBTPacket)
	if data == nil {
		return nil, errors.New("no data")
	}
	if len(data) < 16 {
		return nil, errors.New("packet malformed?")
	}
	pkt.typeSend = data[0]
	pkt.cmd = data[0]

	pkt.token = data[8:15]
	pkt.data = data[16:]

	return pkt, nil
}

func terraWrap(cmd int, data []byte, token []byte) (body []byte, err error) {
	wArr := make([]byte, len(data)+16)

	cmdByte := cmd & 255
	if cmdByte == 0xBA || cmdByte == 0xD9 {
		wArr[0] = 0xAA
	} else {
		wArr[0] = 0xFE
	}

	wArr[1] = byte(cmdByte)
	wArr[2] = 0x0
	wArr[3] = 0x0

	if len(data) != 0 {
		wArr[4] = byte(len(data) & 255)
		wArr[5] = byte((len(data) >> 8) & 255)
	} else {
		wArr[4] = 0x0
		wArr[5] = 0x0
	}

	wArr[6] = 0x0

	b3 := byte(0)
	for i := 0; i < 7; i++ {
		b3 = b3 ^ wArr[i]
	}

	if len(token) == 0 {
		// default for null token is 8 bytes of 0
		token = make([]byte, 8)
	}

	if len(token) != 8 {
		return nil, errors.New("token is malformed")
	}

	b4 := b3
	tokenBytes := []byte(token)
	for _, b := range tokenBytes {
		b4 = b4 ^ b
	}
	b3 = b4

	if len(data) != 0 {
		b6 := b3
		for _, b := range data {
			b6 = b6 ^ b
		}
		b3 = b6
	}

	wArr[7] = b3

	// copy token in

	copy(wArr[8:], tokenBytes)

	// TODO copying misses the last byte, array not big enough maybe? -puck
	if len(data) != 0 {
		copy(wArr[16:], data)
	}

	return wArr, nil
}
