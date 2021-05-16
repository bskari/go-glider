package glider

import (
	"container/list"
	"fmt"
	"github.com/nsf/termbox-go"
	"time"
)

type stringWriter struct {
	Line int
}

var dashboardTime time.Time = time.Now()
var dashboardMessages *list.List

func (writer *stringWriter) WriteLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}

func (writer *stringWriter) IndentLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x+3, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}

func logDashboard(message string) {
	if dashboardMessages == nil {
		dashboardMessages = list.New()
	}
	now := time.Now()
	formatted := fmt.Sprintf("%s %s", now.Format("15:04:05.000"), message)
	dashboardMessages.PushFront(formatted)
	if dashboardMessages.Len() > 3 {
		dashboardMessages.Remove(dashboardMessages.Back())
	}
}

type StringWriter struct {
	Line int
}

func updateDashboard(telemetry *Telemetry, pilot *Pilot) {
	// Only update this often
	if time.Since(dashboardTime).Milliseconds() < 500 {
		return
	}
	dashboardTime = time.Now()

	writer := &StringWriter{Line: 0}
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	writer.WriteLine("=== GPS ===")
	if telemetry.HasGpsLock {
		writer.IndentLine("(No lock)")
	} else {
		position := telemetry.GetPosition()
		writer.IndentLine(fmt.Sprintf("Lat/Long:%10.5f %10.5f", position.Latitude, position.Longitude))
		writer.IndentLine(fmt.Sprintf("Altitude:%6.1f m", position.Altitude))
	}

	writer.WriteLine("=== Telemetry ===")
	axes, err := telemetry.GetAxes()
	if err != nil {
		writer.IndentLine("(error reading axes)")
	} else {
		writer.IndentLine(fmt.Sprintf("Pitch:%6.1f", ToDegrees(axes.Pitch)))
		writer.IndentLine(fmt.Sprintf("Roll:%6.1f", ToDegrees(axes.Roll)))
		writer.IndentLine(fmt.Sprintf("Yaw:%6.1f", ToDegrees(axes.Yaw)))
	}

	writer.WriteLine("=== State ===")
	writer.IndentLine(fmt.Sprintf("%s", pilot.state))
	if pilot.state == testMode {
		writer.IndentLine(fmt.Sprintf("Target yaw:%6.1f", ToDegrees(configuration.FlyDirection)))
		angle_r := GetAngleTo(axes.Yaw, configuration.FlyDirection)
		writer.IndentLine(fmt.Sprintf("Difference:%6.1f", ToDegrees(angle_r)))
		targetRoll_r := getTargetRollHeading(axes.Yaw, configuration.FlyDirection)
		writer.IndentLine(fmt.Sprintf("Target roll:%6.1f", ToDegrees(targetRoll_r)))
	}

	writer.WriteLine("=== Messages ===")
	for e := dashboardMessages.Back(); e != nil; e = e.Prev() {
		writer.IndentLine(e.Value.(string))
	}

	/*
		writer.WriteLine("=== Raw ===")
		xRawA, yRawA, zRawA, err := telemetry.accelerometer.SenseRaw()
		if err != nil {
			writer.WriteLine("(unable to read accelerometer)")
		} else {
			writer.WriteLine(fmt.Sprintf("Accelerometer: %5d %5d %5d", xRawA, yRawA, zRawA))
		}
		xRawM, yRawM, zRawM, err := telemetry.magnetometer.SenseRaw()
		if err != nil {
			writer.WriteLine("(unable to read magnetometer)")
		} else {
			writer.WriteLine(fmt.Sprintf("Magnetometer: %5d %5d %5d", xRawM, yRawM, zRawM))
		}
	*/

	termbox.Flush()
}

func (writer *StringWriter) WriteLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}

func (writer *StringWriter) IndentLine(str string) {
	for x := 0; x < len(str); x++ {
		termbox.SetCell(x+3, writer.Line, rune(str[x]), termbox.ColorWhite, termbox.ColorBlack)
	}
	writer.Line++
}
