package glider

import (
	"io/ioutil"
	"log"
	"math"
	"strings"
	"testing"
	"time"
)

var isPiCache bool
var isPi bool

func IsPi() bool {
	if isPiCache {
		return isPi
	}

	data, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Fatal("couldn't open /proc/cpuinfo")
	}

	isPi = strings.Contains(string(data), "ARM")
	isPiCache = true
	return isPi
}

func skipIfNotPi(t *testing.T) {
	if !IsPi() {
		t.Skip("Skipping non-Pi")
	}
}

var previousLed bool
var ledEnabled = false

func initializeLed() error {
	if !ledEnabled {
		err := ioutil.WriteFile("/sys/class/leds/led0/trigger", []byte("gpio"), 0644)
		if err != nil {
			return err
		}
		ledEnabled = true
	}
	return nil
}

func ToggleLed() error {
	if !IsPi() {
		return nil
	}
	err := initializeLed()
	if err != nil {
		return err
	}

	ledValue := "1"
	if previousLed {
		ledValue = "0"
	}
	previousLed = !previousLed

	err = ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

func SetLed(on bool) error {
	if !IsPi() {
		return nil
	}
	err := initializeLed()
	if err != nil {
		return err
	}

	// The Pi Zero is reversed, because the power LED doubles as the activity
	// LED, so when there's activity, it's _off_
	ledValue := "0"
	if on {
		ledValue = "1"
	}

	err = ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

type LedStatusIndicator struct {
	blinksToShow        uint8
	ledOn               bool
	currentBlinkCount   uint8
	until               time.Time
	betweenBlinks       time.Duration
	betweenSetsOfBlinks time.Duration
}

func NewLedStatusIndicator(blinksToShow uint8) *LedStatusIndicator {
	betweenBlinks, _ := time.ParseDuration("150ms")
	betweenSetsOfBlinks, _ := time.ParseDuration("500ms")
	return &LedStatusIndicator{
		blinksToShow:        blinksToShow,
		ledOn:               false,
		currentBlinkCount:   0,
		until:               time.Now(),
		betweenBlinks:       betweenBlinks,
		betweenSetsOfBlinks: betweenSetsOfBlinks,
	}
}

// Continues blinking the state. If the blink finishes, it will start blinking
// the next state.
func (statusIndicator *LedStatusIndicator) BlinkState(newBlinkCount uint8) bool {
	now := time.Now()

	if now.After(statusIndicator.until) {
		statusIndicator.ledOn = !statusIndicator.ledOn
		SetLed(statusIndicator.ledOn)
		statusIndicator.until = now.Add(statusIndicator.betweenBlinks)
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

func (statusIndicator *LedStatusIndicator) Reset() {
	statusIndicator.ledOn = false
	SetLed(false)
	statusIndicator.currentBlinkCount = 0
	statusIndicator.until = time.Now().Add(statusIndicator.betweenSetsOfBlinks)
}

func ToDegrees(radians float32) Degrees {
	return radians * (180.0 / math.Pi)
}
