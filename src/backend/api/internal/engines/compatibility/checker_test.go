package compatibility

import (
	pb "quadsmith/api/gen/quadsmith"
	"testing"
)

func TestMotorEscCompatibility(t *testing.T) {
	comp := &Components{
		Motor: &pb.Motor{
			StatorDiameterMm: 22,
		},
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MotorCurrentMaxA: 15},
		},
	}

	messages := CheckCompatibility(comp)

	if len(messages) == 0 {
		t.Fatalf("Expected incompatibility message, got none")
	}

	if messages[0].SeverityName != "HEURISTIC_INCOMPATIBILITY" {
		t.Errorf("Expected HEURISTIC_INCOMPATIBILITY, got %s", messages[0].SeverityName)
	}
}

func TestPropellerFrameCompatibility(t *testing.T) {
	comp := &Components{
		Frame: &pb.Frame{
			MaxPropSizeMm: 5.0,
		},
		Propeller: &pb.Propeller{
			DiameterMm: 5.1,
		},
	}

	messages := CheckCompatibility(comp)

	if len(messages) == 0 {
		t.Fatalf("Expected incompatibility message, got none")
	}

	if messages[0].SeverityName != "DEFINITE_INCOMPATIBILITY" {
		t.Errorf("Expected DEFINITE_INCOMPATIBILITY, got %s", messages[0].SeverityName)
	}
}

func TestCompatibleBuild(t *testing.T) {
	comp := &Components{
		Frame: &pb.Frame{
			MaxPropSizeMm: 5.1,
		},
		Propeller: &pb.Propeller{
			DiameterMm: 5.1,
		},
		Motor: &pb.Motor{
			StatorDiameterMm: 22,
		},
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MotorCurrentMaxA: 45},
		},
	}

	messages := CheckCompatibility(comp)

	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages, got %d", len(messages))
	}
}

func TestGenerateCelFilter(t *testing.T) {
	comp := &Components{
		Frame: &pb.Frame{
			MaxPropSizeMm: 127.0,
			MotorCount:    4,
		},
		FlightController: &pb.FlightController{
			MinVoltage: 14.8,
			MaxVoltage: 25.2,
		},
		Motor: &pb.Motor{
			StatorDiameterMm: 22,
			MaxCurrentA:      35.0,
		},
	}

	propFilter := GenerateCelFilter("propellers", comp)
	if propFilter != "diameter_mm <= 127.0" {
		t.Errorf("expected 'diameter_mm <= 127.0', got %q", propFilter)
	}

	batFilter := GenerateCelFilter("batteries", comp)
	expectedBat := "min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 140.00"
	if batFilter != expectedBat {
		t.Errorf("expected %q, got %q", expectedBat, batFilter)
	}

	escFilter := GenerateCelFilter("electronic_speed_controllers", comp)
	if escFilter != "motor_current_max_a >= 20.0" {
		t.Errorf("expected 'motor_current_max_a >= 20.0', got %q", escFilter)
	}

	frameFilter := GenerateCelFilter("frames", &Components{
		Propeller: &pb.Propeller{DiameterMm: 130.0},
	})
	if frameFilter != "max_prop_size_mm >= 130.0" {
		t.Errorf("expected 'max_prop_size_mm >= 130.0', got %q", frameFilter)
	}

	camFilter := GenerateCelFilter("cameras", &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "DJI O3"},
	})
	if camFilter != `protocol == "DJI O3"` {
		t.Errorf(`expected 'protocol == "DJI O3"', got %q`, camFilter)
	}

	vtxFilter := GenerateCelFilter("video_transmitters", &Components{
		Cameras: []*pb.Camera{{Protocol: "Walksnail Avatar"}},
	})
	if vtxFilter != `protocol == "Walksnail Avatar"` {
		t.Errorf(`expected 'protocol == "Walksnail Avatar"', got %q`, vtxFilter)
	}

	motorFilter := GenerateCelFilter("motors", &Components{
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{{MotorCurrentMaxA: 30.0}},
	})
	if motorFilter != "max_current_a <= 30.0" {
		t.Errorf("expected 'max_current_a <= 30.0', got %q", motorFilter)
	}
}

func TestCameraVtxProtocolCompatibility(t *testing.T) {
	comp := &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "Analog"},
		Cameras: []*pb.Camera{
			{Protocol: "DJI O3"},
		},
	}

	messages := CheckCompatibility(comp)
	if len(messages) == 0 {
		t.Fatalf("Expected incompatibility message for protocol mismatch, got none")
	}
	if messages[0].SeverityName != "DEFINITE_INCOMPATIBILITY" {
		t.Errorf("Expected DEFINITE_INCOMPATIBILITY, got %s", messages[0].SeverityName)
	}
}

func TestBatteryVoltageCompatibility(t *testing.T) {
	comp := &Components{
		FlightController: &pb.FlightController{
			MinVoltage: 3.7,
			MaxVoltage: 8.7, // 1S-2S max
		},
		Battery: &pb.Battery{
			MinVoltage: 18.0, // 6S battery
			MaxVoltage: 25.2,
		},
	}

	messages := CheckCompatibility(comp)
	if len(messages) == 0 {
		t.Fatalf("Expected incompatibility message for battery overvoltage, got none")
	}
	if messages[0].SeverityName != "DEFINITE_INCOMPATIBILITY" {
		t.Errorf("Expected DEFINITE_INCOMPATIBILITY, got %s", messages[0].SeverityName)
	}
}
