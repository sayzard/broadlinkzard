package broadlinkzard

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// BroadlinkDeviceInterface : Common Interface
type BroadlinkDeviceInterface interface {
	initVars()
	Close()
	GetDevice() *BroadlinkDevice
	Auth() (bool, error)
	SetLogLevel(int)

	SetPower(bool) (bool, error)
	QueryPower() (bool, error)

	SetPowerMask(uint8, bool) (bool, error)
	SetPowerMulti(int, bool) (bool, error)
	QueryPowerRaw() (uint, error)
}

// BroadlinkDevice : Device Structure
type BroadlinkDevice struct {
	DevID     uint32
	DevType   uint16
	IPAddr    *net.UDPAddr
	HwMac     [6]byte
	bcastAddr *net.UDPAddr

	encKey []byte
	encIv  []byte

	CS        *net.UDPConn
	SendCount uint16

	TimeoutDefault time.Duration
	LogLevel       int

	responses chan ([]byte)

	BroadlinkDeviceInterface
}

// BroadlinkDeviceSp2 : Device Structure for SP2 or similar
type BroadlinkDeviceSp2 struct {
	BroadlinkDevice
}

// BroadlinkDeviceMp1 : Device Structure for MP1 or similar
type BroadlinkDeviceMp1 struct {
	BroadlinkDevice
}

// NewBroadlinkDirectDevice : Create Device Structure with Detail Information
func NewBroadlinkDirectDevice(iType uint16, sIP string, sMac string) BroadlinkDeviceInterface {

	sAddr, err := net.ResolveUDPAddr("udp", sIP+":80")
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	hwMac, err := net.ParseMAC(sMac)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	switch iType {
	case
		0x2711,                         // SP2
		0x2719, 0x7919, 0x271a, 0x791a, // Honeywell SP2
		0x2720,         // SPMini
		0x753e,         // SP3
		0x7D00,         // OEM branded SP3
		0x947a, 0x9479, // SP3S
		0x2728,         // SPMini2
		0x2733, 0x273e, // OEM branded SPMini
		0x7530, 0x7546, 0x7918, // OEM branded SPMini2
		0x7D0D, // TMall OEM SPMini3
		0x2736: //SPMiniPlus
		var devsp2 BroadlinkDeviceSp2
		devsp2.DevType = iType
		devsp2.IPAddr = sAddr
		copy(devsp2.HwMac[:], hwMac[:6])
		devsp2.initVars()
		return &devsp2
	case
		0x4EB5,
		0x4EF7:
		var devmp1 BroadlinkDeviceMp1
		devmp1.DevType = iType
		devmp1.IPAddr = sAddr
		copy(devmp1.HwMac[:], hwMac[:6])
		devmp1.initVars()
		return &devmp1
	}
	var dev BroadlinkDevice
	dev.initVars()
	return &dev
}

func (dev *BroadlinkDevice) initVars() {
	dev.encKey = []byte{0x09, 0x76, 0x28, 0x34, 0x3f, 0xe9, 0x9e, 0x23, 0x76, 0x5c, 0x15, 0x13, 0xac, 0xcf, 0x8b, 0x02}
	dev.encIv = []byte{0x56, 0x2e, 0x17, 0x99, 0x6d, 0x09, 0x3d, 0x28, 0xdd, 0xb3, 0xba, 0x69, 0x5a, 0x2e, 0x6f, 0x58}
	dev.SendCount = 0

	dev.TimeoutDefault = time.Duration(10)
	dev.LogLevel = 1
	dev.CS, _ = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	dev.bcastAddr, _ = net.ResolveUDPAddr("udp4", "255.255.255.255:80")
	dev.responses = make(chan []byte, 1000)
	go dev.udpListener()
}

// SetLogLevel : Set Loglevel
func (dev *BroadlinkDevice) SetLogLevel(level int) {
	dev.LogLevel = level
}

// LogMessage : Show Message
func (dev *BroadlinkDevice) LogMessage(level int, message string) {
	if dev.LogLevel >= level {
		fmt.Print(message)
	}
}

// Close : Close
func (dev *BroadlinkDevice) Close() {
	if dev.CS != nil {
		dev.LogMessage(10, "Close\n")
		dev.CS.Close()
		dev.CS = nil
	}
}

func (dev *BroadlinkDevice) udpListener() {
	for {
		buf := make([]byte, 2048)
		count, _, err := dev.CS.ReadFrom(buf)

		if err != nil {
			if count == 0 {
				break
			}
			continue
		}

		if checkChecksum(buf, 0x20) {
			response := make([]byte, count)
			copy(response, buf)
			dev.responses <- response
		}
	}
}

// GetDevice : Get Device Structure
func (dev *BroadlinkDevice) GetDevice() *BroadlinkDevice {
	return dev
}

// SetPower : Dummy Power Control
func (dev *BroadlinkDevice) SetPower(bool) (bool, error) {
	return false, errors.New("Not supported")
}

// QueryPower : Dummy Query Power
func (dev *BroadlinkDevice) QueryPower() (bool, error) {
	return false, errors.New("Not supported")
}

// SetPowerMask : Dummy Power Control
func (dev *BroadlinkDevice) SetPowerMask(uint8, bool) (bool, error) {
	return false, errors.New("Not supported")
}

// SetPowerMulti : Dummy Power Control
func (dev *BroadlinkDevice) SetPowerMulti(int, bool) (bool, error) {
	return false, errors.New("Not supported")
}

// QueryPowerRaw : Dummy Query Power
func (dev *BroadlinkDevice) QueryPowerRaw() (uint, error) {
	return 0, errors.New("Not supported")
}

//------------

// SetPower : On/Off Power Control
func (dev *BroadlinkDeviceSp2) SetPower(onstate bool) (bool, error) {
	payload := make([]byte, 16)
	payload[0] = 2
	if onstate {
		payload[4] = 1
	} else {
		payload[4] = 0
	}
	dev.LogMessage(5, fmt.Sprintln("PAYLOAD=", hex.Dump(payload)))
	_, err := dev.SendPacket(0x6a, payload)
	if err != nil {
		return false, err
	}
	return true, nil
}

// QueryPower : Query Power
func (dev *BroadlinkDeviceSp2) QueryPower() (bool, error) {
	payload := make([]byte, 16)
	payload[0] = 1
	dev.LogMessage(5, fmt.Sprintln("PAYLOAD=", hex.Dump(payload)))
	resp, err := dev.SendPacket(0x6a, payload)
	if err != nil {
		return false, err
	}
	ierr := binary.LittleEndian.Uint16(resp[0x22:])
	dev.LogMessage(10, fmt.Sprintln("RESP=", resp[0x22], resp[0x23], ierr))
	if ierr != 0 {
		return false, fmt.Errorf("Response %x", ierr)
	}
	dpayload, err := decrypt(dev.encKey, dev.encIv, resp[0x38:])
	if err != nil {
		return false, err
	}
	if dpayload[0x4] == 1 || dpayload[4] == 3 || dpayload[4] == 0xfd {
		return true, nil
	}
	return false, nil
}

// SetPowerMask : On/Off Power Control
func (dev *BroadlinkDeviceMp1) SetPowerMask(smask uint8, onstate bool) (bool, error) {

	dev.LogMessage(5, fmt.Sprintln("POWERMASK=", smask))
	vb2 := (smask << 1)
	if !onstate {
		vb2 = smask
	}
	vb2 += 0xb2
	v0e := smask
	if !onstate {
		v0e = 0
	}
	payload := make([]byte, 16)
	copy(payload[0:], []byte{0x0d, 0x00, 0xa5, 0xa5, 0x5a, 0x5a, vb2, 0xc0, 0x02, 0x00, 0x03, 0x00, 0x00, smask, v0e})
	dev.LogMessage(10, fmt.Sprintln("PAYLOAD=", hex.Dump(payload)))
	_, err := dev.SendPacket(0x6a, payload)
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetPowerMulti : On/Off Power Control
func (dev *BroadlinkDeviceMp1) SetPowerMulti(no int, onstate bool) (bool, error) {
	var smask uint8
	smask = 0x01 << (uint(no) - 1)
	return dev.SetPowerMask(smask, onstate)
}

// QueryPowerRaw : Query Power
func (dev *BroadlinkDeviceMp1) QueryPowerRaw() (uint, error) {
	payload := make([]byte, 16)
	copy(payload[0:], []byte{0x0a, 0x00, 0xa5, 0xa5, 0x5a, 0x5a, 0xae, 0xc0, 0x01})
	dev.LogMessage(5, fmt.Sprintln("PAYLOAD=", hex.Dump(payload)))
	resp, err := dev.SendPacket(0x6a, payload)
	if err != nil {
		return 0, err
	}
	ierr := binary.LittleEndian.Uint16(resp[0x22:])
	dev.LogMessage(10, fmt.Sprintln("RESP=", resp[0x22], resp[0x23], ierr))
	if ierr != 0 {
		return 0, fmt.Errorf("Response %x", ierr)
	}	
	dpayload, err := decrypt(dev.encKey, dev.encIv, resp[0x38:])
	if err != nil {
		return 0, err
	}	
	return uint(dpayload[0x0e]), nil
}

//----
func padding(packet []byte, blockSize int) []byte {
	return append(packet, bytes.Repeat([]byte{0x00}, blockSize-len(packet)%blockSize)...)
}

func unPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func calcChecksum(payload []byte) uint16 {
	checksum := uint16(0xbeaf)

	for _, val := range payload {
		checksum += uint16(val)
	}

	return checksum
}

func checkChecksum(payload []byte, checksumPos int) bool {
	origChecksum := binary.LittleEndian.Uint16(payload[checksumPos : checksumPos+2])
	binary.LittleEndian.PutUint16(payload[checksumPos:checksumPos+2], 0)

	newChecksum := calcChecksum(payload)

	binary.LittleEndian.PutUint16(payload[checksumPos:checksumPos+2], origChecksum)

	return newChecksum == origChecksum
}

func encrypt(key, iv, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := padding(text, aes.BlockSize)
	ciphertext := make([]byte, len(b))

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, b)

	return ciphertext, nil
}

func decrypt(key []byte, iv []byte, encText []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(encText) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	decrypted := make([]byte, len(encText))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, encText)

	return decrypted, nil
}

// -------

// RecvResponse : Recv Response
func (dev *BroadlinkDevice) RecvResponse(timeout time.Duration) ([]byte, error) {
	for {
		select {
		case buf := <-dev.responses:
			return buf, nil

		case <-time.After(timeout * time.Second):
			return nil, errors.New("Timeout")
		}
	}
}

func (dev *BroadlinkDevice) wait4Response(expectedType uint16, timeout time.Duration) ([]byte, error) {
	startTime := time.Now().Add(timeout * time.Second)
	for {
		select {
		case buf := <-dev.responses:
			msgType := binary.LittleEndian.Uint16(buf[0x26:0x28])
			if msgType == expectedType {
				return buf, nil
			}

			dev.responses <- buf
			if !time.Now().Before(startTime) {
				return nil, errors.New("Check time")
			}
		case <-time.After(timeout * time.Second):
			return nil, errors.New("Timeout")
		}
	}
}

// RawSendPacket : Send Packet (W/O Response)
func (dev *BroadlinkDevice) RawSendPacket(command uint16, payload []byte) (bool, error) {
	dev.SendCount++

	packet := make([]byte, 0x38)
	copy(packet[0:], []byte{0x5a, 0xa5, 0xaa, 0x55, 0x5a, 0xa5, 0xaa, 0x55, 0x00})
	binary.LittleEndian.PutUint16(packet[0x24:], dev.DevType)
	binary.LittleEndian.PutUint16(packet[0x26:], command)
	binary.LittleEndian.PutUint16(packet[0x28:], dev.SendCount)
	copy(packet[0x2a:], dev.HwMac[0:])
	binary.LittleEndian.PutUint32(packet[0x30:], dev.DevID)

	if (payload != nil) && (len(payload) > 0) {
		payload = padding(payload, aes.BlockSize)
		binary.LittleEndian.PutUint16(packet[0x34:], calcChecksum(payload))
		encrypted, _ := encrypt(dev.encKey, dev.encIv, payload)
		packet = append(packet, encrypted...)
	}

	binary.LittleEndian.PutUint16(packet[0x20:], calcChecksum(packet))

	dev.LogMessage(20, fmt.Sprintln(hex.Dump(payload)))
	dev.LogMessage(20, fmt.Sprintln(hex.Dump(packet)))

	dev.CS.WriteToUDP(packet, dev.IPAddr)
	return true, nil
}

// SendPacket : Send Packet
func (dev *BroadlinkDevice) SendPacket(command uint16, payload []byte) ([]byte, error) {
	_, err := dev.RawSendPacket(command, payload)
	if err != nil {
		return nil, err
	}
	dev.LogMessage(10, fmt.Sprintln("WAIT FOR RESP"))
	resp, err := dev.RecvResponse(time.Duration(1))
	dev.LogMessage(10, fmt.Sprintln("GOT RESP", err))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Auth : Authenticate
func (dev *BroadlinkDevice) Auth() (bool, error) {
	payload := make([]byte, 0x50)
	payload[0x2d] = 0x01

	hostname, _ := os.Hostname()
	copy(payload[0x30:], []byte(hostname))

	dev.RawSendPacket(0x65, payload)

	resp, err := dev.wait4Response(0x3e9, dev.TimeoutDefault)
	if err != nil {
		return false, err
	}

	if len(resp) >= 0x38 {
		dev.LogMessage(10, fmt.Sprintln("RESPONSE"))
		dev.LogMessage(10, fmt.Sprintln(hex.Dump(resp)))
		decrypted, _ := decrypt(dev.encKey, dev.encIv, resp[0x38:])
		dev.DevID = binary.LittleEndian.Uint32(decrypted[0x00:])
		dev.encKey = decrypted[0x04:0x14]
		dev.LogMessage(2, fmt.Sprintln("Device Id=", dev.DevID))
	}
	return true, nil
}
