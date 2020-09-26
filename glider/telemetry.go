package glider

import (
	"bufio"
	"github.com/adrianmo/go-nmea"
	"github.com/bskari/go-lsm303"
	"github.com/tarm/serial"
	"log"
	"math"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"
	"strings"
	"time"
)

// When the plane is level, the accelerometer gives these readings
const PITCH_OFFSET_D = -5.2
const ROLL_OFFSET_D = 2.3

// Magnetometer calibration
const MAGNETOMETER_X_MAX_T = 19.182
const MAGNETOMETER_X_MIN_T = -24.727
const MAGNETOMETER_X_OFFSET_T = (MAGNETOMETER_X_MAX_T + MAGNETOMETER_X_MIN_T) * 0.5
const MAGNETOMETER_Y_MAX_T = 21.364
const MAGNETOMETER_Y_MIN_T = -22.182
const MAGNETOMETER_Y_OFFSET_T = (MAGNETOMETER_Y_MAX_T + MAGNETOMETER_Y_MIN_T) * 0.5

// Declination from true north for Boulder
const DECLINATION_D = 8.1

type Degrees = float32
type Meters = float32

// Coordinate is separate from Degreees because I want to use float64 fo
// extra precision, but it's overkill for other things
type Coordinate = float64

type Point struct {
	Latitude  Coordinate
	Longitude Coordinate
	Altitude  Meters
}

type Axes struct {
	Pitch Degrees
	Roll  Degrees
	Yaw   Degrees
}

type Telemetry struct {
	previousPoint Point
	lastMps       float32
	gps           *bufio.Reader
	accelerometer *lsm303.Accelerometer
	magnetometer  *lsm303.Magnetometer
	timestamp     int64
}

func NewTelemetry() (*Telemetry, error) {
	var gps *bufio.Reader
	var accelerometer *lsm303.Accelerometer
	var magnetometer *lsm303.Magnetometer
	if IsPi() {
		// Make sure periph is initialized.
		if _, err := host.Init(); err != nil {
			return nil, err
		}

		// Prepare GPS
		config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
		gps_, err := serial.OpenPort(&config)
		if err != nil {
			return nil, err
		}
		gps = bufio.NewReader(gps_)

		// Open a connection, using I²C as an example:
		bus, err := i2creg.Open("")
		if err != nil {
			return nil, err
		}
		defer bus.Close()

		// Prepare LSM303
		accelerometer, err = lsm303.NewAccelerometer(bus, &lsm303.DefaultAccelerometerOpts)
		if err != nil {
			return nil, err
		}
		magnetometer, err = lsm303.NewMagnetometer(bus, &lsm303.DefaultMagnetometerOpts)
		if err != nil {
			return nil, err
		}
	}
	return &Telemetry{
		previousPoint: Point{Latitude: 40.0, Longitude: -105.2, Altitude: 1655},
		lastMps:       0.0,
		gps:           gps,
		accelerometer: accelerometer,
		magnetometer:  magnetometer,
	}, nil
}

func (telemetry *Telemetry) GetAxes() (Axes, error) {
	xRawA, yRawA, zRawA, err := telemetry.accelerometer.SenseRaw()
	if err != nil {
		return Axes{0, 0, 0}, err
	}

	xRawM, yRawM, _, err := telemetry.magnetometer.SenseRaw()
	if err != nil {
		return Axes{0, 0, 0}, err
	}

	// Avoid divide by zero problems
	if zRawA == 0 {
		zRawA = 1
	}

	y2 := yRawA * yRawA
	z2 := zRawA * zRawA

	// The roll calculation assumes that -x is forward, +y is right, and
	// +z is down
	return Axes{
		Pitch: Degrees(math.Atan2(float64(xRawA), math.Sqrt(float64(y2+z2)))),
		Roll:  Degrees(math.Atan2(float64(yRawA), float64(zRawA))),
		Yaw:   Degrees(math.Atan2(float64(yRawM), float64(xRawM))),
	}, nil
}

func (telemetry *Telemetry) GetPosition() (Point, error) {
	line, err := telemetry.gps.ReadString('\n')
	if err != nil {
		return telemetry.previousPoint, err
	}
	if line != "" {
		telemetry.parseSentence(line)
	}
	return telemetry.previousPoint, nil
}

func (telemetry *Telemetry) GetTimestamp() int64 {
	return telemetry.timestamp
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
			log.Printf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.RMC)
		telemetry.previousPoint.Latitude = message.Latitude
		telemetry.previousPoint.Longitude = message.Longitude
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
			log.Printf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.GGA)
		telemetry.previousPoint.Latitude = message.Latitude
		telemetry.previousPoint.Longitude = message.Longitude
		telemetry.previousPoint.Altitude = float32(message.Altitude)
	} else if strings.HasPrefix(sentence, "$GPVTG") {
		parsed, err := nmea.Parse(sentence)
		if err != nil {
			log.Printf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.VTG)
		telemetry.lastMps = float32(message.GroundSpeedKPH * 1000.0 / 3600.0)
	}
}
