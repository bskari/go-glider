package glider

import (
	"github.com/stianeikeland/go-rpio/v4"
	"time"
)

type PilotState int

const (
	initializing PilotState = iota + 1
	waitingForButton
	waitingForLaunch
	flying
	landed
)

type Pilot struct {
	state           PilotState
	telemetry       *Telemetry
	control         *Control
	statusIndicator LedStatusIndicator
	buttonPin       *rpio.Pin
	buttonPressTime time.Time
	zeroSpeedTime   *time.Time
}

func NewPilot() (*Pilot, error) {
	telemetry, err := NewTelemetry()
	if err != nil {
		return nil, err
	}

	err = rpio.Open()
	if err != nil {
		panic(err)
	}
	response := rpio.Pin(24)
	buttonPin := &response
	buttonPin.Input()
	buttonPin.PullUp()

	launchWait, _ := time.ParseDuration("5s")
	landNoMoveDuration, _ := time.ParseDuration("5s")
	pilotDurations = pilotDurationConstants{
		launchWait:         launchWait,
		landNoMoveDuration: landNoMoveDuration,
	}

	return &Pilot{
		state:           initializing,
		control:         NewControl(),
		telemetry:       telemetry,
		statusIndicator: NewLedStatusIndicator(uint8(initializing)),
		buttonPin:       buttonPin,
		buttonPressTime: time.Now(),
		zeroSpeedTime:   nil,
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
			if err != nil {
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
		}

		// TODO: Maybe we want to figure out how long one iteration
		// took, then sleep an appropriate amount of time, so we can get
		// close to however many cycles per second
		time.Sleep(50)
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
	}
}

func (pilot *Pilot) runWaitForLaunch() {
	// TODO: Give it a few seconds to launch, or wait for x acceleration forward
	Logger.Info("Launched, now flying")
	pilot.state = flying
}

const P_MULTIPLIER = 0.5

func (pilot *Pilot) runFlying() {
	// First, check to see if we have landed
	if pilot.telemetry.GetSpeed() > 0.01 {
		pilot.zeroSpeedTime = nil
	} else if time.Since(*pilot.zeroSpeedTime).Seconds() > 5 {
		pilot.state = landed
		return
	}

	// Now adjust the ailerons to fly straight
	// Just use a PID loop for now?
	axes, err := pilot.telemetry.GetAxes()
	if err != nil {
		// I guess just log it?
		Logger.Errorf("Pilot unable to get axes: %v", err)
		return
	}
	var angle Degrees
	if axes.Roll < -45 {
		angle = -45 * P_MULTIPLIER
	} else if axes.Roll > 45 {
		angle = 45 * P_MULTIPLIER
	} else {
		angle = axes.Roll * P_MULTIPLIER
	}

	// TODO: Adjust for pitch too
	pilot.control.SetLeft(angle)
	pilot.control.SetRight(-angle)
}

func (pilot *Pilot) runLanded() {
	// TODO: Move the servos to center

	// When the button is pressed, start over
	buttonState := pilot.buttonPin.Read()
	if buttonState == rpio.Low {
		pilot.state = waitingForLaunch
	}
}

type pilotDurationConstants struct {
	launchWait         time.Duration
	landNoMoveDuration time.Duration
}

var pilotDurations pilotDurationConstants
