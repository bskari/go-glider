package glider

import (
	"math"
)

type distanceFormula_t uint8

const (
	DISTANCE_FORMULA_HAVERSINE distanceFormula_t = iota
	DISTANCE_FORMULA_SPHERICAL_LAW_OF_COSINES
	DISTANCE_FORMULA_EQUIRECTANGULAR
	DISTANCE_FORMULA_CACHED_EQUIRECTANGULAR
)

// Calculate the distance between two points
func Distance(p1, p2 Point) Meters {
	switch configuration.DistanceFormula {
	case DISTANCE_FORMULA_HAVERSINE:
		return haversineDistance(p1, p2)
	case DISTANCE_FORMULA_SPHERICAL_LAW_OF_COSINES:
		return sphericalLawOfCosinesDistance(p1, p2)
	case DISTANCE_FORMULA_EQUIRECTANGULAR:
		return equirectangularDistance(p1, p2)
	case DISTANCE_FORMULA_CACHED_EQUIRECTANGULAR:
		return cachedEquirectangularDistance(p1, p2)
	default:
		panic("Bad distanceFormula")
	}
}

const RADIUS_M = 6371e3

func haversineDistance(p1, p2 Point) Meters {
	// Taken from https://www.movable-type.co.uk/scripts/latlong.html
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	deltaPhi := ToCoordinateRadians(p2.Latitude - p1.Latitude)
	deltaDelta := ToCoordinateRadians(p2.Longitude - p1.Longitude)
	a := math.Sin(deltaPhi*0.5)*math.Sin(deltaPhi*0.5) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaDelta*0.5)*math.Sin(deltaDelta*0.5)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return RADIUS_M * c
}

func sphericalLawOfCosinesDistance(p1, p2 Point) Meters {
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	deltaLambda := ToCoordinateRadians(p2.Longitude - p1.Longitude)
	return Meters(math.Acos(math.Sin(phi1)*math.Sin(phi2)+math.Cos(phi1)*math.Cos(phi2)*math.Cos(deltaLambda)) * RADIUS_M)
}

func latitudeDistance(lat1, lat2 Coordinate) Meters {
	phi1 := ToCoordinateRadians(lat1)
	phi2 := ToCoordinateRadians(lat2)
	y := (phi2 - phi1)
	return Meters(y * RADIUS_M)
}

func longitudeDistance(p1, p2 Point) Meters {
	lambda1 := ToCoordinateRadians(p1.Longitude)
	lambda2 := ToCoordinateRadians(p2.Longitude)
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	x := (lambda2 - lambda1) * math.Cos((phi1+phi2)*0.5)
	return Meters(x * RADIUS_M)
}

func equirectangularDistance(p1, p2 Point) Meters {
	x := latitudeDistance(p1.Latitude, p2.Latitude)
	y := longitudeDistance(p1, p2)
	return Meters(math.Sqrt(float64(x*x + y*y)))
}

// Like equirectangularDistance but uses a precomputed cosine value
var longitudeMultiplier *float64

func cachedLongitudeDistance(p1, p2 Point) Meters {
	if longitudeMultiplier == nil {
		phi1 := ToCoordinateRadians(p1.Latitude)
		phi2 := ToCoordinateRadians(p2.Latitude)
		temp := math.Cos((phi1+phi2)*0.5) * RADIUS_M
		longitudeMultiplier = &temp
	}
	lambda1 := ToCoordinateRadians(p1.Longitude)
	lambda2 := ToCoordinateRadians(p2.Longitude)
	x := (lambda2 - lambda1) * *longitudeMultiplier
	return Meters(x)
}

func cachedEquirectangularDistance(p1, p2 Point) Meters {
	x := cachedLongitudeDistance(p1, p2)
	y := latitudeDistance(p1.Latitude, p2.Latitude)
	return Meters(math.Sqrt(float64(x*x + y*y)))
}

func equirectangularBearing(start, end Point) Radians {
	y := latitudeDistance(start.Latitude, end.Latitude)
	x := longitudeDistance(start, end)
	theta_r := math.Atan2(y, x)
	// atan returns anticlockwise direction, so negate it
	bearing_r := ToRadians(90.0) - theta_r + ToRadians(360.0)
	if bearing_r >= ToRadians(360.0) {
		bearing_r -= ToRadians(360.0)
	}
	return bearing_r
}

func cachedEquirectangularBearing(start, end Point) Radians {
	y := latitudeDistance(start.Latitude, end.Latitude)
	x := cachedLongitudeDistance(start, end)
	theta_r := math.Atan2(y, x)
	bearing_r := ToRadians(90.0) - theta_r + ToRadians(360.0)
	if bearing_r >= ToRadians(360.0) {
		bearing_r -= ToRadians(360.0)
	}
	return bearing_r
}

type bearingFormula_t uint8

const (
	BEARING_FORMULA_EQUIRECTANGULAR bearingFormula_t = iota
	BEARING_FORMULA_CACHED_EQUIRECTANGULAR
)

// Returns the course from p1 to p2
func Course(p1, p2 Point) Radians {
	switch configuration.BearingFormula {
	case BEARING_FORMULA_EQUIRECTANGULAR:
		return equirectangularBearing(p1, p2)
	case BEARING_FORMULA_CACHED_EQUIRECTANGULAR:
		return cachedEquirectangularBearing(p1, p2)
	default:
		panic("Bad bearingFormula")
	}
}

type TurnDirection uint8

const (
	Left TurnDirection = iota
	Right
	Straight
	UTurn
)

func (td TurnDirection) String() string {
	return []string{"Left", "Right", "Straight", "UTurn"}[td]
}

// Returns the direction needed to turn to get from start to end
func GetTurnDirection(bearing_r Radians, start, end Point) TurnDirection {
	// Based on https://stackoverflow.com/questions/3419341/how-to-calculate-turning-direction
	x := cachedLongitudeDistance(start, end)
	y := latitudeDistance(start.Latitude, end.Latitude)
	antiClockwise_r := ToRadians(360.0) - bearing_r
	// Rotate the vector (0, -10)
	offsetX := 10 * math.Sin(antiClockwise_r)
	offsetY := -10 * math.Cos(antiClockwise_r)

	crossProduct := y*offsetX - x*offsetY
	if crossProduct < 0 {
		return Left
	}
	if crossProduct > 0 {
		return Right
	}
	dotProduct := y*offsetY + x*offsetX
	if dotProduct > 0 {
		return Straight
	}
	return UTurn
}

func GetAngleTo(bearing_r, goal_r Radians) Radians {
	difference_r := goal_r - bearing_r
	if difference_r > ToRadians(180.0) {
		difference_r -= ToRadians(360.0)
	} else if difference_r < -ToRadians(180.0) {
		difference_r += ToRadians(360.0)
	}
	return difference_r
}
