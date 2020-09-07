package glider

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

var isPiCache = false

func isPi() bool {
	if isPiCache {
		return true
	}

	data, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		log.Fatal("couldn't open /proc/cpuinfo")
	}

	if strings.Contains(string(data), "ARM") {
		isPiCache = true
		return true
	}

	return false
}

func skipIfNotPi(t *testing.T) {
	if !isPi() {
		t.Skip("Skipping non-Pi")
	}
}
