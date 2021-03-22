package main

import (
	"bytes"
	"fmt"
	"github.com/bskari/go-glider/glider"
	"github.com/nsf/termbox-go"
	"github.com/stianeikeland/go-rpio/v4"
	"math"
	"math/rand"
	"net"
	"os"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/host"
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

// Serves data for calibration on the given port
func serveCalibrationData(port uint16) {
	portString := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", portString)
	check(err)
	fmt.Printf("Listening on port %d\n", port)
	for {
		conn, err := listener.Accept()
		check(err)
		fmt.Println("Got new connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	_, err := host.Init()
	check(err)
	bus, err := i2creg.Open("")
	check(err)
	defer bus.Close()
	accelerometer, err := glider.NewAdxl345(bus)
	check(err)
	magnetometer, err := glider.NewHmc5883L(bus)
	check(err)

	for {
		xRawA, yRawA, zRawA, err := accelerometer.SenseRaw()
		check(err)
		xRawM, yRawM, zRawM, err := magnetometer.SenseRaw()
		check(err)

		// TODO: Spit out gyroscope data too
		line := fmt.Sprintf("Raw:%d,%d,%d,%d,%d,%d,%d,%d,%d\n", xRawA, yRawA, zRawA, 0, 0, 0, xRawM, yRawM, zRawM)
		_, err = conn.Write([]byte(line))
		fmt.Print(".")
		if err != nil {
			fmt.Println("Closing connection")
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func dumpSensors() {
	xMinAccelerometer, xMaxAccelerometer, yMinAccelerometer, yMaxAccelerometer, zMinAccelerometer, zMaxAccelerometer, xMinFlux, xMaxFlux, yMinFlux, yMaxFlux, zMinFlux, zMaxFlux := dumpSensorsInner()

	var buffer [2000]byte
	outputBuffer := bytes.NewBuffer(buffer[:])
	outputBuffer.WriteString("accelerometer\n")
	outputBuffer.WriteString(fmt.Sprintf("x max:%v min:%v\n", xMinAccelerometer, xMaxAccelerometer))
	outputBuffer.WriteString(fmt.Sprintf("y max:%v min:%v\n", yMinAccelerometer, yMaxAccelerometer))
	outputBuffer.WriteString(fmt.Sprintf("z max:%v min:%v\n", zMinAccelerometer, zMaxAccelerometer))
	outputBuffer.WriteString("magnetometer\n")
	outputBuffer.WriteString(fmt.Sprintf("x max:%v min:%v\n", xMinFlux, xMaxFlux))
	outputBuffer.WriteString(fmt.Sprintf("y max:%v min:%v\n", yMinFlux, yMaxFlux))
	outputBuffer.WriteString(fmt.Sprintf("z max:%v min:%v\n", zMinFlux, zMaxFlux))

	// On a read-only filesystem, we won't get an error until we try to write
	// to the file
	fileWriteSuccess := false
	file, err := os.Create("readings.txt")
	if err == nil {
		defer file.Close()
		_, err = file.WriteString(outputBuffer.String())
		if err == nil {
			fileWriteSuccess = true
		}
	}
	if !fileWriteSuccess {
		fmt.Print(outputBuffer.String())
	}
}

func dumpSensorsInner() (int16, int16, int16, int16, int16, int16, int16, int16, int16, int16, int16, int16) {
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
		check(err)
		defer bus.Close()

		accelerometer, err = glider.NewAdxl345(bus)
		check(err)
		magnetometer, err = glider.NewHmc5883L(bus)
		check(err)
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
	check(err)
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
				check(err)
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
				check(err)
				xRawA, yRawA, zRawA, err = accelerometer.SenseRaw()
				check(err)
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
				check(err)
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

			// (These ignore that I mounted it 180 degrees off)
			// Offsets from MotionCal: 80.5, -134.0, -194.0
			// Offsets from keeping 2 accelerometer axes at 0 and measuring raw:
			// (-189 + 334) / 2 = 72, (-399 + 104) / 2 = -148, (-324 + 114) / 2 = -105
			xM := xRawM - 72
			yM := yRawM - -148
			zM := zRawM - -105
			xHorizontal := float64(xM)*math.Cos(-pitch_r) + float64(yM)*math.Sin(roll_r)*math.Sin(-pitch_r) - float64(zM)*math.Cos(roll_r)*math.Sin(-pitch_r)
			yHorizontal := float64(yM)*math.Cos(roll_r) + float64(zM)*math.Sin(roll_r)
			heading_d := glider.ToDegrees(float32(math.Atan2(yHorizontal, xHorizontal)))
			// The magnetometer is mounted rotated 180 degrees, so rotate it
			heading_d = heading_d + 180
			if heading_d > 360 {
				heading_d -= 360
			}

			writer.WriteLine("=== Magnetometer ===")
			writer.IndentLine(fmt.Sprintf("x: %v, raw: %v, min: %v, max: %v", xM, xRawM, xMinFlux, xMaxFlux))
			writer.IndentLine(fmt.Sprintf("y: %v, raw: %v, min: %v, max: %v", yM, yRawM, yMinFlux, yMaxFlux))
			writer.IndentLine(fmt.Sprintf("z: %v, raw: %v, min: %v, max: %v", zM, zRawM, zMinFlux, zMaxFlux))
			writer.IndentLine(fmt.Sprintf("x horizontal: %0.2f", xHorizontal))
			writer.IndentLine(fmt.Sprintf("y horizontal: %0.2f", yHorizontal))
			const declination = 8.1 // Boulder is 8 degrees east
			withDeclination_d := heading_d + declination
			if withDeclination_d > 360 {
				withDeclination_d -= 360
			}
			writer.IndentLine(fmt.Sprintf("heading: %0.1f (with declination %0.1f) %0.1f", heading_d, declination, withDeclination_d))

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
					check(err)
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
	return xMinAccelerometerRaw, xMaxAccelerometerRaw, yMinAccelerometerRaw, yMaxAccelerometerRaw, zMinAccelerometerRaw, zMaxAccelerometerRaw, xMinFlux, xMaxFlux, yMinFlux, yMaxFlux, zMinFlux, zMaxFlux
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

func check(err error) {
	if err != nil {
		panic(err)
	}
}
