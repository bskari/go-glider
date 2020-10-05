package glider

import "testing"

func createTestWaypoints() *Waypoints {
	return &Waypoints{
		first: []Point{
			Point{
				Latitude:  1,
				Longitude: 1,
			},
			Point{
				Latitude:  2,
				Longitude: 2,
			},
		},
		repeating: []Point{
			Point{
				Latitude:  3,
				Longitude: 3,
			},
			Point{
				Latitude:  4,
				Longitude: 4,
			},
		},
		index:            0,
		inRange:          false,
		previousDistance: 1000000,
	}
}

func TestGetWaypoint(t *testing.T) {
	waypoints := createTestWaypoints()
	if waypoints.GetWaypoint().Latitude != 1 {
		t.Errorf("Bad waypoint %v", waypoints.GetWaypoint().Latitude)
	}
	waypoints.Next()

	if waypoints.GetWaypoint().Latitude != 2 {
		t.Errorf("Bad waypoint %v", waypoints.GetWaypoint().Latitude)
	}
	waypoints.Next()

	// From here on out they should repeat
	for i := 0; i < 5; i++ {
		if waypoints.GetWaypoint().Latitude != 3 {
			t.Errorf("Bad waypoint %v", waypoints.GetWaypoint().Latitude)
		}
		waypoints.Next()
		if waypoints.GetWaypoint().Latitude != 4 {
			t.Errorf("Bad waypoint %v", waypoints.GetWaypoint().Latitude)
		}
		waypoints.Next()
	}
}

func TestReached(t *testing.T) {
	waypoints := createTestWaypoints()
	if waypoints.Reached(Point{Latitude: 0, Longitude: 0}) {
		t.Error("Bad reached")
	}
	if waypoints.Reached(Point{Latitude: 40, Longitude: -105}) {
		t.Error("Bad reached")
	}

	// Check sort of close
	if waypoints.Reached(Point{Latitude: 0.9996, Longitude: 0.9996}) {
		t.Error("Bad reached")
	}
	// Going closer should still say false...
	if waypoints.Reached(Point{Latitude: 0.9997, Longitude: 0.9997}) {
		t.Error("Bad reached")
	}
	if waypoints.Reached(Point{Latitude: 0.9998, Longitude: 0.9998}) {
		t.Error("Bad reached")
	}
	// But going away again should return true
	if !waypoints.Reached(Point{Latitude: 0.9996, Longitude: 0.9996}) {
		t.Error("Bad reached")
	}

	waypoints = createTestWaypoints()
	// Continually getting close should eventually say true
	if waypoints.Reached(Point{Latitude: 0.9996, Longitude: 0.9996}) {
		t.Error("Bad reached")
	}
	if waypoints.Reached(Point{Latitude: 0.9997, Longitude: 0.9997}) {
		t.Error("Bad reached")
	}
	if waypoints.Reached(Point{Latitude: 0.9998, Longitude: 0.9998}) {
		t.Error("Bad reached")
	}
	if !waypoints.Reached(Point{Latitude: 0.9999, Longitude: 0.9999}) {
		t.Error("Bad reached")
	}
	if !waypoints.Reached(Point{Latitude: 1.0000, Longitude: 1.0000}) {
		t.Error("Bad reached")
	}
}
