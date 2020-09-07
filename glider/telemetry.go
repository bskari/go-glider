package glider

import (
	"bufio"
	"github.com/adrianmo/go-nmea"
	"github.com/tarm/serial"
	"log"
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

type Point struct {
	Latitude_d  float64
	Longitude_d float64
	Altitude_m  float32
}

type Telemetry struct {
	previousPoint Point
	lastMps       float32
	gps           *bufio.Scanner
}

func NewTelemetry() Telemetry {
	var gps *bufio.Scanner
	if isPi() {
		config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
		gps_, err := serial.OpenPort(&config)
		if err != nil {
			log.Fatal(err)
		}
		gps = bufio.NewScanner(gps_)
	} else {
		gps = nil
	}
	return Telemetry{
		previousPoint: Point{Latitude_d: 40.0, Longitude_d: -105.2, Altitude_m: 1655},
		lastMps:       0.0,
		gps:           gps,
	}
}

func (telemetry *Telemetry) GetHeading() float32 {
	return 0.0
}

func (telemetry *Telemetry) GetPosition() Point {
	line := telemetry.gps.Text()
	if line != "" {
		telemetry.parseSentence(line)
	}
	return telemetry.previousPoint
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
		telemetry.previousPoint.Latitude_d = message.Latitude
		telemetry.previousPoint.Longitude_d = message.Longitude
	} else if strings.HasPrefix(sentence, "$GPGGA") {
		parsed, err := nmea.Parse(sentence)
		if err != nil {
			log.Printf("Failed to parse GPRMC message %v\n", sentence)
			return
		}
		message := parsed.(nmea.GGA)
		telemetry.previousPoint.Latitude_d = message.Latitude
		telemetry.previousPoint.Longitude_d = message.Longitude
		telemetry.previousPoint.Altitude_m = float32(message.Altitude)
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
