package glider

import (
	"github.com/stianeikeland/go-rpio/v4"
	"io"
	"math"
	"time"
)

type PilotState uint8

// These are defined in reverse order, so that blinking the status makes
// more sense. Fewer blink patterns indicate we're done initializing.
const (
	flying PilotState = iota + 1
	waitingForLaunch
	waitingForButton
	initializing
	landed
	testMode
)

func (ps PilotState) String() string {
	return []string{
		"(unused-0-state)",
		"flying",
		"waitingForLaunch",
		"waitingForButton",
		"initializing",
		"landed",
		"testMode",
	}[ps]
}

type Pilot struct {
	state                 PilotState
	telemetry             *Telemetry
	control               *Control
	statusIndicator       *LedStatusIndicator
	buttonPin             *rpio.Pin
	buttonPressTime       time.Time
	zeroSpeedTime         *time.Time
	waypoints             *Waypoints
	previousUpdateRoll_r  Radians 
	previousUpdatePitch_r Radians
}

func NewPilot() (*Pilot, error) {
	telemetry, err := NewTelemetry()
	if err != nil {
		return nil, err
	}

	response := rpio.Pin(configuration.ButtonPin)
	buttonPin := &response
	buttonPin.Input()
	buttonPin.PullUp()

	return &Pilot{
		// TODO
		//state:           initializing,
		state:           testMode,
		control:         NewControl(),
		telemetry:       telemetry,
		statusIndicator: NewLedStatusIndicator(uint8(initializing)),
		buttonPin:       buttonPin,
		buttonPressTime: time.Now(),
		zeroSpeedTime:   nil,
		waypoints:       NewWaypoints(),
	}, nil
}

// Run the local glide test, e.g. when throwing the plane down a hill
func (pilot *Pilot) RunGlideTestForever() {
	previousState := pilot.state
	Logger.Infof("Starting RunGlideTestForever in state %s", pilot.state)
	for {
		if previousState != pilot.state {
			Logger.Infof("RunGlideTestForever new state %s", pilot.state)
			previousState = pilot.state
		}
		pilot.statusIndicator.BlinkState(uint8(pilot.state))

		// Parse all queued messages
		for {
			parsed, err := pilot.telemetry.ParseQueuedMessage()
			if err != nil && err != io.EOF {
				Logger.Errorf("Unable to parse GPS message: %v", err)
				break
			}
			if !parsed {
				break
			}
		}

		switch pilot.state {
		case initializing:
			pilot.runInitializing()
		case waitingForButton:
			pilot.runWaitingForButton()
		case waitingForLaunch:
			pilot.runWaitForLaunch()
		case flying:
			pilot.runFlying()
		case landed:
			pilot.runLanded()
		case testMode:
			pilot.runGlideLevel()
		}

		// TODO: Maybe we want to figure out how long one iteration
		// took, then sleep an appropriate amount of time, so we can get
		// close to however many cycles per second
		time.Sleep(configuration.IterationSleepTime)
	}
}

func (pilot *Pilot) runInitializing() {
	if pilot.telemetry.HasGpsLock {
		pilot.state = waitingForButton
		Logger.Info("Got GPS lock, waiting for button")
	}
}

func (pilot *Pilot) runWaitingForButton() {
	buttonState := pilot.buttonPin.Read()
	if buttonState == rpio.Low {
		Logger.Info("Button pressed, waiting for launch")
		pilot.state = waitingForLaunch
		pilot.buttonPressTime = time.Now()
	}
}

func (pilot *Pilot) runWaitForLaunch() {
	// Just adjust the ailerons to keep the plane level
	if time.Since(pilot.buttonPressTime) < configuration.LaunchGlideDuration {
		pilot.runGlideLevel()
	} else {
		pilot.state = flying
	}
}

func (pilot *Pilot) runFlying() {
	// TODO: When we launch the balloon, check that the altitude is
	// below configuration.LandingPointAltitude + configuration.LandingPointAltitudeOffset
	if pilot.telemetry.GetSpeed() > 0.1 {
		pilot.zeroSpeedTime = nil
	} else if time.Since(*pilot.zeroSpeedTime) > configuration.LandNoMoveDuration {
		pilot.state = landed
		return
	}

	position := pilot.telemetry.GetPosition()
	if pilot.waypoints.Reached(position) {
		pilot.waypoints.Next()
	}

	waypoint := pilot.waypoints.GetWaypoint()
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("runFlying unable to get axes: %v", err)
		time.Sleep(configuration.ErrorSleepDuration)
		return
	}
	targetRoll_d := getTargetRoll(axes.Yaw, position, waypoint)
	pilot.adjustAileronsToRollPitch(targetRoll_d, configuration.TargetPitch, axes)
}

func (pilot *Pilot) runLanded() {
	// Move the servos to center
	pilot.control.SetLeft(90)
	pilot.control.SetRight(90)

	// When the button is pressed, start over
	buttonState := pilot.buttonPin.Read()
	if buttonState == rpio.Low {
		pilot.state = waitingForLaunch
		Logger.Info("Waiting for launch")
	}
}

// Just adjust the ailerons to fly roll level. Good for testing or
// immediately after launch.
func (pilot *Pilot) runGlideLevel() {
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("adjustAileronsToRollPitch unable to get axes: %v", err)
		time.Sleep(configuration.ErrorSleepDuration)
		return
	}
	// Now adjust the ailerons to fly straight
	pilot.adjustAileronsToRollPitch(0.0, 0.0, axes)
}

// Adjust the ailerons to match some pitch and roll
func (pilot *Pilot) adjustAileronsToRollPitch(targetRoll_r, targetPitch_r Radians, axes Axes) {
	// Just use a P loop for now?
	rollDifference := axes.Roll - targetRoll_r
	leftAngle_r := rollDifference * configuration.ProportionalRollMultiplier
	rightAngle_r := leftAngle_r

	/*
	adjustment := (configuration.TargetPitch - axes.Pitch) * configuration.ProportionalPitchMultiplier
	adjustment = clamp(adjustment, -configuration.MaxServoPitchAdjustment, configuration.MaxServoPitchAdjustment)

	leftAngle_r -= adjustment
	rightAngle_r += adjustment
	*/

	leftAngle_r = clamp(leftAngle_r, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)
	rightAngle_r = clamp(rightAngle_r, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)

	// Let's only move the servo when it's changed a little so that the
	// servo isn't freaking out due to noisy sensors
	difference_r := math.Abs(float64(pilot.previousUpdateRoll_r - axes.Roll))
	difference_r += math.Abs(float64(pilot.previousUpdatePitch_r - axes.Pitch))

	/*
	fmt.Printf("roll:%v targetRoll:%v\n", ToDegrees(axes.Roll), ToDegrees(targetRoll))
	fmt.Printf("pitch:%v targetPitch:%v\n", ToDegrees(axes.Pitch), ToDegrees(targetPitch))
	fmt.Printf("leftAngle:%v rightAngle:%v\n", ToDegrees(leftAngle), ToDegrees(rightAngle))
	*/
	if difference_r < float64(ToRadians(4)) {
		//fmt.Printf("Difference %v is too low\n\n", ToDegrees(difference_r))
		return
	}

	pilot.previousUpdateRoll_r = axes.Roll
	pilot.previousUpdatePitch_r = axes.Pitch
	leftAngle_r = roundToUnit(leftAngle_r, 3)
	rightAngle_r = roundToUnit(rightAngle_r, 3)
	//fmt.Printf("Setting leftAngle:%v rightAngle:%v\n\n", ToDegrees(leftAngle), ToDegrees(rightAngle))
	pilot.control.SetLeft(ToRadians(90) + leftAngle_r)
	pilot.control.SetRight(ToRadians(90) + rightAngle_r)
}

func getTargetRoll(yaw_r Radians, position, waypoint Point) Radians {
	goalHeading_r := Course(position, waypoint)
	adjustHeading_r := GetAngleTo(yaw_r, goalHeading_r)
	if GetTurnDirection(yaw_r, position, waypoint) == Left {
		adjustHeading_r = -adjustHeading_r
	}
	targetRoll_r := adjustHeading_r * configuration.ProportionalTargetRollMultiplier
	targetRoll_r = clamp(targetRoll_r, -configuration.MaxTargetRoll, configuration.MaxTargetRoll)
	return targetRoll_r
}

func roundToUnit(x, unit float32) float32 {
	bigX := float64(x)
	bigUnit := float64(unit)
	value := math.Round(bigX/bigUnit) * bigUnit
	return float32(value)
}

func clamp(value, minimum, maximum float32) float32 {
	if minimum > maximum {
		temp := minimum
		minimum = maximum
		maximum = temp
		Logger.Errorf("clamp minimum and maximum were reversed: %v %v", minimum, maximum)
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
