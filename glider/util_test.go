package glider

import (
	"bufio"
	"os"
	"reflect"
	"regexp"
	"testing"
)

func TestLoadConfiguration(t *testing.T) {
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

func TestConfigurationValues(t *testing.T) {
	valueCount := 0
	file, err := os.Open("../conf.toml")
	if err != nil {
		t.Error("Unable to open configuration TOML file")
	}
	err = LoadConfiguration(file)
	if err != nil {
		t.Errorf("Unable to load configuration: '%v'", err)
		return
	}
	file.Close()

	file, err = os.Open("../conf.toml")
	if err != nil {
		t.Error("Unable to open configuration TOML file")
	}
	defer file.Close()
	valueRegex, err := regexp.Compile("^([A-Za-z_]+) =")
	if err != nil {
		t.Error("Unable to compile regex")
	}
	tomlConfiguration := tomlConfiguration_t{}
	tomlConfigurationType := reflect.TypeOf(tomlConfiguration)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if valueRegex.MatchString(line) {
			valueCount += 1
			variableName := valueRegex.FindStringSubmatch(line)[1]
			found := false
			// We could do something better than n^2 and avoid recomputing
			// this list, but who cares in a test
			for i := 0; i < tomlConfigurationType.NumField(); i++ {
				field := tomlConfigurationType.Field(i)
				name := field.Name
				if name == variableName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("'%v' not found in tomlCOnfiguration_t", variableName)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Errorf("Error reading TOML file: %v", err)
	}

	if tomlConfigurationType.NumField() != valueCount {
		t.Errorf("TOML file has %v values but tomlConfiguration_t has %v", valueCount, tomlConfigurationType.NumField())
	}

	configuration := configuration_t{}
	configurationType := reflect.TypeOf(configuration)
	// We average the hard offsets for the magnetometer, so there are
	// two fewer values
	if configurationType.NumField() != valueCount-2 {
		t.Errorf("TOML file has %v values but configuration_t has %v", valueCount, configurationType.NumField())
	}
}
