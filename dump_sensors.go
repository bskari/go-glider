package main

import (
	"bufio"
	"github.com/bskari/go-glider/glider"
	"github.com/bskari/go-lsm303"
	"github.com/nsf/termbox-go"
	"github.com/tarm/serial"
	"fmt"
	"math"
	"math/rand"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/host"
	"sort"
	"time"
)

// Used if tesing on non-Pi hardware
type dummyReader struct {
}

func (reader dummyReader) Read(buffer []byte) (n int, err error) {
	const rmc = "$GPRMC,081836,A,3751.65,S,14507.36,E,000.0,360.0,130998,011.3,E*65\n"
	// Only return data X% of the time
	if rand.Intn(100) < 10 {
		for i := 0; i < len(rmc); i++ {
			buffer[i] = rmc[i]
		}
		return len(rmc), nil
	} else {
		return 0, nil
	}
}

type Degrees float32

func dumpSensors() {
	// Set up the GPS
	var gps *bufio.Reader
	if glider.IsPi() {
		config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
		gps_, err := serial.OpenPort(&config)
		if err != nil {
			panic(err)
		}
		gps = bufio.NewReader(gps_)
	} else {
		gps = bufio.NewReader(dummyReader{})
	}

	// Set up accelerometer and magnetometer
	var accelerometer *lsm303.Accelerometer
	if glider.IsPi() {
		// Make sure periph is initialized.
		if _, err := host.Init(); err != nil {
			panic(err)
		}

		// Open a connection, using I²C as an example:
		bus, err := i2creg.Open("")
		if err != nil {
			panic(err)
		}
		defer bus.Close()

		accelerometer, err = lsm303.NewAccelerometer(bus, &lsm303.DefaultOpts)
		if err != nil {
			panic(err)
		}
	} else {
		accelerometer = nil
	}

	// Set up display
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

	gpsMessageTypeToMessage := make(map[string]string)
loop:
	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		xMinMps := math.MaxFloat64
		yMinMps := math.MaxFloat64
		zMinMps := math.MaxFloat64
		xMaxMps := -math.MaxFloat64
		yMaxMps := -math.MaxFloat64
		zMaxMps := -math.MaxFloat64

		select {
		case event := <-eventQueue:
			// Check for any key presses
			if event.Type == termbox.EventKey {
				break loop
			}
		default:

			// Read from the GPS
			text, err := gps.ReadString('\n')
			if err != nil {
				panic(err)
			}
			if text != "" && len(text) > 6 {
				gpsMessageTypeToMessage[text[:6]] = text
			}

			// Output the NMEA sentences
			line := 0
			gpsTypes := make([]string, 0, len(gpsMessageTypeToMessage))
			for type_ := range gpsMessageTypeToMessage {
				gpsTypes = append(gpsTypes, type_)
			}
			sort.Strings(gpsTypes)
			for _, type_ := range gpsTypes {
				writeString(gpsMessageTypeToMessage[type_], line)
				line++
			}

			// Output accelerometer readings
			if accelerometer != nil {
				x, y, z := accelerometer.Sense()
				xMps := float64(x) / float64(physic.Newton)
				yMps := float64(y) / float64(physic.Newton)
				zMps := float64(z) / float64(physic.Newton)
				xMinMps = math.Min(xMps, xMinMps)
				yMinMps = math.Min(yMps, yMinMps)
				zMinMps = math.Min(zMps, zMinMps)
				xMaxMps = math.Max(xMps, xMaxMps)
				yMaxMps = math.Max(yMps, yMaxMps)
				zMaxMps = math.Max(zMps, zMaxMps)
				writeString(fmt.Sprintf("x mps: %v, min: %v, max: %v", xMps, xMinMps, xMaxMps), line)
				line++
				writeString(fmt.Sprintf("y mps: %v, min: %v, max: %v", yMps, yMinMps, yMaxMps), line)
				line++
				writeString(fmt.Sprintf("z mps: %v, min: %v, max: %v", zMps, zMinMps, zMaxMps), line)
				line++
			}

			termbox.Flush()
			time.Sleep(time.Millisecond * 250)
		}
	}
}

func writeString(str string, y int) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x, y, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
}
