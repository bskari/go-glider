package glider

import (
	"testing"
)

func TestGetTargetRoll(t *testing.T) {
	configuration.ProportionalTargetRollMultiplier = 1
	maxRoll_r := ToRadians(15.0)
	configuration.MaxTargetRoll = maxRoll_r
	targetRoll_r := getTargetRoll(0, Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll_r != 0 {
		t.Errorf("Bad targetRoll: %v", targetRoll_r)
	}

	targetRoll_r = getTargetRoll(ToRadians(90), Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll_r != -maxRoll_r {
		t.Errorf("Bad targetRoll: %v", targetRoll_r)
	}

	targetRoll_r = getTargetRoll(ToRadians(270), Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll_r != maxRoll_r {
		t.Errorf("Bad targetRoll: %v", targetRoll_r)
	}

	// If we are close to the target, then the number should be lower
	targetRoll_r = getTargetRoll(ToRadians(1), Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll_r <= -maxRoll_r*0.25 || targetRoll_r > 0 {
		t.Errorf("Bad targetRoll: %v", targetRoll_r)
	}

	targetRoll_r = getTargetRoll(ToRadians(-1), Point{0, 0, 0}, Point{1, 0, 0})
	if targetRoll_r < 0 || targetRoll_r >= maxRoll_r*0.25 {
		t.Errorf("Bad targetRoll: %v", targetRoll_r)
	}
}
