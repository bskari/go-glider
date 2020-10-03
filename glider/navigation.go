package glider

import (
	"math"
)

// Calculate the distance between two points
func Distance(p1, p2 Point) Meters {
	// Just use equirectangular. It's much faster and we're not flying
	// far enough for it to matter.
	return cachedEquirectangularDistance(p1, p2)
}

const radius_m = 6371e3

func haversineDistance(p1, p2 Point) Meters {
	// Taken from https://www.movable-type.co.uk/scripts/latlong.html
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	deltaPhi := ToCoordinateRadians(p2.Latitude - p1.Latitude)
	deltaDelta := ToCoordinateRadians(p2.Longitude - p1.Longitude)
	a := math.Sin(deltaPhi*0.5)*math.Sin(deltaPhi*0.5) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaDelta*0.5)*math.Sin(deltaDelta*0.5)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return float32(radius_m * c)
}

func sphericalLawOfCosinesDistance(p1, p2 Point) Meters {
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	deltaLambda := ToCoordinateRadians(p2.Longitude - p1.Longitude)
	return Meters(math.Acos(math.Sin(phi1)*math.Sin(phi2)+math.Cos(phi1)*math.Cos(phi2)*math.Cos(deltaLambda)) * radius_m)
}

func latitudeDistance(lat1, lat2 Coordinate) Meters {
	phi1 := ToCoordinateRadians(lat1)
	phi2 := ToCoordinateRadians(lat2)
	y := (phi2 - phi1)
	return Meters(y * radius_m)
}

func longitudeDistance(p1, p2 Point) Meters {
	lambda1 := ToCoordinateRadians(p1.Longitude)
	lambda2 := ToCoordinateRadians(p2.Longitude)
	phi1 := ToCoordinateRadians(p1.Latitude)
	phi2 := ToCoordinateRadians(p2.Latitude)
	x := (lambda2 - lambda1) * math.Cos((phi1+phi2)*0.5)
	return Meters(x * radius_m)
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
		temp := math.Cos((phi1+phi2)*0.5) * radius_m
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

func equirectangularBearing(start, end Point) Degrees {
	y := latitudeDistance(start.Latitude, end.Latitude)
	x := longitudeDistance(start, end)
	theta := math.Atan2(float64(y), float64(x))
	// atan returns anticlockweise direction, so negate it
	bearing := 90 - float32(theta*180/math.Pi) + 360
	if bearing >= 360 {
		bearing -= 360
	}
	return bearing
}

func cachedEquirectangularBearing(start, end Point) Degrees {
	y := latitudeDistance(start.Latitude, end.Latitude)
	x := cachedLongitudeDistance(start, end)
	theta := math.Atan2(float64(y), float64(x))
	bearing := 90 - float32(theta*180/math.Pi) + 360
	if bearing >= 360 {
		bearing -= 360
	}
	return bearing
}

// Returns the course from p1 to p2
func Course(p1, p2 Point) Degrees {
	return cachedEquirectangularBearing(p1, p2)
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
func GetTurnDirection(bearing Degrees, start, end Point) TurnDirection {
	// Based on https://stackoverflow.com/questions/3419341/how-to-calculate-turning-direction
	x := cachedLongitudeDistance(start, end)
	y := latitudeDistance(start.Latitude, end.Latitude)
	// math.Sin expects radians to go anti-clockwise, so flip it
	antiClockwise := 360 - bearing
	// Rotate the vector (0, -10)
	offsetX := float32(10 * math.Sin(float64(ToRadians(antiClockwise))))
	offsetY := float32(-10 * math.Cos(float64(ToRadians(antiClockwise))))

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

func GetAngleTo(bearing, goal Degrees) Degrees {
	var part Degrees
	if bearing > goal {
		part = bearing - goal
	} else {
		part = goal - bearing
	}
	if part > 180 {
		return 360 - part
	}
	return part
}