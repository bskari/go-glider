package main

import (
	"bufio"
	"fmt"
	"log"
	"github.com/tarm/serial"
	"time"
)


func main() {
	config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
	gps_, err := serial.OpenPort(&config)
	if err != nil {
		log.Fatal(err)
	}
	gps := bufio.NewScanner(gps_)

	for true {
		text := gps.Text()
		if text != "" {
			fmt.Println(text)
		}
		time.Sleep(time.Millisecond * 250)
	}
}
