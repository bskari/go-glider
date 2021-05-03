package glider

import (
	"github.com/adrianmo/go-nmea"
	"github.com/argandas/serial"
	"math"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"
	"strings"
	"time"
)

// Just use float64 because all of the math libraries use it
type Degrees = float64
type Radians = float64
type Meters = float64

type Coordinate = float64
type MetersPerSecond = float64

type Point struct {
	Latitude  Coordinate
	Longitude Coordinate
	Altitude  Meters
}

type Axes struct {
	Pitch Radians
	Roll  Radians
	Yaw   Radians
}

type sensor interface {
	SenseRaw() (int16, int16, int16, error)
}

type serialInterface interface {
	Available() int
	ReadLine() (string, error)
}

type concreteSerial struct {
	ser *serial.SerialPort
}

func (cs *concreteSerial) Available() int {
	return cs.ser.Available()
}

func (cs *concreteSerial) ReadLine() (string, error) {
	return cs.ser.ReadLine()
}

// The number of sensor readings to average together
const sensorFilterAverageCount = 3

type sensorFilter struct {
	s                sensor
	previousReadings [sensorFilterAverageCount][3]int32
	name             string
}

func (filter *sensorFilter) SenseRaw() (int16, int16, int16, error) {
	x, y, z, err := filter.s.SenseRaw()
	if err != nil {
		return 0, 0, 0, err
	}

	// Move the previous readings down
	const LEN = len(filter.previousReadings)
	for i := 0; i < LEN-1; i++ {
		for j := 0; j < 3; j++ {
			filter.previousReadings[i][j] = filter.previousReadings[i+1][j]
		}
	}
	filter.previousReadings[LEN-1][0] = int32(x)
	filter.previousReadings[LEN-1][1] = int32(y)
	filter.previousReadings[LEN-1][2] = int32(z)

	var sums [3]int32
	sums[0] = int32(x)
	sums[1] = int32(y)
	sums[2] = int32(z)
	for i := 0; i < LEN-1; i++ {
		for j := 0; j < 3; j++ {
			sums[j] += filter.previousReadings[i][j]
		}
	}
	return int16(sums[0] / int32(LEN)), int16(sums[1] / int32(LEN)), int16(sums[2] / int32(LEN)), nil
}

type Telemetry struct {
	HasGpsLock    bool
	recentPoint   Point
	recentSpeed   MetersPerSecond
	gps           serialInterface
	accelerometer sensorFilter
	magnetometer  sensorFilter
	timestamp     int64
}

func NewTelemetry() (*Telemetry, error) {
	var gps serialInterface
	var accelerometer *Adxl345
	var magnetometer *Hmc5883L
	if IsPi() {
		// Make sure periph is initialized.
		if _, err := host.Init(); err != nil {
			return nil, err
		}

		// Prepare GPS
		rawGps := serial.New()
		rawGps.Verbose = false
		err := rawGps.Open(configuration.GpsTty, configuration.GpsBitRate)
		if err != nil {
			return nil, err
		}
		gps = &concreteSerial{ser: rawGps}

		// Open a connection, using I²C as an example:
		bus, err := i2creg.Open("")
		if err != nil {
			return nil, err
		}

		accelerometer, err = NewAdxl345(bus)
		if err != nil {
			return nil, err
		}
		magnetometer, err = NewHmc5883L(bus)
		if err != nil {
			return nil, err
		}
	}

	accelerometerFilter := sensorFilter{
		s:    accelerometer,
		name: "accel",
	}
	magnetometerFilter := sensorFilter{
		s:    magnetometer,
		name: "mag",
	}
	return &Telemetry{
		recentPoint:   Point{Latitude: 40.0, Longitude: -105.2, Altitude: 1655},
		recentSpeed:   0.0,
		gps:           gps,
		accelerometer: accelerometerFilter,
		magnetometer:  magnetometerFilter,
		HasGpsLock:    false,
	}, nil
}

func (telemetry *Telemetry) GetFilteredAxes() (Axes, error) {
	xRawA, yRawA, zRawA, err := telemetry.accelerometer.SenseRaw()
	if err != nil {
		return Axes{0, 0, 0}, err
	}
	xRawM, yRawM, zRawM, err := telemetry.magnetometer.SenseRaw()
	if err != nil {
		return Axes{0, 0, 0}, err
	}
	return computeAxes(
		xRawA,
		yRawA,
		zRawA,
		xRawM,
		yRawM,
		zRawM,
	), nil
}

func (telemetry *Telemetry) GetAxes() (Axes, error) {
	xRawA, yRawA, zRawA, err := telemetry.accelerometer.SenseRaw()
	Logger.Debugf("accel %v %v %v", xRawA, yRawA, zRawA)
	if err != nil {
		Logger.Info("accelerometer.SenseRaw failed")
		return Axes{0, 0, 0}, err
	}

	xRawM, yRawM, zRawM, err := telemetry.magnetometer.SenseRaw()
	Logger.Debugf("mag %v %v %v", xRawM, yRawM, zRawM)
	if err != nil {
		Logger.Info("magnetometer.SenseRaw failed")
		return Axes{0, 0, 0}, err
	}
	return computeAxes(xRawA, yRawA, zRawA, xRawM, yRawM, zRawM), nil
}

func computeAxes(xRawA, yRawA, zRawA, xRawM, yRawM, zRawM int16) Axes {
	// Avoid divide by zero problems
	if zRawA == 0 {
		zRawA = 1
	}
	// The roll calculation assumes that +y is forward, x is right, and
	// +z is up
	x2 := int32(xRawA) * int32(xRawA)
	z2 := int32(zRawA) * int32(zRawA)

	// Tilt compensated compass readings
	pitch_r := math.Atan2(float64(yRawA), math.Sqrt(float64(x2+z2)))
	roll_r := math.Atan2(float64(xRawA), float64(zRawA))

	// TODO: I think these roll and pitch offset calculations are wrong.
	// We need to figure out the x, y, and z components that are off
	// and then add those above to the raw values.

	pitch_r -= configuration.PitchOffset
	for pitch_r < ToRadians(-180.0) {
		pitch_r += ToRadians(360.0)
	}
	for pitch_r > ToRadians(180.0) {
		pitch_r -= ToRadians(360.0)
	}

	roll_r -= configuration.RollOffset
	for roll_r < ToRadians(-180.0) {
		roll_r += ToRadians(360.0)
	}
	for roll_r > ToRadians(180.0) {
		roll_r -= ToRadians(360.0)
	}

	xM := float64(xRawM - (-490 + 712))
	yM := float64(yRawM - (-569 + 601))
	zM := float64(zRawM - (-704 + 435))
	xHorizontal := xM*math.Cos(-pitch_r) + yM*math.Sin(roll_r)*math.Sin(-pitch_r) - zM*math.Cos(roll_r)*math.Sin(-pitch_r)
	yHorizontal := yM*math.Cos(roll_r) + zM*math.Sin(roll_r)
	yaw_r := math.Atan2(yHorizontal, xHorizontal)
	// The magnetometer is mounted 180 off
	yaw_r += ToRadians(180.0)
	if yaw_r > ToRadians(360.0) {
		yaw_r -= ToRadians(360)
	}
	return Axes{
		Pitch: pitch_r,
		Roll:  roll_r,
		Yaw:   yaw_r,
	}
}

// Parse any waiting GPS messages. Users need not call this, but may.
func (telemetry *Telemetry) ParseQueuedMessage() (bool, error) {
	const MINIMUM_BUFFER = 100

	if telemetry.gps.Available() > MINIMUM_BUFFER {
		// If there is still a message queued, then parse it
		line, err := telemetry.gps.ReadLine()
		if err != nil {
			return false, err
		}
		if line != "" {
			Logger.Debug(strings.TrimSpace(line))
			telemetry.parseSentence(line)
		}
		return true, nil
	}

	// No message is ready, let's abort
	return false, nil
}

func (telemetry *Telemetry) GetPosition() Point {
	// TODO: Hook up GPS and return real readings
	temp := Point{
		// Solid State Depot
		Latitude:  40.020,
		Longitude: -105.250,
		Altitude:  1609,
	}
	return temp
	/*
		_, err := telemetry.ParseQueuedMessage()
		if err != nil {
			Logger.Errorf("Unable to parse GPS message: %v", err)
		}
		// TODO: Do some forward projection or Kalman filtering
		return telemetry.recentPoint
	*/
}

func (telemetry *Telemetry) GetTimestamp() int64 {
	return telemetry.timestamp
}

func (telemetry *Telemetry) GetSpeed() MetersPerSecond {
	return telemetry.recentSpeed
}

func (telemetry *Telemetry) parseSentence(sentence string) {
	// Parses a GPS message and save the output
	// We see $GPGSV, $GPRMC, $GPVTG, $GPGGA, $GPGSA, $GPGLL messages
	// $GPGSV is satellites in view, not useful
	// $GPRMC has latitude, longitude, speed in knots, and magnetic variation
	// $GPVTG has speed in knots and km/h
	// $GPGGA has latitude, longitude, and altitude
	// $GPGSA is active satellites, not useful
	// $GPGLL is just latitude and longitude

	if strings.HasPrefix(sentence, "$GPRMC") {
		parsed, err := nmea.Parse(sentence)
		if err != nil {
			Logger.Errorf("Failed to parse GPRMC message '%v': %v\n", sentence, err)
			return
		}
		message := parsed.(nmea.RMC)
		telemetry.HasGpsLock = (message.Validity == nmea.ValidRMC)
		telemetry.recentPoint.Latitude = message.Latitude
		telemetry.recentPoint.Longitude = message.Longitude
		if telemetry.timestamp == 0 {
			t := time.Date(
				message.Date.YY+2000,
				time.Month(message.Date.MM),
				message.Date.DD,
				message.Time.Hour,
				message.Time.Minute,
				message.Time.Second,
				0,
				time.UTC,
			)
			telemetry.timestamp = t.Unix()
		}
	} else if strings.HasPrefix(sentence, "$GPGGA") {
		parsed, err := nmea.Parse(sentence)
		if err != nil {
			Logger.Errorf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.GGA)
		telemetry.recentPoint.Latitude = message.Latitude
		telemetry.recentPoint.Longitude = message.Longitude
		telemetry.recentPoint.Altitude = message.Altitude
	} else if strings.HasPrefix(sentence, "$GPVTG") {
		parsed, err := nmea.Parse(sentence)
		if err != nil {
			Logger.Errorf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.VTG)
		telemetry.recentSpeed = MetersPerSecond(message.GroundSpeedKPH * 1000.0 / 3600.0)
	}
}
