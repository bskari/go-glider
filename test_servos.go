package main

import (
	"fmt"
	"github.com/bskari/go-glider/glider"
	"time"
)


func testServos() {
	if !glider.IsPi() {
		fmt.Println("Not a Pi")
		return
	}

	control := glider.NewControl()
	angle := float32(75.0)
	for angle < 105.0 {
		fmt.Printf("Setting angle to %v\n", angle)
		control.SetLeft(angle)
		time.Sleep(5 * time.Millisecond)
		control.SetRight(angle)
		time.Sleep(1 * time.Second)
		angle += 5.0
	}
}
