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
	expectedBat := "min_voltage >= 14.80 && max_voltage <= 25.20 && max_current_a >= 140.00 && connector == \"XT60\""
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
	// Analog vs DJI O3 mismatch
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

	// DJI O4 Pro vs DJI O4 Lite mismatch
	compO4Mismatch := &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "DJI O4 Pro"},
		Cameras: []*pb.Camera{
			{Protocol: "DJI O4 Lite"},
		},
	}
	messagesO4 := CheckCompatibility(compO4Mismatch)
	if len(messagesO4) == 0 {
		t.Fatalf("Expected incompatibility message for DJI O4 Pro vs O4 Lite mismatch, got none")
	}
	if messagesO4[0].SeverityName != "DEFINITE_INCOMPATIBILITY" {
		t.Errorf("Expected DEFINITE_INCOMPATIBILITY, got %s", messagesO4[0].SeverityName)
	}

	// DJI O4 Pro matched
	compO4ProMatch := &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "DJI O4 Pro"},
		Cameras: []*pb.Camera{
			{Protocol: "DJI O4 Pro"},
		},
	}
	if msgs := CheckCompatibility(compO4ProMatch); len(msgs) != 0 {
		t.Errorf("Expected 0 messages for matched DJI O4 Pro, got %d", len(msgs))
	}

	// DJI O4 Lite matched
	compO4LiteMatch := &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "DJI O4 Lite"},
		Cameras: []*pb.Camera{
			{Protocol: "DJI O4 Lite"},
		},
	}
	if msgs := CheckCompatibility(compO4LiteMatch); len(msgs) != 0 {
		t.Errorf("Expected 0 messages for matched DJI O4 Lite, got %d", len(msgs))
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

func TestBatteryConnectorCompatibility(t *testing.T) {
	t.Run("Mismatched Connector", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 11.1,
				MaxVoltage: 26.0,
			},
			Motor: &pb.Motor{
				MinVoltage:  14.8,
				MaxVoltage:  25.2,
				MaxCurrentA: 35.0, // 140A total -> XT60
			},
			Battery: &pb.Battery{
				MinVoltage:  14.8,
				MaxVoltage:  25.2,
				MaxCurrentA: 150.0,
				Connector:   "XT30", // Mismatched! Build expects XT60
			},
		}

		messages := CheckCompatibility(comp)
		var found bool
		for _, m := range messages {
			if m.CheckerName == "BatteryConnectorChecker" {
				found = true
				if m.SeverityName != "DEFINITE_INCOMPATIBILITY" {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY, got %s", m.SeverityName)
				}
			}
		}
		if !found {
			t.Errorf("Expected BatteryConnectorChecker message, got none")
		}
	})

	t.Run("Matched Connector", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 11.1,
				MaxVoltage: 26.0,
			},
			Motor: &pb.Motor{
				MinVoltage:  14.8,
				MaxVoltage:  25.2,
				MaxCurrentA: 35.0, // XT60
			},
			Battery: &pb.Battery{
				MinVoltage:  14.8,
				MaxVoltage:  25.2,
				MaxCurrentA: 150.0,
				Connector:   "XT60", // Matched
			},
		}

		messages := CheckCompatibility(comp)
		for _, m := range messages {
			if m.CheckerName == "BatteryConnectorChecker" {
				t.Errorf("Unexpected BatteryConnectorChecker message for matched connector: %s", m.Message)
			}
		}
	})

	t.Run("1S Whoop Cross Compatible BT2.0 and A30", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 3.0,
				MaxVoltage: 4.35,
			},
			Battery: &pb.Battery{
				MinVoltage:  3.0,
				MaxVoltage:  4.35,
				MaxCurrentA: 15.0,
				Connector:   "A30",
			},
		}

		messages := CheckCompatibility(comp)
		for _, m := range messages {
			if m.CheckerName == "BatteryConnectorChecker" {
				t.Errorf("A30 should be compatible with 1S whoop build: %s", m.Message)
			}
		}
	})
}

func TestDetermineBatteryConnector(t *testing.T) {
	t.Run("1S Build BT2.0", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MaxVoltage: 4.35,
			},
		}
		primary, compatible := DetermineBatteryConnector(comp, 3.0, 4.35, 15.0)
		if primary != "BT2.0" {
			t.Errorf("expected primary BT2.0, got %s", primary)
		}
		if len(compatible) != 3 {
			t.Errorf("expected 3 compatible connectors, got %d", len(compatible))
		}
	})

	t.Run("Micro Toothpick XT30", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				WheelbaseMm: 120,
			},
		}
		primary, compatible := DetermineBatteryConnector(comp, 7.4, 16.8, 30.0)
		if primary != "XT30" {
			t.Errorf("expected primary XT30, got %s", primary)
		}
		if len(compatible) != 1 || compatible[0] != "XT30" {
			t.Errorf("expected [XT30], got %v", compatible)
		}
	})

	t.Run("Standard 5 Inch XT60", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				WheelbaseMm: 220,
			},
		}
		primary, compatible := DetermineBatteryConnector(comp, 14.8, 25.2, 140.0)
		if primary != "XT60" {
			t.Errorf("expected primary XT60, got %s", primary)
		}
		if len(compatible) != 1 || compatible[0] != "XT60" {
			t.Errorf("expected [XT60], got %v", compatible)
		}
	})

	t.Run("Explicit Component Mention in Name", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				Name: "SuperX HD AIO 1S Whoop A30",
			},
		}
		primary, _ := DetermineBatteryConnector(comp, 3.0, 4.35, 10.0)
		if primary != "A30" {
			t.Errorf("expected primary A30, got %s", primary)
		}
	})
}

func TestComponentVoltageCompatibility(t *testing.T) {
	t.Run("Conflict: Motor min exceeds ESC max", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{MinVoltage: 11.1, MaxVoltage: 26.0},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{MinVoltage: 6.0, MaxVoltage: 17.4},
			},
			Motor: &pb.Motor{MinVoltage: 18.0, MaxVoltage: 34.8},
		}

		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "ComponentVoltageChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected ComponentVoltageChecker message, got none")
		}
	})

	t.Run("Valid overlapping voltages", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{MinVoltage: 11.1, MaxVoltage: 26.0},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{MinVoltage: 11.1, MaxVoltage: 34.0, MotorCurrentMaxA: 65},
			},
			Motor: &pb.Motor{MinVoltage: 18.0, MaxVoltage: 34.8},
		}

		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "ComponentVoltageChecker" {
				t.Errorf("Unexpected ComponentVoltageChecker message: %s", m.Message)
			}
		}
	})
}

func TestEscChannelCountCompatibility(t *testing.T) {
	t.Run("Insufficient ESC channels", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{MotorCount: 4},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{MaxMotors: 2},
			},
		}

		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "EscChannelCountChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected EscChannelCountChecker message, got none")
		}
	})

	t.Run("Sufficient ESC channels", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{MotorCount: 4},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{MaxMotors: 4},
			},
		}

		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "EscChannelCountChecker" {
				t.Errorf("Unexpected EscChannelCountChecker message: %s", m.Message)
			}
		}
	})
}

func TestBatteryBoundsCompatibility(t *testing.T) {
	comp := &Components{
		FlightController: &pb.FlightController{MinVoltage: 11.1, MaxVoltage: 25.2},
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MinVoltage: 11.1, MaxVoltage: 25.2, MotorCurrentMaxA: 40},
		},
		Motor: &pb.Motor{MinVoltage: 14.8, MaxVoltage: 25.2, MaxCurrentA: 30},
		Battery: &pb.Battery{
			MinVoltage:  10.0, // below 14.8
			MaxVoltage:  30.0, // exceeds 25.2
			MaxCurrentA: 50.0, // below 120.0 (30*4)
		},
	}

	msgs := CheckCompatibility(comp)
	hasMaxV := false
	hasMinV := false
	hasMaxA := false
	for _, m := range msgs {
		if m.CheckerName == "BatteryMaxVoltageChecker" {
			hasMaxV = true
		}
		if m.CheckerName == "BatteryMinVoltageChecker" {
			hasMinV = true
		}
		if m.CheckerName == "BatteryMaxCurrentChecker" {
			hasMaxA = true
		}
	}

	if !hasMaxV {
		t.Errorf("Expected BatteryMaxVoltageChecker message, got none")
	}
	if !hasMinV {
		t.Errorf("Expected BatteryMinVoltageChecker message, got none")
	}
	if !hasMaxA {
		t.Errorf("Expected BatteryMaxCurrentChecker message, got none")
	}
}

func TestGenerateCelFilter_ConflictHandling(t *testing.T) {
	comp := &Components{
		Motor: &pb.Motor{MinVoltage: 18.0, MaxVoltage: 34.8},
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MinVoltage: 6.0, MaxVoltage: 17.4},
		},
	}

	batFilter := GenerateCelFilter("batteries", comp)
	if batFilter != "" {
		t.Errorf("Expected empty battery filter for conflicting components, got %q", batFilter)
	}

	// Test motor min_voltage propagates to ESC filter
	escFilter := GenerateCelFilter("electronic_speed_controllers", &Components{
		Motor: &pb.Motor{MinVoltage: 18.0},
	})
	if escFilter != "max_voltage >= 18.00" {
		t.Errorf("Expected 'max_voltage >= 18.00', got %q", escFilter)
	}

	// Test ESC max_voltage propagates to Motor filter
	motorFilter := GenerateCelFilter("motors", &Components{
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MaxVoltage: 17.4},
		},
	})
	if motorFilter != "min_voltage <= 17.40" {
		t.Errorf("Expected 'min_voltage <= 17.40', got %q", motorFilter)
	}
}
