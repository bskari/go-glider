package glider

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"
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
