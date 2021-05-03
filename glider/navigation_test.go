package glider

import (
	"math"
	"testing"
)

func TestDistanceFormulas(t *testing.T) {
	// Just check that the three formulas are similar
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  0,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  0,
	}
	haversine := haversineDistance(start, end)
	spherical := sphericalLawOfCosinesDistance(start, end)
	rectangular := equirectangularDistance(start, end)
	cached := cachedEquirectangularDistance(start, end)

	// Make sure that it's not small, anyway
	if haversine < 1000 {
		t.Errorf("Test case too small")
	}

	if math.Abs(float64(haversine)-float64(spherical)) > 1 {
		t.Errorf("haversine %v spherical %v", haversine, spherical)
	}
	if math.Abs(float64(haversine)-float64(rectangular)) > 1 {
		t.Errorf("haversine %v rectangular %v", haversine, rectangular)
	}
	if math.Abs(float64(haversine)-float64(cached)) > 1 {
		t.Errorf("haversine %v cached %v", haversine, cached)
	}
	if math.Abs(float64(spherical)-float64(rectangular)) > 1 {
		t.Errorf("spherical %v rectangular %v", spherical, rectangular)
	}
	if math.Abs(float64(spherical)-float64(cached)) > 1 {
		t.Errorf("spherical %v cached %v", spherical, cached)
	}
	if math.Abs(float64(rectangular)-float64(cached)) > 1 {
		t.Errorf("rectangular %v cached %v", rectangular, cached)
	}
}

func TestCardinalDistanceFormulas(t *testing.T) {
	// Just check that the formulas are similar
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  0,
	}
	end := Point{
		Latitude:  41.0,
		Longitude: -105.0,
		Altitude:  0,
	}
	haversine := haversineDistance(start, end)
	cardinal := latitudeDistance(start.Latitude, end.Latitude)

	// Make sure that it's not small, anyway
	if haversine < 1000 {
		t.Errorf("Test case too small")
	}

	if math.Abs(float64(haversine)-float64(cardinal)) > 1 {
		t.Errorf("haversine %v cardinal %v", haversine, cardinal)
	}

	// Just check that the formulas are similar
	start = Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  0,
	}
	end = Point{
		Latitude:  40.0,
		Longitude: -104.0,
		Altitude:  0,
	}
	haversine = haversineDistance(start, end)
	cardinal = longitudeDistance(start, end)

	// Make sure that it's not small, anyway
	if haversine < 1000 {
		t.Errorf("Test case too small")
	}

	if math.Abs(float64(haversine)-float64(cardinal)) > 1 {
		t.Errorf("haversine %v cardinal %v", haversine, cardinal)
	}
}

func TestDistance(t *testing.T) {
	// Just check that the chosen distance formula is close to the most
	// accurate one
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  1500,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  1500,
	}
	distance := Distance(start, end)
	haversine := haversineDistance(start, end)

	// Make sure that it's not small, anyway
	if haversine < 1000 {
		t.Errorf("Test case too small")
	}

	if math.Abs(float64(haversine)-float64(distance)) > 1 {
		t.Errorf("haversine %v distance %v", haversine, distance)
	}
}

func TestBearingFormulas(t *testing.T) {
	// Just check that the chosen bearing formula is close to the most accurate
	// one
	start := Point{
		Latitude:  40.0,
		Longitude: 105.0,
		Altitude:  1500,
	}
	const latitudeOffset = 0.5
	const longitudeOffset = 0.653 // Approximately same distance as latitude = 0.5 at latitude 40
	endPoints := []Point{
		// Up right
		Point{
			Latitude:  start.Latitude + latitudeOffset,
			Longitude: start.Longitude + longitudeOffset,
			Altitude:  1500,
		},
		// Right
		Point{
			Latitude:  start.Latitude,
			Longitude: start.Longitude + longitudeOffset,
			Altitude:  1500,
		},
		// Down right
		Point{
			Latitude:  start.Latitude - latitudeOffset,
			Longitude: start.Longitude + longitudeOffset,
			Altitude:  1500,
		},
		// Down
		Point{
			Latitude:  start.Latitude - latitudeOffset,
			Longitude: start.Longitude,
			Altitude:  1500,
		},
		// Down left
		Point{
			Latitude:  start.Latitude - latitudeOffset,
			Longitude: start.Longitude - longitudeOffset,
			Altitude:  1500,
		},
		// Left
		Point{
			Latitude:  start.Latitude,
			Longitude: start.Longitude - longitudeOffset,
			Altitude:  1500,
		},
		// Up left
		Point{
			Latitude:  start.Latitude + latitudeOffset,
			Longitude: start.Longitude - longitudeOffset,
			Altitude:  1500,
		},
		// Up
		Point{
			Latitude:  start.Latitude + latitudeOffset,
			Longitude: start.Longitude,
			Altitude:  1500,
		},
	}

	expected := []Degrees{45, 90, 135, 180, 225, 270, 315, 360}

	for i := 0; i < len(endPoints); i++ {
		end := endPoints[i]
		equirectangular := equirectangularBearing(start, end)
		cached := cachedEquirectangularBearing(start, end)

		// Sanity test
		if math.Abs(equirectangular-ToRadians(expected[i])) > 1 {
			t.Errorf("equirectangular %v expected %v", equirectangular, expected[i])
		}

		if math.Abs(equirectangular-cached) > 0.5 {
			t.Errorf("equirectangular %v cached %v", equirectangular, cached)
		}
	}
}

func TestGetTurnDirection(t *testing.T) {
	// We can't really test Straight and UTurns because of float
	// precision. It'll return Left or Right instead of Straight.
	var direction TurnDirection

	// 0 degrees right turns
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{1, 1, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{-1, 1, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{0, 1, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{0, 1, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	// 0 degrees left turns
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{1, -1, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{0, -1, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(0, Point{0, 0, 0}, Point{-1, -1, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}

	// Go towards up from different angles
	direction = GetTurnDirection(ToRadians(179), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(135), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(90), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(45), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(30), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(1), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(181), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(215), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(270), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(359), Point{0, 0, 0}, Point{1, 0, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}

	// Random tests
	direction = GetTurnDirection(ToRadians(85), Point{1, 3, 0}, Point{2, -1, 0})
	if direction != Left {
		t.Errorf("Bad turn direction %v", direction)
	}
	direction = GetTurnDirection(ToRadians(180), Point{1, 3, 0}, Point{1, 2, 0})
	if direction != Right {
		t.Errorf("Bad turn direction %v", direction)
	}
}

func checkGetAngleTo(t *testing.T, angle1, angle2, expectedAngle Degrees) {
	angleTo := GetAngleTo(ToRadians(angle1), ToRadians(angle2))
	expected := ToRadians(expectedAngle)
	if !approximatelyEqual(angleTo, expected) {
		t.Errorf("Bad angle %0.1f %0.1f, expected %0.1f, calculated %0.1f", angle1, angle2, expectedAngle, ToDegrees(angleTo))
		//t.Error("Bad angle")
	}
}

func TestGetAngleTo(t *testing.T) {
	checkGetAngleTo(t, 10, 40, 30)
	checkGetAngleTo(t, 40, 10, -30)
	checkGetAngleTo(t, 150, 190, 40)
	checkGetAngleTo(t, 190, 150, -40)
	checkGetAngleTo(t, 10, 10, 0)
	checkGetAngleTo(t, 0, 180, 180)
	checkGetAngleTo(t, 180, 0, -180)
	checkGetAngleTo(t, 0, 90, 90)
	checkGetAngleTo(t, 90, 0, -90)
	checkGetAngleTo(t, 315, 45, 90)
	checkGetAngleTo(t, 45, 315, -90)
	checkGetAngleTo(t, 350, 90, 100)
	checkGetAngleTo(t, 90, 350, -100)
	checkGetAngleTo(t, 10, 340, -30)
	checkGetAngleTo(t, 340, 10, 30)
	checkGetAngleTo(t, 50, 310, -100)
	checkGetAngleTo(t, 310, 50, 100)
}

func BenchmarkHaversineDistance(b *testing.B) {
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  1500,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  1500,
	}
	for i := 0; i < b.N; i++ {
		haversineDistance(start, end)
	}
}

func BenchmarkSphericalLawOfCosinesDistance(b *testing.B) {
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  1500,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  1500,
	}
	for i := 0; i < b.N; i++ {
		sphericalLawOfCosinesDistance(start, end)
	}
}

func BenchmarkEquirectangularDistance(b *testing.B) {
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  1500,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  1500,
	}
	for i := 0; i < b.N; i++ {
		equirectangularDistance(start, end)
	}
}

func BenchmarkCachedEquirectangularDistance(b *testing.B) {
	start := Point{
		Latitude:  40.0,
		Longitude: -105.0,
		Altitude:  1500,
	}
	end := Point{
		Latitude:  40.5,
		Longitude: -105.5,
		Altitude:  1500,
	}
	for i := 0; i < b.N; i++ {
		cachedEquirectangularDistance(start, end)
	}
}
