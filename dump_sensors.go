package main

import (
	"bufio"
	"github.com/bskari/go-glider/glider"
	"github.com/nsf/termbox-go"
	"github.com/tarm/serial"
	"math/rand"
	"time"
)

type dummyReader struct {
}

func (reader dummyReader) Read(buffer []byte) (n int, err error) {
	const rmc = "$GPRMC,081836,A,3751.65,S,14507.36,E,000.0,360.0,130998,011.3,E*65\n"
	// Only return data X% of the time
	if rand.Intn(100) < 10 {
		for i := 0; i < len(rmc); i++ {
			buffer[i] = rmc[i]
		}
		return len(rmc), nil
	} else {
		return 0, nil
	}
}

func dumpSensors() {
	var gps *bufio.Reader
	if glider.IsPi() {
		config := serial.Config{Name: "/dev/ttyS0", Baud: 9600, ReadTimeout: time.Millisecond * 0}
		gps_, err := serial.OpenPort(&config)
		if err != nil {
			panic(err)
		}
		gps = bufio.NewReader(gps_)
	} else {
		gps = bufio.NewReader(dummyReader{})
	}

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	eventQueue := make(chan termbox.Event)
	go func() {
		for {
			eventQueue <- termbox.PollEvent()
		}
	}()

	gpsMessageTypeToMessage := make(map[string]string)
loop:
	for {
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		select {
		case event := <-eventQueue:
			// Check for any key presses
			if event.Type == termbox.EventKey {
				break loop
			}
		default:

			// Read from the sensors
			text, err := gps.ReadString('\n')
			if err != nil {
				panic(err)
			}
			if text != "" {
				gpsMessageTypeToMessage[text[:5]] = text
			}

			// Output the stuff
			line := 0
			for _, value := range gpsMessageTypeToMessage {
				writeString(value, line)
				line++
			}

			termbox.Flush()
			time.Sleep(time.Millisecond * 250)
		}
	}
}

func writeString(str string, y int) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x, y, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
}
