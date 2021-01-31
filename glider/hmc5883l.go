package glider

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/mmr"
	"periph.io/x/periph/conn/physic"
	"time"
)

// Device bandwidth and output data rates
type Hmc5883LRate int

const (
	HMC5883L_RATE_0_75_HZ Hmc5883LRate = 0x00
	HMC5883L_RATE_1_5HZ                = 0x01
	HMC5883L_RATE_3_HZ                 = 0x02
	HMC5883L_RATE_7_5_HZ               = 0x03
	HMC5883L_RATE_15_HZ                = 0x04
	HMC5883L_RATE_30_HZ                = 0x05
	HMC5883L_RATE_75_HZ                = 0x06
)

// Measurement gain
type Hmc5883LGain int

const (
	HMC5883L_GAIN_0_88_GA Hmc5883LGain = 0b000 << 5
	HMC5883L_GAIN_1_3_GA               = 0b001 << 5
	HMC5883L_GAIN_1_9_GA               = 0b010 << 5
	HMC5883L_GAIN_2_5_GA               = 0b011 << 5
	HMC5883L_GAIN_4_0_GA               = 0b100 << 5
	HMC5883L_GAIN_4_7_GA               = 0b101 << 5
	HMC5883L_GAIN_5_6_GA               = 0b110 << 5
	HMC5883L_GAIN_8_1_GA               = 0b111 << 5
	// The physic package uses Tesla, and 1 Tesla = 10,000 Ga
	HMC5883L_GAIN_0_000088_T Hmc5883LGain = HMC5883L_GAIN_0_88_GA
	HMC5883L_GAIN_0_00013_T               = HMC5883L_GAIN_1_3_GA
	HMC5883L_GAIN_0_00019_T               = HMC5883L_GAIN_1_9_GA
	HMC5883L_GAIN_0_00025_T               = HMC5883L_GAIN_2_5_GA
	HMC5883L_GAIN_0_00040_T               = HMC5883L_GAIN_4_0_GA
	HMC5883L_GAIN_0_00047_T               = HMC5883L_GAIN_4_7_GA
	HMC5883L_GAIN_0_00056_T               = HMC5883L_GAIN_5_6_GA
	HMC5883L_GAIN_0_00081_T               = HMC5883L_GAIN_8_1_GA
)

// Measurement modes
type Hmc5883LMeasurementMode int

const (
	HMC5883L_MODE_CONTINUOUS Hmc5883LMeasurementMode = 0
	HMC5883L_MODE_SINGLE                             = 1
	HMC5883L_MODE_IDLE                               = 2
	// There's a second idle mode with value 3, but the docs don't
	// differentiate between 2 and 3
)

// Sample averaging modes
type Hmc5883LSampleAveraging int

const (
	HMC5883L_SAMPLES_1 Hmc5883LSampleAveraging = 0b00 << 5
	HMC5883L_SAMPLES_2                         = 0b01 << 5
	HMC5883L_SAMPLES_4                         = 0b10 << 5
	HMC5883L_SAMPLES_8                         = 0b11 << 5
)

type Hmc5883L struct {
	mmr  mmr.Dev8
	gain Hmc5883LGain
}

func NewHmc5883L(bus i2c.Bus) (*Hmc5883L, error) {
	device := &Hmc5883L{
		mmr: mmr.Dev8{
			Conn: &i2c.Dev{Bus: bus, Addr: uint16(HMC5883L_READ_ADDRESS)},
			// I don't think we ever access more than 1 byte at once, so
			// this is irrelevant
			Order: binary.BigEndian,
		},
		gain: HMC5883L_GAIN_0_00013_T,
	}

	chipId, err := device.mmr.ReadUint8(HMC5883L_IDENTIFICATION_REGISTER_A)
	if err != nil {
		return nil, err
	}
	if chipId != 0x48 {
		return nil, fmt.Errorf("No HMC5883L detected: %v", chipId)
	}

	chipId, err = device.mmr.ReadUint8(HMC5883L_IDENTIFICATION_REGISTER_B)
	if err != nil {
		return nil, err
	}
	if chipId != 0x34 {
		return nil, fmt.Errorf("No HMC5883L detected: %v", chipId)
	}

	chipId, err = device.mmr.ReadUint8(HMC5883L_IDENTIFICATION_REGISTER_C)
	if err != nil {
		return nil, err
	}
	if chipId != 0x33 {
		return nil, fmt.Errorf("No HMC5883L detected: %v", chipId)
	}

	err = device.SetRate(HMC5883L_RATE_15_HZ)
	if err != nil {
		return nil, err
	}
	err = device.SetRange(HMC5883L_GAIN_1_3_GA)
	if err != nil {
		return nil, err
	}
	err = device.SetMeasurementMode(HMC5883L_MODE_CONTINUOUS)
	if err != nil {
		return nil, err
	}
	err = device.SetSampleAveraging(HMC5883L_SAMPLES_2)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (a *Hmc5883L) SetRate(newRate Hmc5883LRate) error {
	value, err := a.mmr.ReadUint8(HMC5883L_CONFIGURATION_REGISTER_A)
	if err != nil {
		return err
	}
	value = value & 0b11100011
	value = value | uint8(newRate)
	err = a.mmr.WriteUint8(HMC5883L_CONFIGURATION_REGISTER_A, value)
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (a *Hmc5883L) SetRange(newGain Hmc5883LGain) error {
	// The upper 3 bits set the gain. The lower 5 bits must be cleared for
	// correct operation, so we can just clobber them.
	err := a.mmr.WriteUint8(HMC5883L_CONFIGURATION_REGISTER_B, uint8(newGain))
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	return nil
}

func (a *Hmc5883L) SetMeasurementMode(newMode Hmc5883LMeasurementMode) error {
	// The upper 6 bits should be written to 0
	err := a.mmr.WriteUint8(HMC5883L_MODE_REGISTER, uint8(newMode))
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)

	return nil
}

func (a *Hmc5883L) SetSampleAveraging(newSample Hmc5883LSampleAveraging) error {
	value, err := a.mmr.ReadUint8(HMC5883L_CONFIGURATION_REGISTER_A)
	if err != nil {
		return err
	}
	value = value & 0b10011111
	value = value | uint8(newSample)
	err = a.mmr.WriteUint8(HMC5883L_CONFIGURATION_REGISTER_A, value)
	if err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return nil
}

func (a *Hmc5883L) SenseRaw() (int16, int16, int16, error) {
	// I'm not sure if we can do full reads at once, so comment out for now
	var buffer [6]byte
	err := a.mmr.Conn.Tx([]byte{HMC5883L_DATA_OUTPUT_XMSB_REGISTER}, buffer[:])
	if err != nil {
		return 0, 0, 0, err
	}
	reader := bytes.NewReader(buffer[:])
	var x int16
	var y int16
	var z int16
	// The order is x, z, y! See the registers below
	err = binary.Read(reader, binary.BigEndian, &x)
	err = binary.Read(reader, binary.BigEndian, &z)
	err = binary.Read(reader, binary.BigEndian, &y)

	return x, y, z, nil
}

// This method is disabled because it's probably slower than the above
func (a *Hmc5883L) _SenseRaw() (int16, int16, int16, error) {
	xMsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_XMSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	xLsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_XLSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	yMsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_YMSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	yLsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_YLSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	zMsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_ZMSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	zLsb, err := a.mmr.ReadUint8(HMC5883L_DATA_OUTPUT_ZLSB_REGISTER)
	if err != nil {
		return 0, 0, 0, err
	}
	x := int16(xMsb)<<8 + int16(xLsb)
	y := int16(yMsb)<<8 + int16(yLsb)
	z := int16(zMsb)<<8 + int16(zLsb)

	return x, y, z, nil
}

func (a *Hmc5883L) Sense() (physic.MagneticFluxDensity, physic.MagneticFluxDensity, physic.MagneticFluxDensity, error) {
	xRaw, yRaw, zRaw, err := a.SenseRaw()
	if err != nil {
		return 0, 0, 0, err
	}

	multiplier := getGainMultiplier(a.gain)
	xValue := physic.MagneticFluxDensity(float64(xRaw) * multiplier)
	yValue := physic.MagneticFluxDensity(float64(yRaw) * multiplier)
	zValue := physic.MagneticFluxDensity(float64(zRaw) * multiplier)
	return xValue, yValue, zValue, nil
}

func getGainMultiplier(gain Hmc5883LGain) float64 {
	switch gain {
	case HMC5883L_GAIN_0_000088_T:
		return 0.073
	case HMC5883L_GAIN_0_00013_T:
		return 0.92
	case HMC5883L_GAIN_0_00019_T:
		return 1.22
	case HMC5883L_GAIN_0_00025_T:
		return 1.52
	case HMC5883L_GAIN_0_00040_T:
		return 2.27
	case HMC5883L_GAIN_0_00047_T:
		return 2.56
	case HMC5883L_GAIN_0_00056_T:
		return 3.03
	case HMC5883L_GAIN_0_00081_T:
		return 4.35
	}
	return 0.92
}

// HMC5883L registers
const (
	// Copied from the data sheet. Unused values are commented out.
	HMC5883L_CONFIGURATION_REGISTER_A  = 0
	HMC5883L_CONFIGURATION_REGISTER_B  = 1
	HMC5883L_MODE_REGISTER             = 2
	HMC5883L_DATA_OUTPUT_XMSB_REGISTER = 3
	HMC5883L_DATA_OUTPUT_XLSB_REGISTER = 4
	HMC5883L_DATA_OUTPUT_ZMSB_REGISTER = 5
	HMC5883L_DATA_OUTPUT_ZLSB_REGISTER = 6
	HMC5883L_DATA_OUTPUT_YMSB_REGISTER = 7
	HMC5883L_DATA_OUTPUT_YLSB_REGISTER = 8
	HMC5883L_STATUS_REGISTER           = 9
	HMC5883L_IDENTIFICATION_REGISTER_A = 10
	HMC5883L_IDENTIFICATION_REGISTER_B = 11
	HMC5883L_IDENTIFICATION_REGISTER_C = 12
)

const HMC5883L_READ_ADDRESS = 0x1E
