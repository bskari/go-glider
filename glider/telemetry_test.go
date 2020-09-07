package glider

import (
	"testing"
)


func TestParseSentence(t* testing.T) {
	telemetry := NewTelemetry()
    parseSentence(&telemetry, "$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69")
	if telemetry.previousPoint.Latitude_d != 37.0 {
		t.Error("Failed to parse RMC latitude")
	}
	if telemetry.previousPoint.Longitude_d != -133.0 {
		t.Error("Failed to parse RMC longitude")
	}

    parseSentence(&telemetry, "$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43")
	if telemetry.previousPoint.Latitude_d != -43.0 {
		t.Error("Failed to parse GGA latitude")
	}
	if telemetry.previousPoint.Longitude_d != 40.0 {
		t.Error("Failed to parse GGA longitude")
	}

    parseSentence(&telemetry, "$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E")
	if telemetry.lastMps != 2 {
		t.Error("Failed to parse VTG mps")
	}
}


func BenchmarkParseSentence(b* testing.B) {
	telemetry := NewTelemetry()
	for i := 0; i < b.N; i++ {
		parseSentence(&telemetry, "$GPRMC,081836,A,3700.00,N,13300.00,W,000.0,360.0,130998,011.3,E*69")
		parseSentence(&telemetry, "$GPGGA,134658.00,4300.00,S,04000,E,2,09,1.0,1048.47,M,-16.27,M,08,AAAA*43")
		parseSentence(&telemetry, "$GPVTG,054.7,T,034.4,M,005.5,N,007.2,K*4E")
	}
}
