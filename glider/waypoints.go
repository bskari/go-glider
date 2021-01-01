package glider

// Continues through several waypoints, then repeats the last few
type Waypoints struct {
	first            []Point
	repeating        []Point
	index            int
	inRange          bool
	previousDistance Meters
}

func NewWaypoints() *Waypoints {
	// Wonderland Lake landing site
	return &Waypoints{
		first: []Point{},
		repeating: []Point{
			Point{
				Latitude:  40.055966,
				Longitude: -105.290124,
			},
			Point{
				Latitude:  40.055994,
				Longitude: -105.288681,
			},
			Point{
				Latitude:  40.054785,
				Longitude: -105.289467,
			},
		},
		index:            0,
		inRange:          false,
		previousDistance: 1000000,
	}
}

func (waypoints *Waypoints) GetWaypoint() Point {
	if waypoints.index < len(waypoints.first) {
		return waypoints.first[waypoints.index]
	}
	if waypoints.index < len(waypoints.first)+len(waypoints.repeating) {
		return waypoints.repeating[waypoints.index-len(waypoints.first)]
	}
	Logger.Errorf("Invalid waypoints index: %v", waypoints.index)
	return Point{
		Latitude:  configuration.DefaultWaypointLatitude,
		Longitude: configuration.DefaultWaypointLongitude,
	}
}

func (waypoints *Waypoints) Next() {
	waypoints.index++
	if waypoints.index >= len(waypoints.first)+len(waypoints.repeating) {
		waypoints.index = len(waypoints.first)
	}
	waypoints.inRange = false
	waypoints.previousDistance = 1000000
}

func (waypoints *Waypoints) Reached(current Point) bool {
	waypoint := waypoints.GetWaypoint()
	distance := Distance(current, waypoint)
	// If we are close, then we hit it
	if distance < configuration.WaypointReachedDistance {
		return true
	}
	// If we were within range but have started going further away, count it as
	// reached
	if waypoints.inRange && distance > waypoints.previousDistance {
		return true
	}
	waypoints.previousDistance = distance
	if distance < configuration.WaypointInRangeDistance {
		waypoints.inRange = true
	}
	return false
}
