package main

import (
        "fmt"
        "github.com/bskari/go-glider/glider"
)

func main() {
        fmt.Println("Hello, world!")
        telemetry := glider.NewTelemetry()
        fmt.Printf("Heading %v\n", glider.GetHeading(&telemetry))
}
