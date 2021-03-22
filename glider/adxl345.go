package glider

import (
	"encoding/binary"
	"fmt"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/mmr"
	"periph.io/x/periph/conn/physic"
	"time"
)

// Device bandwidth and output data rates
type Adxl345Rate int

const (
	ADXL345_RATE_1600HZ Adxl345Rate = 0x0F
	ADXL345_RATE_800HZ              = 0x0E
	ADXL345_RATE_400HZ              = 0x0D
	ADXL345_RATE_200HZ              = 0x0C
	ADXL345_RATE_100HZ              = 0x0B
	ADXL345_RATE_50HZ               = 0x0A
	ADXL345_RATE_25HZ               = 0x09
)

// Measurement Range
type Adxl345Range int

const (
	ADXL345_RANGE_2G  Adxl345Range = 0x00
	ADXL345_RANGE_4G               = 0x01
	ADXL345_RANGE_8G               = 0x02
	ADXL345_RANGE_16G              = 0x03
)

type Adxl345 struct {
	Mmr mmr.Dev8
}

func NewAdxl345(bus i2c.Bus) (*Adxl345, error) {
	device := &Adxl345{
		Mmr: mmr.Dev8{
			Conn: &i2c.Dev{Bus: bus, Addr: uint16(ADXL345_ADDRESS)},
			// I don't think we ever access more than 1 byte at once, so
			// this is irrelevant
			Order: binary.BigEndian,
		},
	}
	chipId, err := device.Mmr.ReadUint8(ADXL345_DEVID)
	if err != nil {
		return nil, err
	}
	if chipId != 0xE5 {
		return nil, fmt.Errorf("No ADXL345 detected: %v", chipId)
	}

	// Enable measurements
	device.Mmr.WriteUint8(ADXL345_POWER_CTL, 0b00001000)
	time.Sleep(20 * time.Millisecond)

	err = device.SetRate(ADXL345_RATE_25HZ)
	if err != nil {
		return nil, err
	}

	/*
		err = device.SetRange(ADXL345_RANGE_2G)
		if err != nil {
			return nil, err
		}
	*/

	return device, nil
}

func (a *Adxl345) SetRate(newRate Adxl345Rate) error {
	err := a.Mmr.WriteUint8(ADXL345_BW_RATE, uint8(newRate))
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (a *Adxl345) SetRange(newRange Adxl345Range) error {
	format, err := a.Mmr.ReadUint8(ADXL345_DATA_FORMAT)
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	// The lower 2 bits set the range
	format = format & 0b11111100
	format = format | uint8(newRange)
	err = a.Mmr.WriteUint8(ADXL345_DATA_FORMAT, format)
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	return nil
}

func (a *Adxl345) SenseRaw() (int16, int16, int16, error) {
	var buffer [6]byte
	err := a.Mmr.Conn.Tx([]byte{ADXL345_DATAX0}, buffer[:])
	if err != nil {
		return 0, 0, 0, err
	}
	x := int16(buffer[0]) | (int16(buffer[1]) << 8)
	y := int16(buffer[2]) | (int16(buffer[3]) << 8)
	z := int16(buffer[4]) | (int16(buffer[5]) << 8)
	return x, y, z, nil
}

func (a *Adxl345) Sense() (physic.Speed, physic.Speed, physic.Speed, error) {
	const multiplier = 0.0039
	const gravity_ms2 = 9.80665
	const adjustment = physic.Speed(multiplier * gravity_ms2 * float64(physic.MetrePerSecond))
	xRaw, yRaw, zRaw, err := a.SenseRaw()
	if err != nil {
		return 0, 0, 0, err
	}

	xValue := physic.Speed(float64(xRaw)) * adjustment
	yValue := physic.Speed(float64(yRaw)) * adjustment
	zValue := physic.Speed(float64(zRaw)) * adjustment
	return xValue, yValue, zValue, err
}

// ADXL345 registers
const (
	// Copied from the data sheet. Unused values are commented out.
	ADXL345_DEVID = 0x00 // Device ID.
	// 0x01 to 0x01C are reserved. Do not access.
	//ADXL345_THRESH_TAP = 0x1D // Tap threshold.
	//ADXL345_OFSX = 0x1E  // X-axis offset.
	//ADXL345_OFSY = 0x1F  // Y-axis offset.
	//ADXL345_OFSZ = 0x20  // Z-axis offset.
	//ADXL345_DUR = 0x21  // Tap duration.
	//ADXL345_Latent = 0x22  // Tap latency.
	//ADXL345_Window = 0x23  // Tap window.
	//ADXL345_THRESH_ACT = 0x24  // Activity threshold.
	//ADXL345_THRESH_INACT = 0x25  // Inactivity threshold.
	//ADXL345_TIME_INACT = 0x26  // Inactivity time.
	//ADXL345_ACT_INACT_CTL = 0x27  // Axis enable control for activity and inactivity detection.
	//ADXL345_THRESH_FF = 0x28  // Free-fall threshold.
	//ADXL345_TIME_FF = 0x29  // Free-fall time.
	//ADXL345_TAP_AXES = 0x2A  // Axis control for tap/double tap.
	//ADXL345_ACT_TAP_STATUS = 0x2B  // Source of tap/double tap.
	ADXL345_BW_RATE   = 0x2C // Data rate and power mode control.
	ADXL345_POWER_CTL = 0x2D // Power-saving features control.
	//ADXL345_INT_ENABLE = 0x2E  // Interrupt enable control.
	//ADXL345_INT_MAP = 0x2F  // Interrupt mapping control.
	//ADXL345_INT_SOURCE = 0x30  // Source of interrupts.
	ADXL345_DATA_FORMAT = 0x31 // Data format control.
	ADXL345_DATAX0      = 0x32 // X-Axis Data 0.
	ADXL345_DATAX1      = 0x33 // X-Axis Data 1.
	ADXL345_DATAY0      = 0x34 // Y-Axis Data 0.
	ADXL345_DATAY1      = 0x35 // Y-Axis Data 1.
	ADXL345_DATAZ0      = 0x36 // Z-Axis Data 0.
	ADXL345_DATAZ1      = 0x37 // Z-Axis Data 1.
	//ADXL345_FIFO_CTL = 0x38  // FIFO control.
	//ADXL345_FIFO_STATUS = 0x39  // FIFO status
)

const ADXL345_ADDRESS = 0x53

// The typical scale factor in g/LSB
const scaleMultiplier = 0.0039
