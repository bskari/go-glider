// Control for the ailerons
package glider

import (
	"errors"
	"github.com/stianeikeland/go-rpio/v4"
)

const HERTZ = 50

// Originally I had MULTIPLIER set to 100_000 but then my cycle wouldn't come
// out as 20 us, instead it was around 16? Checked with an oscilloscope. Maybe
// some overflow.
const MULTIPLIER = 20000
const US_PER_CYCLE = (1000 * 1000) / HERTZ
const US_PER_DEGREE = 800 / 90

type Control struct {
	left         *rpio.Pin
	right        *rpio.Pin
	leftZero_us  float32
	rightZero_us float32
}

func NewControl() *Control {
	tempLeft := rpio.Pin(configuration.LeftServoPin)
	tempRight := rpio.Pin(configuration.RightServoPin)
	tempLeft.Pwm()
	tempRight.Pwm()
	tempLeft.Freq(HERTZ * MULTIPLIER)
	tempRight.Freq(HERTZ * MULTIPLIER)
	tempLeft.DutyCycle(1000, MULTIPLIER)
	control := Control{
		left:         &tempLeft,
		right:        &tempRight,
		leftZero_us:  float32(configuration.LeftServoCenter_us - US_PER_DEGREE*90),
		rightZero_us: float32(configuration.RightServoCenter_us - US_PER_DEGREE*90),
	}
	// Param freq should be in range 4688Hz - 19.2MHz to prevent
	// unexpected behavior
	return &control
}

func (control *Control) SetLeft(angle_r Radians) error {
	return control.set(control.left, angle_r, control.leftZero_us)
}

func (control *Control) SetRight(angle_r Radians) error {
	return control.set(control.right, angle_r, control.rightZero_us)
}

func (control *Control) set(pin *rpio.Pin, angle_r Radians, offset float32) error {
	// Output frequency is computed as pwm clock frequency divided by cycle length.
	// So, to set Pwm pin to freqency 38kHz with duty cycle 1/4, use this combination:
	//  pin.DutyCycle(1, 4)
	//  pin.Freq(38000*4)
	if angle_r < ToRadians(45) || angle_r > ToRadians(135) {
		return errors.New("Bad angle")
	}
	target_us := uint32(ToDegrees(angle_r)*US_PER_DEGREE + offset)
	a := getDutyCycleForUs(target_us)
	pin.DutyCycle(a, MULTIPLIER)
	return nil
}

func getDutyCycleForUs(target_us uint32) uint32 {
	return target_us * MULTIPLIER / US_PER_CYCLE
}
