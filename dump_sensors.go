package main

import (
	//"bufio"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/nsf/termbox-go"
	"github.com/stianeikeland/go-rpio/v4"
	//"github.com/tarm/serial"
	"math"
	"math/rand"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/host"
	//"sort"
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

func dumpSensors() {
	/*
		// Set up the GPS
		var gps *bufio.Reader
		if glider.IsPi() {
			serialPorts := []string{"/dev/ttyS0", "/dev/ttyACM0", "/dev/ttyAMA0"}
			for _, serialPort := range serialPorts {
				fmt.Printf("Trying to open GPS %s\n", serialPort)
				config := serial.Config{Name: serialPort, Baud: 9600, ReadTimeout: time.Millisecond * 0}
				gps_, err := serial.OpenPort(&config)
				if err != nil {
					continue
				}
				gps = bufio.NewReader(gps_)
			}
		} else {
			gps = bufio.NewReader(dummyReader{})
		}
	*/

	// Set up accelerometer and magnetometer
	var accelerometer *glider.Adxl345
	var magnetometer *glider.Hmc5883L
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

		accelerometer, err = glider.NewAdxl345(bus)
		if err != nil {
			panic(err)
		}

		magnetometer, err = glider.NewHmc5883L(bus)
		if err != nil {
			panic(err)
		}
	} else {
		accelerometer = nil
		magnetometer = nil
	}

	// Set up button
	var buttonPin *rpio.Pin
	if glider.IsPi() {
		response := rpio.Pin(24)
		buttonPin = &response
		buttonPin.Input()
		buttonPin.PullUp()
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

	//gpsMessageTypeToMessage := make(map[string]string)
	xMinMps := math.MaxFloat64
	yMinMps := math.MaxFloat64
	zMinMps := math.MaxFloat64
	xMaxMps := -math.MaxFloat64
	yMaxMps := -math.MaxFloat64
	zMaxMps := -math.MaxFloat64
	xMinAccelerometerRaw := int16(math.MaxInt16)
	yMinAccelerometerRaw := int16(math.MaxInt16)
	zMinAccelerometerRaw := int16(math.MaxInt16)
	xMaxAccelerometerRaw := int16(-math.MaxInt16)
	yMaxAccelerometerRaw := int16(-math.MaxInt16)
	zMaxAccelerometerRaw := int16(-math.MaxInt16)
	xMinFlux := int16(math.MaxInt16)
	yMinFlux := int16(math.MaxInt16)
	zMinFlux := int16(math.MaxInt16)
	xMaxFlux := int16(math.MinInt16)
	yMaxFlux := int16(math.MinInt16)
	zMaxFlux := int16(math.MinInt16)

	// Let's also test out the LED status indicator
	statusIndicator := glider.NewLedStatusIndicator(3)
	blinkCount := uint8(3)

	writer := &StringWriter{Line: 0}
loop:
	for {
		// The LED needs to update more often than the termbox
		for i := 0; i < 5; i++ {
			if statusIndicator.BlinkState(blinkCount) {
				blinkCount--
				if blinkCount == 0 {
					blinkCount = 3
				}
			}
			time.Sleep(time.Millisecond * 50)
		}

		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		select {
		case event := <-eventQueue:
			// Check for any key presses
			if event.Type == termbox.EventKey {
				break loop
			}
		default:
			writer.Line = 0

			/*
				// Read from the GPS
				text, err := gps.ReadString('\n')
				if err != nil {
					panic(err)
				}
				if text != "" && len(text) > 6 {
					gpsMessageTypeToMessage[text[:6]] = text
				}

				// Output the NMEA sentences
				writer.WriteLine("=== GPS ===")
				gpsTypes := make([]string, 0, len(gpsMessageTypeToMessage))
				for type_ := range gpsMessageTypeToMessage {
					gpsTypes = append(gpsTypes, type_)
				}
				sort.Strings(gpsTypes)
				for _, type_ := range gpsTypes {
					writer.IndentLine(gpsMessageTypeToMessage[type_])
				}
			*/

			// Output accelerometer readings
			var x, y, z physic.Speed
			var xRawA, yRawA, zRawA int16
			if accelerometer != nil {
				x, y, z, err = accelerometer.Sense()
				if err != nil {
					panic(err)
				}
				xRawA, yRawA, zRawA, err = accelerometer.SenseRaw()
				if err != nil {
					panic(err)
				}
			} else {
				offset := -int64(physic.MetrePerSecond) / 10
				randRange := int64(physic.MetrePerSecond) / 5
				x = physic.Speed(offset + rand.Int63n(randRange))
				y = physic.Speed(offset + rand.Int63n(randRange))
				z = physic.Speed(offset + rand.Int63n(randRange) + int64(9.8*float64(physic.MetrePerSecond)))
				xRawA = int16(-10 + rand.Intn(21))
				yRawA = int16(-10 + rand.Intn(21))
				zRawA = int16(90 + rand.Intn(21))
			}
			x2 := int32(xRawA) * int32(xRawA)
			z2 := int32(zRawA) * int32(zRawA)
			pitch_r := math.Atan(float64(yRawA) / math.Sqrt(float64(x2+z2)))
			roll_r := -math.Atan2(float64(xRawA), float64(zRawA))
			pitch_d := glider.ToDegrees(float32(pitch_r))
			roll_d := glider.ToDegrees(float32(roll_r))

			xMps := float64(x) / float64(physic.Newton)
			yMps := float64(y) / float64(physic.Newton)
			zMps := float64(z) / float64(physic.Newton)
			xMinMps = math.Min(xMps, xMinMps)
			yMinMps = math.Min(yMps, yMinMps)
			zMinMps = math.Min(zMps, zMinMps)
			xMaxMps = math.Max(xMps, xMaxMps)
			yMaxMps = math.Max(yMps, yMaxMps)
			zMaxMps = math.Max(zMps, zMaxMps)

			writer.WriteLine("=== Accelerometer ===")
			writer.IndentLine(fmt.Sprintf("x mps: %6.3f, min: %6.3f, max: %6.3f", xMps, xMinMps, xMaxMps))
			writer.IndentLine(fmt.Sprintf("y mps: %6.3f, min: %6.3f, max: %6.3f", yMps, yMinMps, yMaxMps))
			writer.IndentLine(fmt.Sprintf("z mps: %6.3f, min: %6.3f, max: %6.3f", zMps, zMinMps, zMaxMps))

			xMinAccelerometerRaw = min(xRawA, xMinAccelerometerRaw)
			yMinAccelerometerRaw = min(yRawA, yMinAccelerometerRaw)
			zMinAccelerometerRaw = min(zRawA, zMinAccelerometerRaw)
			xMaxAccelerometerRaw = max(xRawA, xMaxAccelerometerRaw)
			yMaxAccelerometerRaw = max(yRawA, yMaxAccelerometerRaw)
			zMaxAccelerometerRaw = max(zRawA, zMaxAccelerometerRaw)
			writer.IndentLine(fmt.Sprintf("x raw: %4d, min: %4d, max: %4d", xRawA, xMinAccelerometerRaw, xMaxAccelerometerRaw))
			writer.IndentLine(fmt.Sprintf("y raw: %4d, min: %4d, max: %4d", yRawA, yMinAccelerometerRaw, yMaxAccelerometerRaw))
			writer.IndentLine(fmt.Sprintf("z raw: %4d, min: %4d, max: %4d", zRawA, zMinAccelerometerRaw, zMaxAccelerometerRaw))

			writer.IndentLine(fmt.Sprintf("pitch: %5.1f   roll: %5.1f", pitch_d, roll_d))

			// Output magnetometer readings
			var xRawM, yRawM, zRawM int16
			if magnetometer != nil {
				xRawM, yRawM, zRawM, err = magnetometer.SenseRaw()
				if err != nil {
					panic(err)
				}
			} else {
				xRawM = int16(-10 + rand.Intn(21))
				yRawM = int16(-10 + rand.Intn(21))
				zRawM = int16(-10 + rand.Intn(21))
			}
			xMinFlux = min(xRawM, xMinFlux)
			yMinFlux = min(yRawM, yMinFlux)
			zMinFlux = min(zRawM, zMinFlux)
			xMaxFlux = max(xRawM, xMaxFlux)
			yMaxFlux = max(yRawM, yMaxFlux)
			zMaxFlux = max(zRawM, zMaxFlux)

			xHorizontal := float64(xRawM)*math.Cos(pitch_r) + float64(yRawM)*math.Sin(roll_r)*math.Sin(pitch_r) - float64(zRawM)*math.Cos(roll_r)*math.Sin(pitch_r)
			yHorizontal := float64(yRawM)*math.Cos(roll_r) + float64(zRawM)*math.Sin(roll_r)
			heading_d := glider.ToDegrees(float32(math.Atan2(yHorizontal, xHorizontal)))
			// The magnetometer is mounted rotated 180 degrees, so rotate it
			heading_d = heading_d + 180
			if heading_d > 360 {
				heading_d -= 360
			}

			writer.WriteLine("=== Magnetometer ===")
			writer.IndentLine(fmt.Sprintf("x: %v, min: %v, max: %v", xRawM, xMinFlux, xMaxFlux))
			writer.IndentLine(fmt.Sprintf("y: %v, min: %v, max: %v", yRawM, yMinFlux, yMaxFlux))
			writer.IndentLine(fmt.Sprintf("z: %v, min: %v, max: %v", zRawM, zMinFlux, zMaxFlux))
			writer.IndentLine(fmt.Sprintf("heading: %v", heading_d))

			// Output button state
			var buttonState rpio.State
			if buttonPin != nil {
				buttonState = buttonPin.Read()
			} else {
				buttonState = rpio.High
			}
			writer.WriteLine("=== Button ===")
			buttonStateString := "unknown"
			if buttonState == rpio.High {
				buttonStateString = "high"
			} else if buttonState == rpio.Low {
				buttonStateString = "low"
			}
			writer.IndentLine(fmt.Sprintf("%v", buttonStateString))

			/*
				// Output temperature
				var value_c float32
				if magnetometer != nil {
					temperature, err := magnetometer.SenseRelativeTemperature()
					if err != nil {
						panic(err)
					}
					value_c = float32(temperature-physic.ZeroCelsius) / float32(physic.Celsius)
				}
				writer.WriteLine("=== Temperature ===")
				writer.IndentLine(fmt.Sprintf("%v", value_c))
			*/

			now := time.Now()
			writer.WriteLine(fmt.Sprintf("%v", now))
			termbox.Flush()
			time.Sleep(250 * time.Millisecond)

		}
	}
}

type StringWriter struct {
	Line int
}

func (writer *StringWriter) WriteLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}
func (writer *StringWriter) IndentLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x+3, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}

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
