// Control for the ailerons
package glider

import (
	"errors"
	"github.com/stianeikeland/go-rpio/v4"
)

const LEFT_SERVO_PIN = 18  // BCM 18 = board 12
const RIGHT_SERVO_PIN = 13 // BCM 13 = board 33
const HERTZ = 50
const MULTIPLIER = 100000
const CENTER_MS = 1400
const NINETY_DEGREES_MS = 400

type Control struct {
	left  rpio.Pin
	right rpio.Pin
}

func NewControl() *Control {
	control := Control{
		left:  rpio.Pin(LEFT_SERVO_PIN),
		right: rpio.Pin(RIGHT_SERVO_PIN),
	}
	control.left.Mode(rpio.Pwm)
	control.right.Mode(rpio.Pwm)
	// Param freq should be in range 4688Hz - 19.2MHz to prevent
	// unexpected behavior
	control.left.Freq(HERTZ * MULTIPLIER)
	control.right.Freq(HERTZ * MULTIPLIER)
	return &control
}

func (control *Control) SetLeft(angle Degrees) error {
	return control.set(&control.left, angle)
}

func (control *Control) SetRight(angle Degrees) error {
	return control.set(&control.right, angle)
}

func (control *Control) set(pin *rpio.Pin, angle Degrees) error {
	// Output frequency is computed as pwm clock frequency divided by cycle length.
	// So, to set Pwm pin to freqency 38kHz with duty cycle 1/4, use this combination:
	//  pin.DutyCycle(1, 4)
	//  pin.Freq(38000*4)
	if angle < 30 || angle > 150 {
		return errors.New("Bad angle")
	}
	targetUs := uint32((float32(angle)-90.0)*NINETY_DEGREES_MS + CENTER_MS)
	a := getDutyCycleForUs(targetUs, HERTZ, MULTIPLIER)
	pin.DutyCycle(a, MULTIPLIER)
	return nil
}

func getDutyCycleForUs(targetUs, frequencyHz, multiplier uint32) uint32 {
	// We want a / multiplier = duty / cycle, where cycle = 1s / Hz
	// a / multiplier = duty / (1,000,000 us / 50 hz) = 20,000 us
	// a = duty / (1,000,000 us / 50 hz) * multiplier
	return uint32(float32(targetUs) / (float32(1000*1000) / float32(frequencyHz)) * float32(multiplier))
}
