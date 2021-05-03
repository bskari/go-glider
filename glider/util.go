package glider

import (
	"errors"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
	"math"
	"strings"
	"testing"
	"time"
)

var isPiCache bool
var isPi bool

func IsPi() bool {
	if isPiCache {
		return isPi
	}

	data, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		Logger.Errorf("couldn't open /proc/cpuinfo")
	}

	isPi = strings.Contains(string(data), "ARM")
	isPiCache = true
	return isPi
}

func skipIfNotPi(t *testing.T) {
	if !IsPi() {
		t.Skip("Skipping non-Pi")
	}
}

func approximatelyEqual(a, b float64) bool {
	if math.Abs(a-b) < 0.001 {
		return true
	}
	return false
}

var previousLed bool
var ledEnabled = false

func initializeLed() error {
	if !ledEnabled {
		err := ioutil.WriteFile("/sys/class/leds/led0/trigger", []byte("gpio"), 0644)
		if err != nil {
			return err
		}
		ledEnabled = true
	}
	return nil
}

func ToggleLed() error {
	if !IsPi() {
		return nil
	}
	err := initializeLed()
	if err != nil {
		return err
	}

	ledValue := "1"
	if previousLed {
		ledValue = "0"
	}
	previousLed = !previousLed

	err = ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

func SetLed(on bool) error {
	if !IsPi() {
		return nil
	}
	err := initializeLed()
	if err != nil {
		return err
	}

	// The Pi Zero is reversed, because the power LED doubles as the activity
	// LED, so when there's activity, it's _off_
	ledValue := "0"
	if on {
		ledValue = "1"
	}

	err = ioutil.WriteFile("/sys/class/leds/led0/brightness", []byte(ledValue), 0644)
	if err != nil {
		return err
	}
	return nil
}

type LedStatusIndicator struct {
	blinksToShow        uint8
	ledOn               bool
	currentBlinkCount   uint8
	until               time.Time
	betweenBlinks       time.Duration
	betweenSetsOfBlinks time.Duration
}

func NewLedStatusIndicator(blinksToShow uint8) *LedStatusIndicator {
	betweenBlinks, _ := time.ParseDuration("150ms")
	betweenSetsOfBlinks, _ := time.ParseDuration("500ms")
	return &LedStatusIndicator{
		blinksToShow:        blinksToShow,
		ledOn:               false,
		currentBlinkCount:   0,
		until:               time.Now(),
		betweenBlinks:       betweenBlinks,
		betweenSetsOfBlinks: betweenSetsOfBlinks,
	}
}

// Continues blinking the state. If the blink finishes, it will start blinking
// the next state.
func (statusIndicator *LedStatusIndicator) BlinkState(newBlinkCount uint8) bool {
	now := time.Now()

	if now.After(statusIndicator.until) {
		statusIndicator.ledOn = !statusIndicator.ledOn
		SetLed(statusIndicator.ledOn)
		statusIndicator.until = now.Add(statusIndicator.betweenBlinks)
		if !statusIndicator.ledOn {
			statusIndicator.currentBlinkCount++
			// I expect this to be == but for safety, check >=
			if statusIndicator.currentBlinkCount >= statusIndicator.blinksToShow {
				// Time to go to the new blink count
				statusIndicator.blinksToShow = newBlinkCount
				statusIndicator.Reset()
				return true
			}
		}
	}
	return false
}

func (statusIndicator *LedStatusIndicator) Reset() {
	statusIndicator.ledOn = false
	SetLed(false)
	statusIndicator.currentBlinkCount = 0
	statusIndicator.until = time.Now().Add(statusIndicator.betweenSetsOfBlinks)
}

const PI = 3.14159265358979

func ToDegrees(radians Radians) Degrees {
	return radians * (180.0 / PI)
}

func ToRadians(degrees Degrees) Radians {
	return degrees * (PI / 180.0)
}

func ToCoordinateRadians(coordinate Coordinate) float64 {
	return coordinate * (PI / 180.0)
}

type configuration_t struct {
	DistanceFormula                  distanceFormula_t
	BearingFormula                   bearingFormula_t
	WaypointReachedDistance          Meters
	WaypointInRangeDistance          Meters
	DefaultWaypointLatitude          Coordinate
	DefaultWaypointLongitude         Coordinate
	PitchOffset                      Radians
	RollOffset                       Radians
	MagnetometerXOffset_t            float64
	MagnetometerYOffset_t            float64
	Declination                      Radians
	GpsTty                           string
	GpsBitRate                       int
	IterationSleepTime               time.Duration
	LandNoMoveDuration               time.Duration
	LaunchGlideDuration              time.Duration
	ProportionalRollMultiplier       float64
	ProportionalPitchMultiplier      float64
	ProportionalTargetRollMultiplier float64
	MaxTargetRoll                    Radians
	LandingPointAltitude             Meters
	LandingPointAltitudeOffset       Meters
	TargetPitch                      Radians
	MaxServoPitchAdjustment          Radians
	MaxServoAngleOffset              Radians
	LeftServoCenter_us               uint16
	RightServoCenter_us              uint16
	ButtonPin                        uint8
	LeftServoPin                     uint8
	RightServoPin                    uint8
	ErrorSleepDuration               time.Duration
	FlyDirection                     Radians
}

var configuration configuration_t

type tomlConfiguration_t struct {
	// One of "haversine", "sphericalLawOfCosines", "equirectangular",
	// or "cachedEquirectangular"
	DistanceFormula string
	// One of "equirectangular", "cachedEquirectangular"
	BearingFormula                   string
	WaypointReachedDistance_m        float64
	WaypointInRangeDistance_m        float64
	DefaultWaypointLatitude          float64
	DefaultWaypointLongitude         float64
	PitchOffset_d                    float64
	RollOffset_d                     float64
	MagnetometerXMax_t               float64
	MagnetometerXMin_t               float64
	MagnetometerYMax_t               float64
	MagnetometerYMin_t               float64
	Declination_d                    float64
	GpsTty                           string
	GpsBitRate                       int64
	IterationSleepTime_s             float64
	LandNoMoveDuration_s             float64
	LaunchGlideDuration_s            float64
	ProportionalRollMultiplier       float64
	ProportionalPitchMultiplier      float64
	ProportionalTargetRollMultiplier float64
	MaxTargetRoll_d                  float64
	LandingPointAltitude_m           float64
	LandingPointAltitudeOffset_m     float64
	TargetPitch_d                    float64
	MaxServoPitchAdjustment_d        float64
	MaxServoAngleOffset_d            float64
	LeftServoCenter_us               int64
	RightServoCenter_us              int64
	ButtonPin                        int64
	LeftServoPin                     int64
	RightServoPin                    int64
	ErrorSleepDuration_s             float64
	FlyDirection_d                   float64
}

func LoadConfiguration(configurationReader io.Reader) error {
	var tomlConfiguration tomlConfiguration_t
	_, err := toml.DecodeReader(configurationReader, &tomlConfiguration)
	if err != nil {
		Logger.Errorf("Error loading configuration: %v", err)
	}

	switch tomlConfiguration.DistanceFormula {
	case "haversine":
		configuration.DistanceFormula = DISTANCE_FORMULA_HAVERSINE
	case "sphericalLawOfCosines":
		configuration.DistanceFormula = DISTANCE_FORMULA_SPHERICAL_LAW_OF_COSINES
	case "equirectangular":
		configuration.DistanceFormula = DISTANCE_FORMULA_EQUIRECTANGULAR
	case "cachedEquirectangular":
		configuration.DistanceFormula = DISTANCE_FORMULA_CACHED_EQUIRECTANGULAR
	default:
		return errors.New("Bad DistanceFormula in configuration file")
	}

	switch tomlConfiguration.BearingFormula {
	case "equirectangular":
		configuration.BearingFormula = BEARING_FORMULA_EQUIRECTANGULAR
	case "cachedEquirectangular":
		configuration.BearingFormula = BEARING_FORMULA_CACHED_EQUIRECTANGULAR
	default:
		return errors.New("Bad BearingFormula in configuration file")
	}

	configuration.WaypointReachedDistance = float64(tomlConfiguration.WaypointReachedDistance_m)
	configuration.WaypointInRangeDistance = float64(tomlConfiguration.WaypointInRangeDistance_m)
	configuration.DefaultWaypointLatitude = tomlConfiguration.DefaultWaypointLatitude
	configuration.DefaultWaypointLongitude = tomlConfiguration.DefaultWaypointLongitude

	configuration.PitchOffset = ToRadians(Degrees(tomlConfiguration.PitchOffset_d))
	configuration.RollOffset = ToRadians(Degrees(tomlConfiguration.RollOffset_d))
	configuration.MagnetometerXOffset_t = float64(tomlConfiguration.MagnetometerXMax_t + tomlConfiguration.MagnetometerXMin_t*0.5)
	configuration.MagnetometerYOffset_t = float64(tomlConfiguration.MagnetometerYMax_t + tomlConfiguration.MagnetometerYMin_t*0.5)
	configuration.Declination = float64(tomlConfiguration.Declination_d)
	configuration.GpsTty = tomlConfiguration.GpsTty
	configuration.GpsBitRate = int(tomlConfiguration.GpsBitRate)

	configuration.IterationSleepTime = time.Duration(tomlConfiguration.IterationSleepTime_s * float64(time.Second))

	configuration.ButtonPin = uint8(tomlConfiguration.ButtonPin)
	configuration.LeftServoPin = uint8(tomlConfiguration.LeftServoPin)
	configuration.RightServoPin = uint8(tomlConfiguration.RightServoPin)

	configuration.LandNoMoveDuration = time.Duration(tomlConfiguration.LandNoMoveDuration_s * float64(time.Second))
	configuration.LaunchGlideDuration = time.Duration(tomlConfiguration.LaunchGlideDuration_s * float64(time.Second))
	configuration.ProportionalRollMultiplier = float64(tomlConfiguration.ProportionalRollMultiplier)
	configuration.ProportionalPitchMultiplier = float64(tomlConfiguration.ProportionalPitchMultiplier)
	configuration.ProportionalTargetRollMultiplier = float64(tomlConfiguration.ProportionalTargetRollMultiplier)
	configuration.MaxTargetRoll = ToRadians(Degrees(tomlConfiguration.MaxTargetRoll_d))
	configuration.LandingPointAltitude = Meters(tomlConfiguration.LandingPointAltitude_m)
	configuration.LandingPointAltitudeOffset = Meters(tomlConfiguration.LandingPointAltitudeOffset_m)
	configuration.TargetPitch = ToRadians(Degrees(tomlConfiguration.TargetPitch_d))
	configuration.MaxServoPitchAdjustment = ToRadians(Degrees(tomlConfiguration.MaxServoPitchAdjustment_d))
	configuration.MaxServoAngleOffset = ToRadians(Degrees(tomlConfiguration.MaxServoAngleOffset_d))
	configuration.LeftServoCenter_us = uint16(tomlConfiguration.LeftServoCenter_us)
	configuration.RightServoCenter_us = uint16(tomlConfiguration.RightServoCenter_us)

	configuration.ErrorSleepDuration = time.Duration(tomlConfiguration.ErrorSleepDuration_s * float64(time.Second))
	configuration.FlyDirection = ToRadians(Degrees(tomlConfiguration.FlyDirection_d))

	return nil
}
