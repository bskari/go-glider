package glider

import (
	"io/ioutil"
	"log"
	"math"
	"strings"
	"testing"
	"time"
)

var isPiCache = false

func IsPi() bool {
	if isPiCache {
		return true
	}

	data, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Fatal("couldn't open /proc/cpuinfo")
	}

	if strings.Contains(string(data), "ARM") {
		isPiCache = true
		return true
	}

	return false
}

func skipIfNotPi(t *testing.T) {
	if !IsPi() {
		t.Skip("Skipping non-Pi")
	}
}

var previousLed bool
var ledEnabled = false

func ToggleLed() error {
	if !ledEnabled {
		err := ioutil.WriteFile("/sys/class/leds/led0/trigger", []byte("gpio"), 0644)
		if err != nil {
			return err
		}
		ledEnabled = true
	}

	ledValue := "1"
	if previousLed {
		ledValue = "0"
	}
	previousLed = !previousLed

	err := ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

func SetLed(on bool) error {
	if !ledEnabled {
		err := ioutil.WriteFile("/sys/class/leds/led0/trigger", []byte("gpio"), 0644)
		if err != nil {
			return err
		}
		ledEnabled = true
	}

	// The Pi Zero is reversed, because the power LED doubles as the activity
	// LED, so when there's activity, it's _off_
	ledValue := "1"
	if on {
		ledValue = "0"
	}

	err := ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

type LedStatusIndicator struct {
	blinksToShow      uint8
	ledOn             bool
	currentBlinkCount uint8
	until             time.Time
}

func NewLedStatusIndicator(blinksToShow uint8) LedStatusIndicator {
	betweenBlinks, _ := time.ParseDuration("200ms")
	betweenSetsOfBlinks, _ := time.ParseDuration("1s")
	blinkDurations = blinkDurationConstants{
		betweenBlinks:       betweenBlinks,
		betweenSetsOfBlinks: betweenSetsOfBlinks,
	}
	return LedStatusIndicator{
		blinksToShow:      blinksToShow,
		ledOn:             false,
		currentBlinkCount: 0,
		until:             time.Now(),
	}
}

// Continues blinking the state. If the blink finishes, it will start blinking
// the next state.
func (statusIndicator LedStatusIndicator) BlinkState(newBlinkCount uint8) bool {
	if time.Now().After(statusIndicator.until) {
		statusIndicator.ledOn = !statusIndicator.ledOn
		ToggleLed()
		statusIndicator.until = time.Now().Add(blinkDurations.betweenBlinks)
		if !statusIndicator.ledOn {
			statusIndicator.currentBlinkCount++
			// I expect this to be == but for safety, check >=
			if statusIndicator.currentBlinkCount >= statusIndicator.blinksToShow {
				// Time to go to the new blink count
				statusIndicator.blinksToShow = newBlinkCount
				statusIndicator.Reset()
				return true
			}
		}
	}
	return false
}

func (statusIndicator LedStatusIndicator) Reset() {
	statusIndicator.ledOn = false
	statusIndicator.currentBlinkCount = 0
	statusIndicator.until = time.Now().Add(blinkDurations.betweenSetsOfBlinks)
}

type blinkDurationConstants struct {
	betweenBlinks       time.Duration
	betweenSetsOfBlinks time.Duration
}

var blinkDurations blinkDurationConstants

func ToDegrees(radians float32) Degrees {
	return radians * (180.0 / math.Pi)
}
