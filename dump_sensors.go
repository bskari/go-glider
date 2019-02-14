package main

import (
	"bufio"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"time"
)

func dumpSensors() {
	config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
	gps_, err := serial.OpenPort(&config)
	if err != nil {
		log.Fatal(err)
	}
	gps := bufio.NewReader(gps_)

	for true {
		text, err := gps.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		if text != "" {
			// text should end in newline anyway
			fmt.Print(text)
		}
		time.Sleep(time.Millisecond * 250)
	}
}
