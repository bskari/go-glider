package glider

import (
	"os"
	"testing"
)

func TestTomlConfiguration(t *testing.T) {
	// Load configuration
	file, err := os.Open("../conf.toml")
	if err != nil {
		t.Error("Unable to open configuration TOML file")
	}
	defer file.Close()
	err = LoadConfiguration(file)
	if err != nil {
		t.Errorf("Unable to load configuration: '%v'", err)
	}
}
