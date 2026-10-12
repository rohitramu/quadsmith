package compatibility

import (
	"strings"
	"testing"

	pb "quadsmith/api/gen/quadsmith"
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
	expectedEsc := "motor_current_max_a >= 20.0 && motor_current_max_a >= 35.0 && min_voltage <= 25.20 && max_voltage >= 14.80"
	if escFilter != expectedEsc {
		t.Errorf("expected %q, got %q", expectedEsc, escFilter)
	}

	frameFilter := GenerateCelFilter("frames", &Components{
		Propeller: &pb.Propeller{DiameterMm: 130.0},
	})
	if frameFilter != "max_prop_size_mm >= 130.0" {
		t.Errorf("expected 'max_prop_size_mm >= 130.0', got %q", frameFilter)
	}

	camFilter := GenerateCelFilter("cameras", &Components{
		VideoTransmitter: &pb.VideoTransmitter{Protocol: "DJI O3"},
		Frame:            &pb.Frame{CameraMountWidthMm: 20},
	})
	if camFilter != `protocol == "DJI O3" && width_mm <= 20` {
		t.Errorf(`expected 'protocol == "DJI O3" && width_mm <= 20', got %q`, camFilter)
	}

	vtxFilter := GenerateCelFilter("video_transmitters", &Components{
		FlightController: &pb.FlightController{
			VtxVoltageMinV: 9.0,
			VtxVoltageMaxV: 12.0,
			VtxPorts:       []string{"SH1.0-6P"},
		},
	})
	if vtxFilter != `input_voltage_min_v <= 12.0 && input_voltage_max_v >= 9.0 && port_type == "SH1.0-6P"` {
		t.Errorf(`expected 'input_voltage_min_v <= 12.0 && input_voltage_max_v >= 9.0 && port_type == "SH1.0-6P"', got %q`, vtxFilter)
	}

	motorFilter := GenerateCelFilter("motors", &Components{
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{{MotorCurrentMaxA: 30.0}},
		Frame:                      &pb.Frame{MotorMountPatterns: []string{"16x16mm", "19x19mm"}},
	})
	if motorFilter != `(mount_pattern == "16x16mm" || mount_pattern == "19x19mm") && max_current_a <= 30.0` {
		t.Errorf(`expected '(mount_pattern == "16x16mm" || mount_pattern == "19x19mm") && max_current_a <= 30.0', got %q`, motorFilter)
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

func TestCheckCompatibility_MotorPropellerShaft(t *testing.T) {
	// 1. Matching shafts (5.0mm motor with 5.0mm prop) -> Compatible
	compMatch := &Components{
		Motor:     &pb.Motor{ShaftDiameterMm: 5.0},
		Propeller: &pb.Propeller{ShaftDiameterMm: 5.0},
	}
	msgs := CheckCompatibility(compMatch)
	for _, m := range msgs {
		if m.CheckerName == "MotorPropellerShaftChecker" {
			t.Errorf("Expected no shaft incompatibility for matching 5.0mm shafts, got: %v", m)
		}
	}

	// 2. Mismatched shafts (5.0mm motor with 1.5mm prop) -> Definite Incompatibility
	compMismatch := &Components{
		Motor:     &pb.Motor{ShaftDiameterMm: 5.0},
		Propeller: &pb.Propeller{ShaftDiameterMm: 1.5},
	}
	msgsMismatch := CheckCompatibility(compMismatch)
	found := false
	for _, m := range msgsMismatch {
		if m.CheckerName == "MotorPropellerShaftChecker" {
			found = true
			if m.SeverityLevel != 0 {
				t.Errorf("Expected SeverityLevel 0 (DEFINITE_INCOMPATIBILITY), got %d", m.SeverityLevel)
			}
			if !strings.Contains(m.Message, "5.0mm") || !strings.Contains(m.Message, "1.5mm") {
				t.Errorf("Message does not mention shaft sizes: %s", m.Message)
			}
		}
	}
	if !found {
		t.Errorf("Expected MotorPropellerShaftChecker message for 5.0mm vs 1.5mm, found none")
	}

	// 3. Unspecified shaft (0.0 on motor or prop) -> No error
	compUnspecified := &Components{
		Motor:     &pb.Motor{ShaftDiameterMm: 0.0},
		Propeller: &pb.Propeller{ShaftDiameterMm: 5.0},
	}
	msgsUnspecified := CheckCompatibility(compUnspecified)
	for _, m := range msgsUnspecified {
		if m.CheckerName == "MotorPropellerShaftChecker" {
			t.Errorf("Expected no incompatibility when motor shaft is unspecified, got: %v", m)
		}
	}
}

func TestGenerateCelFilter_ShaftDiameter(t *testing.T) {
	// Propellers filter when motor is selected
	propFilter := GenerateCelFilter("propellers", &Components{
		Motor: &pb.Motor{ShaftDiameterMm: 5.0},
	})
	if propFilter != "shaft_diameter_mm == 5.0" {
		t.Errorf("Expected 'shaft_diameter_mm == 5.0', got %q", propFilter)
	}

	// Propellers filter with frame and motor
	propFilterCombined := GenerateCelFilter("propellers", &Components{
		Frame: &pb.Frame{MaxPropSizeMm: 127.0},
		Motor: &pb.Motor{ShaftDiameterMm: 5.0},
	})
	expectedCombined := "diameter_mm <= 127.0 && shaft_diameter_mm == 5.0"
	if propFilterCombined != expectedCombined {
		t.Errorf("Expected %q, got %q", expectedCombined, propFilterCombined)
	}

	// Motors filter when propeller is selected
	motorFilter := GenerateCelFilter("motors", &Components{
		Propeller: &pb.Propeller{ShaftDiameterMm: 1.5},
	})
	if motorFilter != "shaft_diameter_mm == 1.5" {
		t.Errorf("Expected 'shaft_diameter_mm == 1.5', got %q", motorFilter)
	}
}

func TestMotorFrameMountCompatibility(t *testing.T) {
	t.Run("Matching mount pattern", func(t *testing.T) {
		comp := &Components{
			Motor: &pb.Motor{MountPattern: "16x16mm"},
			Frame: &pb.Frame{MotorMountPatterns: []string{"16x16mm", "19x19mm"}},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "MotorFrameMountChecker" {
				t.Errorf("Unexpected MotorFrameMountChecker message for matching pattern: %s", m.Message)
			}
		}
	})

	t.Run("Mismatched mount pattern", func(t *testing.T) {
		comp := &Components{
			Motor: &pb.Motor{MountPattern: "12x12mm"},
			Frame: &pb.Frame{MotorMountPatterns: []string{"16x16mm", "19x19mm"}},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "MotorFrameMountChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected MotorFrameMountChecker message, got none")
		}
	})
}

func TestCameraFrameMountCompatibility(t *testing.T) {
	t.Run("Camera fits frame width", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountWidthMm: 19.0},
			Cameras: []*pb.Camera{{WidthMm: 19}},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "CameraFrameMountChecker" {
				t.Errorf("Unexpected CameraFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Camera exceeds frame width", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountWidthMm: 14.0},
			Cameras: []*pb.Camera{{WidthMm: 19}},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "CameraFrameMountChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected CameraFrameMountChecker message, got none")
		}
	})
}

func TestFlightControllerVtxCompatibility(t *testing.T) {
	t.Run("Voltage and Port Compatible", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				VtxVoltageMinV: 7.4,
				VtxVoltageMaxV: 12.0,
				VtxPorts:       []string{"SH1.0-6P", "Solder Pads"},
			},
			VideoTransmitter: &pb.VideoTransmitter{
				InputVoltageMinV: 7.0,
				InputVoltageMaxV: 26.0,
				PortType:         "SH1.0-6P",
			},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerVtxVoltageChecker" || m.CheckerName == "FlightControllerVtxPortChecker" {
				t.Errorf("Unexpected FC-VTX compatibility message: %s", m.Message)
			}
		}
	})

	t.Run("FC Max Voltage below VTX Min Voltage", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				VtxVoltageMinV: 5.0,
				VtxVoltageMaxV: 5.0,
			},
			VideoTransmitter: &pb.VideoTransmitter{
				InputVoltageMinV: 7.4,
				InputVoltageMaxV: 26.0,
			},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerVtxVoltageChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected FlightControllerVtxVoltageChecker message, got none")
		}
	})

	t.Run("FC Min Voltage above VTX Max Voltage", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				VtxVoltageMinV: 12.0,
				VtxVoltageMaxV: 12.0,
			},
			VideoTransmitter: &pb.VideoTransmitter{
				InputVoltageMinV: 3.7,
				InputVoltageMaxV: 5.0,
			},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerVtxVoltageChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected FlightControllerVtxVoltageChecker message, got none")
		}
	})

	t.Run("Port Mismatch", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				VtxPorts: []string{"SH1.0-6P"},
			},
			VideoTransmitter: &pb.VideoTransmitter{
				PortType: "JST-GH-4P",
			},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerVtxPortChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected FlightControllerVtxPortChecker message, got none")
		}
	})

	t.Run("Solder Pads Fallback Matches Any Port", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				VtxPorts: []string{"Solder Pads"},
			},
			VideoTransmitter: &pb.VideoTransmitter{
				PortType: "Direct Solder",
			},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerVtxPortChecker" {
				t.Errorf("Unexpected FlightControllerVtxPortChecker message when FC has solder pads: %s", m.Message)
			}
		}
	})
}

func TestFlightControllerFrameMountCompatibility(t *testing.T) {
	t.Run("Compatible Board Mount Pattern", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"30.5x30.5mm", "20x20mm"},
			},
			FlightController: &pb.FlightController{
				MountPattern: "30.5x30.5mm",
			},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerFrameMountChecker" {
				t.Errorf("Unexpected FlightControllerFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Incompatible Board Mount Pattern", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"20x20mm", "25.5x25.5mm"},
			},
			FlightController: &pb.FlightController{
				MountPattern: "30.5x30.5mm",
			},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerFrameMountChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected FlightControllerFrameMountChecker message, got none")
		}
	})
}

func TestVideoTransmitterFrameMountCompatibility(t *testing.T) {
	t.Run("Compatible Board Mount Pattern", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"30.5x30.5mm", "20x20mm"},
			},
			VideoTransmitter: &pb.VideoTransmitter{
				MountPattern: "20x20mm",
			},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "VideoTransmitterFrameMountChecker" {
				t.Errorf("Unexpected VideoTransmitterFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Incompatible Board Mount Pattern", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"25.5x25.5mm"},
			},
			VideoTransmitter: &pb.VideoTransmitter{
				MountPattern: "30.5x30.5mm",
			},
		}
		msgs := CheckCompatibility(comp)
		found := false
		for _, m := range msgs {
			if m.CheckerName == "VideoTransmitterFrameMountChecker" {
				found = true
				if m.SeverityLevel != 0 {
					t.Errorf("Expected DEFINITE_INCOMPATIBILITY (0), got %d", m.SeverityLevel)
				}
			}
		}
		if !found {
			t.Errorf("Expected VideoTransmitterFrameMountChecker message, got none")
		}
	})
}

func TestBoardMountCelFilters(t *testing.T) {
	t.Run("Flight Controller CEL Filter with Frame Board Mount Patterns", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"30.5x30.5mm", "20x20mm"},
			},
		}
		fcFilter := GenerateCelFilter("flight_controllers", comp)
		expected := "(mount_pattern == \"30.5x30.5mm\" || mount_pattern == \"20x20mm\")"
		if fcFilter != expected {
			t.Errorf("expected %q, got %q", expected, fcFilter)
		}
	})

	t.Run("Video Transmitter CEL Filter with Frame Board Mount Patterns and FC", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"20x20mm"},
			},
			FlightController: &pb.FlightController{
				VtxVoltageMaxV: 9.0,
				VtxVoltageMinV: 5.0,
				VtxPorts:       []string{"SH1.0-6P"},
			},
		}
		vtxFilter := GenerateCelFilter("video_transmitters", comp)
		expected := "input_voltage_min_v <= 9.0 && input_voltage_max_v >= 5.0 && port_type == \"SH1.0-6P\" && mount_pattern == \"20x20mm\""
		if vtxFilter != expected {
			t.Errorf("expected %q, got %q", expected, vtxFilter)
		}
	})
}

func TestFlightControllerElectronicSpeedControllerVoltageCompatibility(t *testing.T) {
	t.Run("ESC requires higher min voltage than FC max voltage -> Incompatible", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 3.0,
				MaxVoltage: 4.4, // 1S Whoop FC
			},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{
					MinVoltage: 7.4,  // Requires at least 2S
					MaxVoltage: 25.2, // 6S ESC
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "FlightControllerElectronicSpeedControllerVoltageChecker" {
				found = true
				if m.SeverityName != "DEFINITE_INCOMPATIBILITY" {
					t.Errorf("expected DEFINITE_INCOMPATIBILITY, got %s", m.SeverityName)
				}
				if !strings.Contains(m.Message, "Electronic speed controller minimum operating voltage") {
					t.Errorf("unexpected message: %s", m.Message)
				}
			}
		}
		if !found {
			t.Errorf("expected FlightControllerElectronicSpeedControllerVoltageChecker message, got none")
		}
	})

	t.Run("ESC max voltage is below FC min voltage -> Incompatible", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 14.8, // 4S minimum
				MaxVoltage: 25.2,
			},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{
					MinVoltage: 3.7,
					MaxVoltage: 8.4, // 2S max
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "FlightControllerElectronicSpeedControllerVoltageChecker" {
				found = true
				if m.SeverityName != "DEFINITE_INCOMPATIBILITY" {
					t.Errorf("expected DEFINITE_INCOMPATIBILITY, got %s", m.SeverityName)
				}
			}
		}
		if !found {
			t.Errorf("expected FlightControllerElectronicSpeedControllerVoltageChecker message, got none")
		}
	})

	t.Run("Overlapping voltages -> Compatible", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 11.1,
				MaxVoltage: 25.2,
			},
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{
					MinVoltage:       14.8,
					MaxVoltage:       30.0,
					MotorCurrentMaxA: 45.0,
				},
			},
		}
		messages := CheckCompatibility(comp)
		for _, m := range messages {
			if m.CheckerName == "FlightControllerElectronicSpeedControllerVoltageChecker" {
				t.Errorf("unexpected incompatibility message: %s", m.Message)
			}
		}
	})

	t.Run("GenerateCelFilter for ESC with FC selected", func(t *testing.T) {
		comp := &Components{
			FlightController: &pb.FlightController{
				MinVoltage: 11.1,
				MaxVoltage: 25.2,
			},
		}
		escFilter := GenerateCelFilter("electronic_speed_controllers", comp)
		expected := "min_voltage <= 25.20 && max_voltage >= 11.10"
		if escFilter != expected {
			t.Errorf("expected %q, got %q", expected, escFilter)
		}
	})

	t.Run("GenerateCelFilter for FC with ESC selected", func(t *testing.T) {
		comp := &Components{
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{
					MinVoltage: 14.8,
					MaxVoltage: 25.2,
				},
			},
		}
		fcFilter := GenerateCelFilter("flight_controllers", comp)
		expected := "min_voltage <= 25.20 && max_voltage >= 14.80"
		if fcFilter != expected {
			t.Errorf("expected %q, got %q", expected, fcFilter)
		}
	})
}
