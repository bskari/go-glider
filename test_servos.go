package main

import (
	"bufio"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/stianeikeland/go-rpio/v4"
	"os"
	"strconv"
	"time"
)

const LEFT_SERVO_PIN = 12
const RIGHT_SERVO_PIN = 13

func testServos() {
	if !glider.IsPi() {
		fmt.Println("Not a Pi")
		return
	}
	controlTest()
}

func controlTest() {
	control := glider.NewControl()
	angle := float32(45.0)

	// Pause a bit after setting the first angle
	fmt.Printf("Setting angle to %v\n", angle)
	control.SetLeft(angle)
	time.Sleep(250 * time.Millisecond)
	control.SetRight(angle)
	time.Sleep(250 * time.Millisecond)
	angle += 5.0
	time.Sleep(3 * time.Second)

	for angle < 90.0+45.0 {
		fmt.Printf("Setting angle to %v\n", angle)
		control.SetLeft(angle)
		time.Sleep(250 * time.Millisecond)
		control.SetRight(angle)
		time.Sleep(250 * time.Millisecond)
		angle += 5.0
	}
	// Pause a bit after setting the last angle
	time.Sleep(3 * time.Second)

	// Reset back to 90
	angle = 90
	fmt.Printf("Setting angle to %v\n", angle)
	control.SetLeft(angle)
	time.Sleep(5 * time.Millisecond)
	control.SetRight(angle)
	time.Sleep(1 * time.Second)
}

// Manual testing with oscilloscope
func manualTest() {
	left := rpio.Pin(LEFT_SERVO_PIN)
	right := rpio.Pin(RIGHT_SERVO_PIN)
	left.Pwm()
	right.Pwm()
	const HERTZ = glider.HERTZ
	const MULTIPLIER = glider.MULTIPLIER
	left.Freq(HERTZ * MULTIPLIER)
	right.Freq(HERTZ * MULTIPLIER)

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Set Hz to %v\n", HERTZ)
	fmt.Printf("Set Multiplier to %v\n", MULTIPLIER)
	line := "1400"

	for line != "" {
		value, err := strconv.Atoi(line)
		if err != nil {
			fmt.Printf("Bad atoi: %v\n", err)
		}
		valueu32 := uint32(value)
		if valueu32 > 1900 {
			fmt.Println("Too high")
			return
		}
		if valueu32 < 400 {
			fmt.Println("Too low")
			return
		}

		left.DutyCycle(valueu32, MULTIPLIER)
		right.DutyCycle(valueu32, MULTIPLIER)

		fmt.Print("Enter duty cycle: ")
		line, err = reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Bad line: %v\n", err)
		}
		line = line[:len(line)-1]
	}
}
