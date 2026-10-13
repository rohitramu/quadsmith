package compatibility

import (
	"strings"
	"testing"

	pb "quadsmith/api/gen/quadsmith/api"
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
	expectedEsc := "max_motors >= 4 && min_voltage <= 25.20 && max_voltage >= 14.80"
	if escFilter != expectedEsc {
		t.Errorf("expected %q, got %q", expectedEsc, escFilter)
	}

	frameFilter := GenerateCelFilter("frames", &Components{
		Propeller: &pb.Propeller{DiameterMm: 130.0},
	})
	if frameFilter != "" {
		t.Errorf("expected empty frame filter, got %q", frameFilter)
	}

	camFilter := GenerateCelFilter("cameras", &Components{
		Frame: &pb.Frame{CameraMountInterfaces: []string{"20mm (DJI O3)"}},
	})
	if camFilter != `camera_mount_interface == "20mm (DJI O3)"` {
		t.Errorf(`expected 'camera_mount_interface == "20mm (DJI O3)"', got %q`, camFilter)
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
		t.Errorf(`expected motor filter with ESC current limit, got %q`, motorFilter)
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

	// In unidirectional flow, ESC (Step 5) does not filter on Motor (Step 6)
	escFilter := GenerateCelFilter("electronic_speed_controllers", &Components{
		Motor: &pb.Motor{MinVoltage: 18.0},
	})
	if escFilter != "" {
		t.Errorf("Expected empty ESC filter when motor selected, got %q", escFilter)
	}

	// Motors (Step 6) filter based on ESC (Step 5)
	motorFilter := GenerateCelFilter("motors", &Components{
		ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
			{MaxVoltage: 17.4},
		},
	})
	if motorFilter != "min_voltage <= 17.40" {
		t.Errorf("Expected 'min_voltage <= 17.40', got %q", motorFilter)
	}
}

func TestCheckCompatibility_MotorPropellerMount(t *testing.T) {
	// 1. Matching interface (M5 motor with M5 prop) -> Compatible
	compMatch := &Components{
		Motor:     &pb.Motor{PropMountInterfaces: []string{"M5"}},
		Propeller: &pb.Propeller{PropMountInterface: "M5"},
	}
	msgs := CheckCompatibility(compMatch)
	for _, m := range msgs {
		if m.CheckerName == "MotorPropellerMountChecker" {
			t.Errorf("Expected no mount incompatibility for matching M5, got: %v", m)
		}
	}

	// 2. Multi-mount motor (1.5mm T-Mount + 1.5mm Press-Fit) with 1.5mm T-Mount prop -> Compatible
	compMulti := &Components{
		Motor:     &pb.Motor{PropMountInterfaces: []string{"1.5mm T-Mount", "1.5mm Press-Fit"}},
		Propeller: &pb.Propeller{PropMountInterface: "1.5mm T-Mount"},
	}
	msgsMulti := CheckCompatibility(compMulti)
	for _, m := range msgsMulti {
		if m.CheckerName == "MotorPropellerMountChecker" {
			t.Errorf("Expected no mount incompatibility for multi-mount motor, got: %v", m)
		}
	}

	// 3. Mismatched interfaces (M5 motor with 1.5mm T-Mount prop) -> Definite Incompatibility
	compMismatch := &Components{
		Motor:     &pb.Motor{PropMountInterfaces: []string{"M5"}},
		Propeller: &pb.Propeller{PropMountInterface: "1.5mm T-Mount"},
	}
	msgsMismatch := CheckCompatibility(compMismatch)
	found := false
	for _, m := range msgsMismatch {
		if m.CheckerName == "MotorPropellerMountChecker" {
			found = true
			if m.SeverityLevel != 0 {
				t.Errorf("Expected SeverityLevel 0 (DEFINITE_INCOMPATIBILITY), got %d", m.SeverityLevel)
			}
			if !strings.Contains(m.Message, "M5") || !strings.Contains(m.Message, "1.5mm T-Mount") {
				t.Errorf("Message does not mention mount interfaces: %s", m.Message)
			}
		}
	}
	if !found {
		t.Errorf("Expected MotorPropellerMountChecker message for M5 vs 1.5mm T-Mount, found none")
	}

	// 4. Unspecified interface -> No error
	compUnspecified := &Components{
		Motor:     &pb.Motor{PropMountInterfaces: nil},
		Propeller: &pb.Propeller{PropMountInterface: "M5"},
	}
	msgsUnspecified := CheckCompatibility(compUnspecified)
	for _, m := range msgsUnspecified {
		if m.CheckerName == "MotorPropellerMountChecker" {
			t.Errorf("Expected no incompatibility when motor mount is unspecified, got: %v", m)
		}
	}
}

func TestGenerateCelFilter_PropMountInterface(t *testing.T) {
	// Propellers filter when single-mount motor is selected
	propFilterSingle := GenerateCelFilter("propellers", &Components{
		Motor: &pb.Motor{PropMountInterfaces: []string{"M5"}},
	})
	if propFilterSingle != `prop_mount_interface == "M5"` {
		t.Errorf("Expected 'prop_mount_interface == \"M5\"', got %q", propFilterSingle)
	}

	// Propellers filter when multi-mount motor is selected
	propFilterMulti := GenerateCelFilter("propellers", &Components{
		Motor: &pb.Motor{PropMountInterfaces: []string{"1.5mm T-Mount", "1.5mm Press-Fit"}},
	})
	expectedMulti := `(prop_mount_interface == "1.5mm T-Mount" || prop_mount_interface == "1.5mm Press-Fit")`
	if propFilterMulti != expectedMulti {
		t.Errorf("Expected %q, got %q", expectedMulti, propFilterMulti)
	}

	// Propellers filter with frame and motor
	propFilterCombined := GenerateCelFilter("propellers", &Components{
		Frame: &pb.Frame{MaxPropSizeMm: 127.0},
		Motor: &pb.Motor{PropMountInterfaces: []string{"M5"}},
	})
	expectedCombined := `diameter_mm <= 127.0 && prop_mount_interface == "M5"`
	if propFilterCombined != expectedCombined {
		t.Errorf("Expected %q, got %q", expectedCombined, propFilterCombined)
	}

	// Motors are NOT filtered when propeller is selected
	motorFilter := GenerateCelFilter("motors", &Components{
		Propeller: &pb.Propeller{PropMountInterface: "1.5mm T-Mount"},
	})
	if motorFilter != "" {
		t.Errorf("Expected empty motor filter when propeller is selected, got %q", motorFilter)
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
	t.Run("Camera fits frame mount interface", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountInterfaces: []string{"19mm (Micro)", "20mm (DJI O3)"}},
			Cameras: []*pb.Camera{{CameraMountInterface: "19mm (Micro)"}},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "CameraFrameMountChecker" {
				t.Errorf("Unexpected CameraFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Camera size normalized matches (19mm vs 19mm (Micro))", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountInterfaces: []string{"19mm (Micro)"}},
			Cameras: []*pb.Camera{{CameraMountInterface: "19mm"}},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "CameraFrameMountChecker" {
				t.Errorf("Unexpected CameraFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Camera size normalized matches (20mm vs 20mm (DJI O3))", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountInterfaces: []string{"20mm"}},
			Cameras: []*pb.Camera{{CameraMountInterface: "20mm (DJI O3)"}},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "CameraFrameMountChecker" {
				t.Errorf("Unexpected CameraFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Camera does not match frame mount interface", func(t *testing.T) {
		comp := &Components{
			Frame:   &pb.Frame{CameraMountInterfaces: []string{"14mm (Nano)"}},
			Cameras: []*pb.Camera{{CameraMountInterface: "19mm (Micro)"}},
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

	t.Run("Normalized Board Mount Pattern (20x20 vs 20x20mm)", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"20x20mm"},
			},
			FlightController: &pb.FlightController{
				MountPattern: "20x20",
			},
		}
		msgs := CheckCompatibility(comp)
		for _, m := range msgs {
			if m.CheckerName == "FlightControllerFrameMountChecker" {
				t.Errorf("Unexpected FlightControllerFrameMountChecker message: %s", m.Message)
			}
		}
	})

	t.Run("Multi-mount Board Mount Pattern (20x20 / 25.5x25.5 vs 25.5x25.5mm)", func(t *testing.T) {
		comp := &Components{
			Frame: &pb.Frame{
				BoardMountPatterns: []string{"25.5x25.5mm"},
			},
			FlightController: &pb.FlightController{
				MountPattern: "20x20 / 25.5x25.5",
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

func TestEscFlightControllerVoltageFilter(t *testing.T) {
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

	t.Run("GenerateCelFilter for FC does not filter on ESC voltage", func(t *testing.T) {
		comp := &Components{
			ElectronicSpeedControllers: []*pb.ElectronicSpeedController{
				{
					MinVoltage: 14.8,
					MaxVoltage: 25.2,
				},
			},
		}
		fcFilter := GenerateCelFilter("flight_controllers", comp)
		if fcFilter != "" {
			t.Errorf("expected empty filter for FC when no frame selected, got %q", fcFilter)
		}
	})
}

func TestAntennaCompatibility(t *testing.T) {
	t.Run("RX 2.4GHz with 915MHz antenna -> Incompatible", func(t *testing.T) {
		comp := &Components{
			Receivers: []*pb.Receiver{
				{
					FrequencyBandMhz: 2400,
				},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 915,
					Connector:        "U.FL",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "ReceiverAntennaFrequencyChecker" {
				found = true
				if m.SeverityName != "DEFINITE_INCOMPATIBILITY" {
					t.Errorf("expected DEFINITE_INCOMPATIBILITY, got %s", m.SeverityName)
				}
			}
		}
		if !found {
			t.Errorf("expected ReceiverAntennaFrequencyChecker message, got none")
		}
	})

	t.Run("RX with SMA antenna -> Connector Incompatible", func(t *testing.T) {
		comp := &Components{
			Receivers: []*pb.Receiver{
				{
					FrequencyBandMhz: 2400,
				},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 2400,
					Connector:        "SMA",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "ReceiverAntennaConnectorChecker" {
				found = true
				if m.SeverityName != "DEFINITE_INCOMPATIBILITY" {
					t.Errorf("expected DEFINITE_INCOMPATIBILITY, got %s", m.SeverityName)
				}
			}
		}
		if !found {
			t.Errorf("expected ReceiverAntennaConnectorChecker message, got none")
		}
	})

	t.Run("RX 2.4GHz with 2.4GHz U.FL antenna -> Compatible", func(t *testing.T) {
		comp := &Components{
			Receivers: []*pb.Receiver{
				{
					FrequencyBandMhz: 2400,
				},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 2400,
					Connector:        "U.FL",
				},
			},
		}
		messages := CheckCompatibility(comp)
		for _, m := range messages {
			if strings.Contains(m.CheckerName, "Antenna") {
				t.Errorf("unexpected antenna incompatibility: %s", m.Message)
			}
		}
	})

	t.Run("DJI O3 with SMA antenna -> Connector Incompatible", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol: "DJI O3",
				Name:     "DJI O3 Air Unit",
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 5800,
					Connector:        "SMA",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "VideoTransmitterAntennaConnectorChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected VideoTransmitterAntennaConnectorChecker message, got none")
		}
	})

	t.Run("GenerateCelFilter for Antennas with 2.4GHz RX", func(t *testing.T) {
		comp := &Components{
			Receivers: []*pb.Receiver{
				{
					FrequencyBandMhz: 2400,
				},
			},
		}
		filter := GenerateCelFilter("antennas", comp)
		expected := "frequency_band_mhz == 2400 && (connector == \"U.FL\" || connector == \"IPEX\")"
		if filter != expected {
			t.Errorf("expected %q, got %q", expected, filter)
		}
	})

	t.Run("GenerateCelFilter for Antennas with DJI O3 VTX", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol: "DJI O3",
				Name:     "DJI O3 Air Unit",
			},
		}
		filter := GenerateCelFilter("antennas", comp)
		expected := "frequency_band_mhz == 5800 && (connector == \"Dual U.FL\" || connector == \"U.FL\")"
		if filter != expected {
			t.Errorf("expected %q, got %q", expected, filter)
		}
	})

	t.Run("VTX 5.8GHz with 2.4GHz antenna -> Frequency Incompatible", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol:          "Analog",
				Name:              "TBS Unify Pro32 HV",
				FrequencyBandMhz:  5800,
				AntennaConnectors: []string{"SMA"},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 2400,
					Connector:        "SMA",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "VideoTransmitterAntennaFrequencyChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected VideoTransmitterAntennaFrequencyChecker message, got none")
		}
	})

	t.Run("VTX SMA with U.FL antenna -> Connector Incompatible", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol:          "Analog",
				Name:              "TBS Unify Pro32 HV",
				FrequencyBandMhz:  5800,
				AntennaConnectors: []string{"SMA"},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 5800,
					Connector:        "U.FL",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "VideoTransmitterAntennaConnectorChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected VideoTransmitterAntennaConnectorChecker message, got none")
		}
	})

	t.Run("VTX 5.8GHz SMA with 5.8GHz SMA antenna -> Compatible", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol:          "Analog",
				Name:              "TBS Unify Pro32 HV",
				FrequencyBandMhz:  5800,
				AntennaConnectors: []string{"SMA"},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 5800,
					Connector:        "SMA",
				},
			},
		}
		messages := CheckCompatibility(comp)
		for _, m := range messages {
			if strings.HasPrefix(m.CheckerName, "VideoTransmitterAntenna") {
				t.Errorf("unexpected incompatibility: %s", m.Message)
			}
		}
	})

	t.Run("VTX 3.3GHz SMA with 5.8GHz SMA antenna -> Frequency Incompatible", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				Protocol:          "Analog",
				Name:              "3.3G 5W High-Power SMA VTX",
				FrequencyBandMhz:  3300,
				AntennaConnectors: []string{"SMA"},
			},
			Antennas: []*pb.Antenna{
				{
					FrequencyBandMhz: 5800,
					Connector:        "SMA",
				},
			},
		}
		messages := CheckCompatibility(comp)
		found := false
		for _, m := range messages {
			if m.CheckerName == "VideoTransmitterAntennaFrequencyChecker" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected VideoTransmitterAntennaFrequencyChecker message, got none")
		}
	})

	t.Run("GenerateCelFilter for vtx_antennas with SMA VTX", func(t *testing.T) {
		comp := &Components{
			VideoTransmitter: &pb.VideoTransmitter{
				FrequencyBandMhz:  5800,
				AntennaConnectors: []string{"SMA"},
			},
		}
		filter := GenerateCelFilter("vtx_antennas", comp)
		expected := "frequency_band_mhz == 5800 && connector == \"SMA\""
		if filter != expected {
			t.Errorf("expected %q, got %q", expected, filter)
		}
	})
}

func TestMountPatternNormalization(t *testing.T) {
	if !isMountPatternCompatible("20x20", []string{"20x20mm"}) {
		t.Errorf("expected 20x20 to be compatible with 20x20mm")
	}
	if !isMountPatternCompatible("20x20mm", []string{"20x20 / 25.5x25.5"}) {
		t.Errorf("expected 20x20mm to be compatible with 20x20 / 25.5x25.5")
	}
	if !isMountPatternCompatible("25.5x25.5", []string{"20x20 / 25.5x25.5"}) {
		t.Errorf("expected 25.5x25.5 to be compatible with 20x20 / 25.5x25.5")
	}
	if isMountPatternCompatible("30.5x30.5", []string{"20x20", "25.5x25.5"}) {
		t.Errorf("expected 30.5x30.5 to NOT be compatible with 20x20, 25.5x25.5")
	}
}

func TestCameraMountNormalization(t *testing.T) {
	if !isCameraMountCompatible("19mm", []string{"19mm (Micro)"}) {
		t.Errorf("expected 19mm to be compatible with 19mm (Micro)")
	}
	if !isCameraMountCompatible("19mm (Micro)", []string{"19mm", "20mm"}) {
		t.Errorf("expected 19mm (Micro) to be compatible with 19mm")
	}
	if !isCameraMountCompatible("20mm (DJI O3)", []string{"20mm"}) {
		t.Errorf("expected 20mm (DJI O3) to be compatible with 20mm")
	}
	if isCameraMountCompatible("14mm (Nano)", []string{"19mm (Micro)"}) {
		t.Errorf("expected 14mm (Nano) to NOT be compatible with 19mm (Micro)")
	}
}
