package glider

import (
	"testing"
)

func TestGetDutyCyce(t *testing.T) {
	dutyLength := getDutyCycleForUs(0)
	if dutyLength != 0 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
	dutyLength = getDutyCycleForUs(1000 / HERTZ)
	if dutyLength != 20 {
		t.Errorf("Bad dutyLength: %v", dutyLength)
	}
}
