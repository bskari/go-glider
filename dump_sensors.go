package main

import (
	"bufio"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/bskari/go-lsm303"
	"github.com/nsf/termbox-go"
	"github.com/tarm/serial"
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
	var message string
	switch rand.Intn(4) {
	case 0:
		message = fmt.Sprintf("$GPRMC,081836,A,4000.%02d,N,10500.%02d,W,000.0,360.0,130998,011.3,E*65\n", rand.Intn(100), rand.Intn(100))
	case 1:
		message = "$GPGSA,A,3,,,,,,16,18,,22,24,,,3.6,2.1,2.2*3C\n"
	case 2:
		message = fmt.Sprintf("$GPGGA,134658.00,4000.%04d,N,10500.%04d,W,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*60\n", rand.Intn(10000), rand.Intn(10000))
	case 3:
		message = fmt.Sprintf("$GPGLL,4000.%02d,N,10500.%02d,W,225444,A\n", rand.Intn(100), rand.Intn(100))
	default:
		message = "bad switch in test data\n"
	}

	// Only return data some of the time
	if rand.Intn(100) < 10 {
		for i := 0; i < len(message) && i < len(buffer); i++ {
			buffer[i] = message[i]
		}
		return len(message), nil
	} else {
		return 0, nil
	}
}

type Degrees float32

func min(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}

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
	xMinMps := math.MaxFloat64
	yMinMps := math.MaxFloat64
	zMinMps := math.MaxFloat64
	xMaxMps := -math.MaxFloat64
	yMaxMps := -math.MaxFloat64
	zMaxMps := -math.MaxFloat64
	xMinRaw := int16(math.MaxInt16)
	yMinRaw := int16(math.MaxInt16)
	zMinRaw := int16(math.MaxInt16)
	xMaxRaw := int16(-math.MaxInt16)
	yMaxRaw := int16(-math.MaxInt16)
	zMaxRaw := int16(-math.MaxInt16)

loop:
	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

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
			var x, y, z physic.Force
			var xRaw, yRaw, zRaw int16
			if accelerometer != nil {
				x, y, z = accelerometer.Sense()
				xRaw, yRaw, zRaw = accelerometer.SenseRaw()
			} else {
				offset := -int64(physic.EarthGravity) / 10
				randRange := int64(physic.EarthGravity) / 5
				x = physic.Force(offset + rand.Int63n(randRange))
				y = physic.Force(offset + rand.Int63n(randRange))
				z = physic.Force(offset+rand.Int63n(randRange)) + physic.EarthGravity
				xRaw = int16(-10 + rand.Intn(21))
				yRaw = int16(-10 + rand.Intn(21))
				zRaw = int16(90 + rand.Intn(21))
			}
			xMps := float64(x) / float64(physic.Newton)
			yMps := float64(y) / float64(physic.Newton)
			zMps := float64(z) / float64(physic.Newton)
			xMinMps = math.Min(xMps, xMinMps)
			yMinMps = math.Min(yMps, yMinMps)
			zMinMps = math.Min(zMps, zMinMps)
			xMaxMps = math.Max(xMps, xMaxMps)
			yMaxMps = math.Max(yMps, yMaxMps)
			zMaxMps = math.Max(zMps, zMaxMps)
			writeString(fmt.Sprintf("x mps: %6.3f, min: %6.3f, max: %6.3f", xMps, xMinMps, xMaxMps), line)
			line++
			writeString(fmt.Sprintf("y mps: %6.3f, min: %6.3f, max: %6.3f", yMps, yMinMps, yMaxMps), line)
			line++
			writeString(fmt.Sprintf("z mps: %6.3f, min: %6.3f, max: %6.3f", zMps, zMinMps, zMaxMps), line)
			line++

			xMinRaw = min(xRaw, xMinRaw)
			yMinRaw = min(yRaw, yMinRaw)
			zMinRaw = min(zRaw, zMinRaw)
			xMaxRaw = max(xRaw, xMaxRaw)
			yMaxRaw = max(yRaw, yMaxRaw)
			zMaxRaw = max(zRaw, zMaxRaw)
			writeString(fmt.Sprintf("x raw: %4d, min: %4d, max: %4d", xRaw, xMinRaw, xMaxRaw), line)
			line++
			writeString(fmt.Sprintf("y raw: %4d, min: %4d, max: %4d", yRaw, yMinRaw, yMaxRaw), line)
			line++
			writeString(fmt.Sprintf("z raw: %4d, min: %4d, max: %4d", zRaw, zMinRaw, zMaxRaw), line)
			line++

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
