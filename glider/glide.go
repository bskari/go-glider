// Basic glide test
package glider

import (
	"github.com/stianeikeland/go-rpio/v4"
	"time"
)

func Glide() {
	logger := GetLogger()
	if !IsPi() {
		logger.Error("Can't glide on non-Pi hardware")
		return
	}
	// Set up button
	var buttonPin *rpio.Pin
	if IsPi() {
		err := rpio.Open()
		if err != nil {
			panic(err)
		}
		response := rpio.Pin(24)
		buttonPin = &response
		buttonPin.Input()
		buttonPin.PullUp()
	}

	// Blink the LED quickly and wait for button press
	for i := 0; i < 100; i++ {
		ToggleLed()
		time.Sleep(time.Millisecond * 250)
		if buttonPin.Read() == rpio.Low {
			logger.Info("Button pressed")
			break
		}
	}
}
