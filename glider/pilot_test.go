package glider

import (
	"testing"
)

func TestGetTargetRoll(t *testing.T) {
	configuration.ProportionalTargetRollMultiplier = 1
	maxRoll := Degrees(15.0)
	configuration.MaxTargetRoll = maxRoll
	targetRoll := getTargetRoll(0, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll != 0 {
		t.Errorf("Bad targetRoll: %v", targetRoll)
	}

	targetRoll = getTargetRoll(90, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll != -maxRoll {
		t.Errorf("Bad targetRoll: %v", targetRoll)
	}

	targetRoll = getTargetRoll(270, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll != maxRoll {
		t.Errorf("Bad targetRoll: %v", targetRoll)
	}

	// If we are close to the target, then the number should be lower
	targetRoll = getTargetRoll(1, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll <= -maxRoll*0.25 || targetRoll > 0 {
		t.Errorf("Bad targetRoll: %v", targetRoll)
	}

	targetRoll = getTargetRoll(-1, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll < 0 || targetRoll >= maxRoll*0.25 {
		t.Errorf("Bad targetRoll: %v", targetRoll)
	}
}
