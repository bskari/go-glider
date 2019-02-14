package glider

import (
	"bufio"
	"testing"
)

func TestParseSentence(t *testing.T) {
	telemetry := NewTelemetry()
	telemetry.parseSentence("$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69")
	if telemetry.previousPoint.Latitude_d != 37.0 {
		t.Error("Failed to parse RMC latitude")
	}
	if telemetry.previousPoint.Longitude_d != -133.0 {
		t.Error("Failed to parse RMC longitude")
	}

	telemetry.parseSentence("$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43")
	if telemetry.previousPoint.Latitude_d != -43.0 {
		t.Error("Failed to parse GGA latitude")
	}
	if telemetry.previousPoint.Longitude_d != 40.0 {
		t.Error("Failed to parse GGA longitude")
	}

	telemetry.parseSentence("$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E")
	if telemetry.lastMps != 2 {
		t.Error("Failed to parse VTG mps")
	}
}

func BenchmarkParseSentence(b *testing.B) {
	telemetry := NewTelemetry()
	for i := 0; i < b.N; i++ {
		telemetry.parseSentence("$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69")
		telemetry.parseSentence("$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43")
		telemetry.parseSentence("$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E")
	}
}

type FakeGps struct {
	lines []string
	count int
}

func (fakeGps *FakeGps) Read(buffer []byte) (count int, err error) {
	target := fakeGps.lines[fakeGps.count]
	fakeGps.count++
	for i := 0; i < len(target); i++ {
		buffer[i] = target[i]
	}
	return len(target), nil
}

func TestGetPosition(t *testing.T) {
	fakeGps := &FakeGps{lines: make([]string, 0), count: 0}
	// Easy case: each read returns a full sentence
	fakeGps.lines = append(fakeGps.lines, "$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69\n")
	fakeGps.lines = append(fakeGps.lines, "$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43\n")
	fakeGps.lines = append(fakeGps.lines, "$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E\n")
	// Hard case: each read returns a partial sentence
	fakeGps.lines = append(fakeGps.lines, "$GPRMC,08183")
	fakeGps.lines = append(fakeGps.lines, "6,A,3700.00,")
	fakeGps.lines = append(fakeGps.lines, "N,13300.00,W")
	fakeGps.lines = append(fakeGps.lines, ",000.0,360.0")
	fakeGps.lines = append(fakeGps.lines, ",130998,011.")
	fakeGps.lines = append(fakeGps.lines, "3,E*69\n")
	// Hard case: split sentences
	fakeGps.lines = append(fakeGps.lines, "$GPVTG,054.7,T,034.4,M,005.5,N")
	fakeGps.lines = append(fakeGps.lines, ",007.2,K*4E\n$GPVTG,054.7,T,034.4,M,005.5,N\n")
	// Hard case: each read returns multiple sentences
	fakeGps.lines = append(fakeGps.lines,
		`$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69\n
		$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43\n
		$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E\n`)

	telemetry := Telemetry{
		previousPoint: Point{Latitude_d: 40.0, Longitude_d: -105.2, Altitude_m: 1655},
		lastMps:       0.0,
		gps:           bufio.NewReader(fakeGps),
	}

	telemetry.GetPosition()
	if fakeGps.count == 0 {
		t.Error("Position not fetched")
	}
}
