package main

import (
	"flag"
	"fmt"
)

func main() {
	dumpSensorsPtr := flag.Bool("dump", false, "Dump the sensor data")
	flag.Parse()

	if *dumpSensorsPtr {
		dumpSensors()
	} else {
		fmt.Println("TODO: Run the glider code")
	}
}
