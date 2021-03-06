package glider

import (
	"github.com/nsf/termbox-go"
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
	previousAxes          Axes
	axesIdleTime          time.Time
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

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

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

		Logger.Debug("Running step")
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
			pilot.runGlideDirection()
		}

		select {
		case event := <-eventQueue:
			// Check for any key presses
			if event.Type == termbox.EventKey {
				return
			}
		default:
			updateDashboard(pilot.telemetry, pilot)
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
	// Fly in a direction

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

	if pilot.hasLanded(axes) {
		pilot.state = landed
		return
	}

	targetRoll_r := getTargetRollPosition(axes.Yaw, position, waypoint)
	pilot.adjustAileronsToRollPitch(targetRoll_r, configuration.TargetPitch, axes)
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

// Just adjust the ailerons to fly in a direction.
func (pilot *Pilot) runGlideDirection() {
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("runGlideDirection unable to get axes: %v", err)
		time.Sleep(configuration.ErrorSleepDuration)
		return
	}

	// If we've landed, stop adjusting the ailerons
	if pilot.hasLanded(axes) {
		pilot.state = landed
		return
	}

	targetRoll_r := getTargetRollHeading(axes.Yaw, configuration.FlyDirection)
	Logger.Debugf("targetRoll:%0.1f", ToDegrees(targetRoll_r))

	// Now adjust the ailerons to fly that direction
	pilot.adjustAileronsToRollPitch(targetRoll_r, configuration.TargetPitch, axes)
}

// Just adjust the ailerons to fly roll level. Good for testing or
// immediately after launch.
func (pilot *Pilot) runGlideLevel() {
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("runGlideLevel unable to get axes: %v", err)
		time.Sleep(configuration.ErrorSleepDuration)
		return
	}
	// If we've landed, stop adjusting the ailerons
	if pilot.hasLanded(axes) {
		pilot.state = landed
		return
	}

	// Now adjust the ailerons to fly straight
	pilot.adjustAileronsToRollPitch(0.0, configuration.TargetPitch, axes)
}

func (pilot *Pilot) hasLanded(axes Axes) bool {
	// TODO: When we launch the balloon, check that the altitude is
	// below configuration.LandingPointAltitude + configuration.LandingPointAltitudeOffset
	var returnValue bool
	if math.Abs(pilot.previousAxes.Roll-axes.Roll) > ToRadians(Degrees(1.0)) {
		pilot.axesIdleTime = time.Now()
		returnValue = false
	} else if time.Since(pilot.axesIdleTime) > configuration.LandNoMoveDuration {
		returnValue = true
	}

	if pilot.telemetry.GetSpeed() > 0.1 {
		pilot.zeroSpeedTime = nil
		returnValue = false
	} else if pilot.zeroSpeedTime == nil {
		returnValue = false
	} else if time.Since(*pilot.zeroSpeedTime) > configuration.LandNoMoveDuration {
		returnValue = returnValue && true
	}

	pilot.previousAxes = axes
	return returnValue
}

// Adjust the ailerons to match some pitch and roll
func (pilot *Pilot) adjustAileronsToRollPitch(targetRoll_r, targetPitch_r Radians, axes Axes) {
	// Just use a P loop for now?
	rollDifference := axes.Roll - targetRoll_r
	leftAngle_r := rollDifference * configuration.ProportionalRollMultiplier
	rightAngle_r := leftAngle_r

	adjustment := (targetPitch_r - axes.Pitch) * configuration.ProportionalPitchMultiplier
	adjustment = clamp(adjustment, -configuration.MaxServoPitchAdjustment, configuration.MaxServoPitchAdjustment)

	leftAngle_r -= adjustment
	rightAngle_r += adjustment

	leftAngle_r = clamp(leftAngle_r, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)
	rightAngle_r = clamp(rightAngle_r, -configuration.MaxServoAngleOffset, configuration.MaxServoAngleOffset)

	// Let's only move the servo when it's changed a little so that the
	// servo isn't freaking out due to noisy sensors
	difference_r := math.Abs(pilot.previousUpdateRoll_r - axes.Roll)
	difference_r += math.Abs(pilot.previousUpdatePitch_r - axes.Pitch)

	Logger.Debugf("roll:%0.1f targetRoll:%0.1f", ToDegrees(axes.Roll), ToDegrees(targetRoll_r))
	Logger.Debugf("pitch:%0.1f targetPitch:%0.1f", ToDegrees(axes.Pitch), ToDegrees(targetPitch_r))
	Logger.Debugf("leftAngle:%0.1f rightAngle:%0.1f", ToDegrees(leftAngle_r), ToDegrees(rightAngle_r))
	if difference_r < ToRadians(4) {
		Logger.Debugf("difference %0.1f is too low", ToDegrees(difference_r))
		return
	}

	pilot.previousUpdateRoll_r = axes.Roll
	pilot.previousUpdatePitch_r = axes.Pitch
	Logger.Debugf("setting leftAngle:%0.1f rightAngle:%0.1f", ToDegrees(leftAngle_r), ToDegrees(rightAngle_r))
	pilot.control.SetLeft(ToRadians(90) + leftAngle_r)
	pilot.control.SetRight(ToRadians(90) + rightAngle_r)
}

func getTargetRollPosition(yaw_r Radians, position, waypoint Point) Radians {
	goalHeading_r := Course(position, waypoint)
	return getTargetRollHeading(yaw_r, goalHeading_r)
}

func getTargetRollHeading(yaw_r, goalHeading_r Radians) Radians {
	adjustHeading_r := GetAngleTo(yaw_r, goalHeading_r)
	targetRoll_r := adjustHeading_r * configuration.ProportionalTargetRollMultiplier
	targetRoll_r = clamp(targetRoll_r, -configuration.MaxTargetRoll, configuration.MaxTargetRoll)
	return targetRoll_r
}

func clamp(value, minimum, maximum float64) float64 {
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
