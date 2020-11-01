// Control for the ailerons
package glider

import (
	"errors"
	"github.com/stianeikeland/go-rpio/v4"
)

const LEFT_SERVO_PIN = 12  // BCM 12 = board 32
const RIGHT_SERVO_PIN = 13 // BCM 13 = board 33
const HERTZ = 50

// Originally I had MULTIPLIER set to 100_000 but then my cycle wouldn't come
// out as 20 us, instead it was around 16? Checked with an oscilloscope. Maybe
// some overflow.
const MULTIPLIER = 20000
const US_PER_CYCLE = (1000 * 1000) / HERTZ
const US_PER_DEGREE = 800 / 90
const ZERO_US = 1430 - US_PER_DEGREE*90

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
	if angle < 45 || angle > 135 {
		return errors.New("Bad angle")
	}
	targetUs := uint32(angle*US_PER_DEGREE + ZERO_US)
	a := getDutyCycleForUs(targetUs)
	pin.DutyCycle(a, MULTIPLIER)
	return nil
}

func getDutyCycleForUs(targetUs uint32) uint32 {
	return targetUs * MULTIPLIER / US_PER_CYCLE
}
