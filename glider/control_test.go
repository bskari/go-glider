package glider

import (
	"fmt"
	"math"
	"testing"
)

func TestGetDutyCyce(t *testing.T) {
	// This example is taken from the documentation
	targetUs := uint32(math.Round(0.25 * (1.0 / float64(38000)) * 1000 * 1000))
	dutyLength := getDutyCycleForUs(targetUs, 38000, 4)
	exp := float32(targetUs) / (1000000.0 / 38000.0) * 4
	fmt.Printf("expected %v\n", exp)
	if dutyLength != 1 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}

	dutyLength = getDutyCycleForUs(0, 5000, 200)
	if dutyLength != 0 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(10, 5000, 200)
	if dutyLength != 10 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(10, 5000, 100)
	if dutyLength != 5 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(10, 5000, 400)
	if dutyLength != 20 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(100, 5000, 200)
	if dutyLength != 100 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(200, 5000, 200)
	if dutyLength != 200 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}

	// This is a real test case from my use for the servos
	// 50 Hz, want 1400 us for center
	dutyLength = getDutyCycleForUs(1400, 50, 100000)
	if dutyLength != 7000 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
}
