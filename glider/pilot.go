package glider

import (
	"github.com/stianeikeland/go-rpio/v4"
	"io"
	"math"
	"time"
)

type PilotState uint8

const (
	initializing PilotState = iota + 1
	waitingForButton
	waitingForLaunch
	flying
	landed
	testMode
)

type Pilot struct {
	state               PilotState
	telemetry           *Telemetry
	control             *Control
	statusIndicator     *LedStatusIndicator
	buttonPin           *rpio.Pin
	buttonPressTime     time.Time
	zeroSpeedTime       *time.Time
	waypoints           *Waypoints
	previousUpdateRoll  float32
	previousUpdatePitch float32
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
	Logger.Info("Waiting for GPS lock")
	for {
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
	//position := pilot.telemetry.GetPosition()

	// TODO: When we launch the balloon, check that the altitude is
	// below configuration.LandingPointAltitude + configuration.LandingPointAltitudeOffset
	if pilot.telemetry.GetSpeed() > 0.1 {
		pilot.zeroSpeedTime = nil
	} else if time.Since(*pilot.zeroSpeedTime) > configuration.LandNoMoveDuration {
		pilot.state = landed
		return
	}

	// TODO: Fly to waypoints
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
	// Now adjust the ailerons to fly straight
	// Just use a P loop for now?
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("Pilot unable to get axes: %v", err)
		time.Sleep(configuration.ErrorSleepDuration)
		return
	}
	leftAngle := axes.Roll * configuration.ProportionalRollMultiplier
	rightAngle := -leftAngle

	adjustment := (configuration.TargetPitch - axes.Pitch) * configuration.ProportionalPitchMultiplier
	adjustment = clamp(adjustment, -configuration.MaxServoPitchAdjustment, configuration.MaxServoPitchAdjustment)

	leftAngle += adjustment
	rightAngle -= adjustment

	leftAngle = clamp(leftAngle, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)
	rightAngle = clamp(rightAngle, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)

	// Let's only move the servo when it's changed a little so that the
	// servo isn't freaking out due to noisy sensors
	difference := math.Abs(float64(pilot.previousUpdateRoll - axes.Roll))
	difference += math.Abs(float64(pilot.previousUpdatePitch - axes.Pitch))
	if difference < 4 {
		return
	}

	pilot.previousUpdateRoll = axes.Roll
	pilot.previousUpdatePitch = axes.Pitch
	leftAngle = roundToUnit(leftAngle, 3)
	rightAngle = roundToUnit(rightAngle, 3)
	pilot.control.SetLeft(90 + leftAngle)
	pilot.control.SetRight(90 + rightAngle)
}

func roundToUnit(x, unit float32) float32 {
	bigX := float64(x)
	bigUnit := float64(unit)
	value := math.Round(bigX/bigUnit) * bigUnit
	return float32(value)
}

func clamp(value, minimum, maximum float32) float32 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
