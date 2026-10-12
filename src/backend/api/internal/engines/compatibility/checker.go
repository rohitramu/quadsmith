package compatibility

import (
	"fmt"
	"strings"

	pb "quadsmith/api/gen/quadsmith"
)

// Components represents a resolved set of hardware components to check for compatibility.
type Components = pb.AssembledComponents

// CheckCompatibility runs all compatibility rules against a given set of hardware components.
func CheckCompatibility(comp *Components) []*pb.CompatibilityMessage {
	var messages []*pb.CompatibilityMessage

	// Rule 1: Motor vs Electronic Speed Controller
	if comp.Motor != nil && len(comp.ElectronicSpeedControllers) > 0 {
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc.MotorCurrentMaxA < 20 && comp.Motor.StatorDiameterMm > 20 {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "MotorElectronicSpeedControllerElectricalChecker",
					Components:    "Motor and Electronic Speed Controller",
					SeverityLevel: 1,
					SeverityName:  "HEURISTIC_INCOMPATIBILITY",
					Message:       "Electronic Speed Controller continuous current rating may be too low for the selected motor size.",
					Resolution:    "Select an Electronic Speed Controller with a higher continuous current rating (e.g., >30A).",
				})
			}
		}
	}

	// Rule 2: Propeller vs Frame
	if comp.Propeller != nil && comp.Frame != nil {
		if float32(comp.Propeller.DiameterMm) > float32(comp.Frame.MaxPropSizeMm) {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "PropellerFrameChecker",
				Components:    "Propeller and Frame",
				SeverityLevel: 0,
				SeverityName:  "DEFINITE_INCOMPATIBILITY",
				Message:       "Propeller diameter exceeds the maximum supported size for this frame.",
				Resolution:    "Select a smaller propeller or a larger frame.",
			})
		}
	}

	// Rule 3: Camera vs Video Transmitter Protocol
	if comp.VideoTransmitter != nil && len(comp.Cameras) > 0 {
		vtxProto := strings.TrimSpace(comp.VideoTransmitter.Protocol)
		camProto := strings.TrimSpace(comp.Cameras[0].Protocol)
		if vtxProto != "" && camProto != "" && !strings.EqualFold(vtxProto, camProto) {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "CameraVideoTransmitterProtocolChecker",
				Components:    "Camera and Video Transmitter",
				SeverityLevel: 0,
				SeverityName:  "DEFINITE_INCOMPATIBILITY",
				Message:       fmt.Sprintf("Camera protocol (%s) is incompatible with Video Transmitter protocol (%s).", camProto, vtxProto),
				Resolution:    "Choose a camera and video transmitter that share the same transmission protocol (e.g. Analog, DJI O3, DJI O4 Pro, DJI O4 Lite, Walksnail, or HDZero).",
			})
		}
	}

	// Rule 4: Component-to-Component Voltage Conflict
	minV, maxV, maxA := CalculateElectricalLimits(comp)
	if minV > 0 && maxV > 0 && minV > maxV {
		messages = append(messages, &pb.CompatibilityMessage{
			CheckerName:   "ComponentVoltageChecker",
			Components:    "Flight Controller, Electronic Speed Controller, and Motor",
			SeverityLevel: 0,
			SeverityName:  "DEFINITE_INCOMPATIBILITY",
			Message:       fmt.Sprintf("Build electrical conflict: Minimum component voltage requirement (%.1fV) exceeds maximum component voltage rating (%.1fV).", minV, maxV),
			Resolution:    "Select motors, flight controllers, and ESCs with overlapping operating voltage ranges.",
		})
	}

	// Rule 5: ESC Channels vs Frame Motor Count
	if comp.Frame != nil && comp.Frame.MotorCount > 0 && len(comp.ElectronicSpeedControllers) > 0 {
		var totalEscChannels uint32 = 0
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc != nil {
				if esc.MaxMotors > 0 {
					totalEscChannels += esc.MaxMotors
				} else {
					totalEscChannels += 1
				}
			}
		}
		if totalEscChannels < comp.Frame.MotorCount {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "EscChannelCountChecker",
				Components:    "Frame and Electronic Speed Controller",
				SeverityLevel: 1,
				SeverityName:  "HEURISTIC_INCOMPATIBILITY",
				Message:       fmt.Sprintf("Insufficient ESC channels: Frame requires %d ESC channels, but only %d configured.", comp.Frame.MotorCount, totalEscChannels),
				Resolution:    fmt.Sprintf("Select an ESC or combination of ESCs supporting at least %d motor channels.", comp.Frame.MotorCount),
			})
		}
	}

	// Rule 6: Battery Compatibility (Voltage, Current, Connector)
	if comp.Battery != nil {
		bat := comp.Battery
		if maxV > 0 && bat.MaxVoltage > maxV {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "BatteryMaxVoltageChecker",
				Components:    "Battery and Build Components",
				SeverityLevel: 0,
				SeverityName:  "DEFINITE_INCOMPATIBILITY",
				Message:       fmt.Sprintf("Battery max voltage (%.1fV) exceeds build component limit (%.1fV).", bat.MaxVoltage, maxV),
				Resolution:    "Select a battery with lower cell count or components rated for higher voltage.",
			})
		}
		if minV > 0 && bat.MinVoltage < minV {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "BatteryMinVoltageChecker",
				Components:    "Battery and Build Components",
				SeverityLevel: 1,
				SeverityName:  "HEURISTIC_INCOMPATIBILITY",
				Message:       fmt.Sprintf("Battery min voltage (%.1fV) is below build minimum operating voltage (%.1fV).", bat.MinVoltage, minV),
				Resolution:    "Select a battery with higher cell count to match the build's minimum operating voltage.",
			})
		}
		if maxA > 0 && bat.MaxCurrentA > 0 && bat.MaxCurrentA < maxA {
			messages = append(messages, &pb.CompatibilityMessage{
				CheckerName:   "BatteryMaxCurrentChecker",
				Components:    "Battery and Motors/ESCs",
				SeverityLevel: 1,
				SeverityName:  "HEURISTIC_INCOMPATIBILITY",
				Message:       fmt.Sprintf("Battery max current (%.1fA) is below build recommended current rating (%.1fA).", bat.MaxCurrentA, maxA),
				Resolution:    "Select a battery with a higher continuous discharge C-rating or higher capacity.",
			})
		}
		if comp.FlightController != nil && comp.FlightController.MaxVoltage > 0 {
			if bat.MinVoltage > comp.FlightController.MaxVoltage {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "BatteryFlightControllerVoltageChecker",
					Components:    "Battery and Flight Controller",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Battery minimum voltage (%.1fV) exceeds Flight Controller maximum voltage (%.1fV).", bat.MinVoltage, comp.FlightController.MaxVoltage),
					Resolution:    "Select a battery with lower cell count or an FC rated for higher input voltage.",
				})
			}
		}
		if len(comp.ElectronicSpeedControllers) > 0 {
			for _, esc := range comp.ElectronicSpeedControllers {
				if esc != nil && esc.MaxVoltage > 0 && bat.MinVoltage > esc.MaxVoltage {
					messages = append(messages, &pb.CompatibilityMessage{
						CheckerName:   "BatteryEscVoltageChecker",
						Components:    "Battery and Electronic Speed Controller",
						SeverityLevel: 0,
						SeverityName:  "DEFINITE_INCOMPATIBILITY",
						Message:       fmt.Sprintf("Battery minimum voltage (%.1fV) exceeds Electronic Speed Controller maximum voltage (%.1fV).", bat.MinVoltage, esc.MaxVoltage),
						Resolution:    "Select a battery with lower cell count or an ESC rated for higher voltage.",
					})
				}
			}
		}

		// Battery Connector Compatibility
		primaryConn, compatConns := DetermineBatteryConnector(comp, minV, maxV, maxA)
		if len(compatConns) > 0 && bat.Connector != "" {
			matched := false
			for _, c := range compatConns {
				if strings.EqualFold(bat.Connector, c) {
					matched = true
					break
				}
			}
			if !matched {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "BatteryConnectorChecker",
					Components:    "Battery and Build Connector",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Battery connector (%s) is incompatible with build connector (%s).", bat.Connector, primaryConn),
					Resolution:    fmt.Sprintf("Select a battery with a compatible connector (%s).", strings.Join(compatConns, ", ")),
				})
			}
		}
	}

	return messages
}

// CalculateElectricalLimits calculates the operating min/max voltage and max current from components.
func CalculateElectricalLimits(comp *Components) (minV, maxV, maxA float32) {
	if comp == nil {
		return 0, 0, 0
	}
	if comp.FlightController != nil && comp.FlightController.MinVoltage > minV {
		minV = comp.FlightController.MinVoltage
	}
	for _, esc := range comp.ElectronicSpeedControllers {
		if esc != nil && esc.MinVoltage > minV {
			minV = esc.MinVoltage
		}
	}
	if comp.Motor != nil && comp.Motor.MinVoltage > minV {
		minV = comp.Motor.MinVoltage
	}

	updateMaxV := func(v float32) {
		if v > 0 && (maxV == 0 || v < maxV) {
			maxV = v
		}
	}
	if comp.FlightController != nil {
		updateMaxV(comp.FlightController.MaxVoltage)
	}
	for _, esc := range comp.ElectronicSpeedControllers {
		if esc != nil {
			updateMaxV(esc.MaxVoltage)
		}
	}
	if comp.Motor != nil {
		updateMaxV(comp.Motor.MaxVoltage)
	}

	mc := uint32(4)
	if comp.Frame != nil && comp.Frame.MotorCount > 0 {
		mc = comp.Frame.MotorCount
	}
	if comp.Motor != nil && comp.Motor.MaxCurrentA > 0 {
		maxA = comp.Motor.MaxCurrentA * float32(mc)
	} else {
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc != nil {
				mult := float32(esc.MaxMotors)
				if mult == 0 {
					mult = 1.0
				}
				maxA += esc.MotorCurrentMaxA * mult
			}
		}
	}
	return minV, maxV, maxA
}

func textMentionsConnector(text, connector string) bool {
	lowerText := strings.ToLower(text)
	switch connector {
	case "BT2.0":
		return strings.Contains(lowerText, "bt2.0") || strings.Contains(lowerText, "bt 2.0") || strings.Contains(lowerText, "bt-2.0")
	case "PH2.0":
		return strings.Contains(lowerText, "ph2.0") || strings.Contains(lowerText, "ph 2.0") || strings.Contains(lowerText, "ph-2.0")
	case "A30":
		for _, pattern := range []string{" a30", "a30 ", "(a30)", "-a30", "/a30"} {
			if strings.Contains(lowerText, pattern) {
				return true
			}
		}
		return strings.HasPrefix(lowerText, "a30 ") || strings.HasSuffix(lowerText, " a30") || lowerText == "a30"
	default:
		return strings.Contains(lowerText, strings.ToLower(connector))
	}
}

// DetermineBatteryConnector derives the recommended primary connector and compatible connectors
// based on component metadata, electrical limits, and frame size.
func DetermineBatteryConnector(comp *Components, minV, maxV, maxA float32) (string, []string) {
	// 1. Check explicit component text mentions
	var texts []string
	if comp != nil {
		if comp.FlightController != nil {
			texts = append(texts, comp.FlightController.Name, comp.FlightController.Description)
		}
		if comp.Frame != nil {
			texts = append(texts, comp.Frame.Name, comp.Frame.Description)
		}
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc != nil {
				texts = append(texts, esc.Name, esc.Description)
			}
		}
		if comp.Motor != nil {
			texts = append(texts, comp.Motor.Name, comp.Motor.Description)
		}
	}

	for _, text := range texts {
		if text == "" {
			continue
		}
		if textMentionsConnector(text, "BT2.0") {
			return "BT2.0", []string{"BT2.0", "A30", "PH2.0"}
		}
		if textMentionsConnector(text, "A30") {
			return "A30", []string{"A30", "BT2.0", "PH2.0"}
		}
		if textMentionsConnector(text, "PH2.0") {
			return "PH2.0", []string{"PH2.0", "BT2.0", "A30"}
		}
		if textMentionsConnector(text, "XT30") {
			return "XT30", []string{"XT30"}
		}
		if textMentionsConnector(text, "XT90") {
			return "XT90", []string{"XT90", "XT60"}
		}
		if textMentionsConnector(text, "XT60") {
			return "XT60", []string{"XT60"}
		}
	}

	// 2. Derive from physical frame properties if electrical specs are not yet set
	if maxV == 0 && maxA == 0 && comp != nil && comp.Frame != nil {
		if comp.Frame.WheelbaseMm > 0 && comp.Frame.WheelbaseMm <= 85 {
			return "BT2.0", []string{"BT2.0", "A30", "PH2.0"}
		}
		if (comp.Frame.WheelbaseMm > 0 && comp.Frame.WheelbaseMm <= 160) || (comp.Frame.MaxPropSizeMm > 0 && comp.Frame.MaxPropSizeMm <= 90) {
			return "XT30", []string{"XT30"}
		}
		if comp.Frame.MotorCount >= 8 || comp.Frame.MaxPropSizeMm >= 203.2 {
			return "XT90", []string{"XT90", "XT60"}
		}
		if comp.Frame.WheelbaseMm > 160 || comp.Frame.MaxPropSizeMm > 90 {
			return "XT60", []string{"XT60"}
		}
	}

	// 3. Derive from electrical ratings and frame characteristics
	// 1S Whoop / Toothpick Class
	if maxV > 0 && maxV <= 4.4 {
		return "BT2.0", []string{"BT2.0", "A30", "PH2.0"}
	}

	// Toothpick / Micro 2S-4S Sub-250 Class (XT30)
	if (maxA > 0 && maxA <= 45.0) || (maxV > 0 && maxV <= 17.5 && comp != nil && comp.Frame != nil && ((comp.Frame.WheelbaseMm > 0 && comp.Frame.WheelbaseMm <= 160) || (comp.Frame.MaxPropSizeMm > 0 && comp.Frame.MaxPropSizeMm <= 90))) {
		return "XT30", []string{"XT30"}
	}

	// Heavy Lifter / Cinelifter / Macro Class (XT90)
	if maxA > 180.0 || (comp != nil && comp.Frame != nil && (comp.Frame.MotorCount >= 8 || comp.Frame.MaxPropSizeMm >= 203.2)) {
		return "XT90", []string{"XT90", "XT60"}
	}

	// Standard Quadcopter Class (XT60)
	if minV > 0 || maxV > 0 || maxA > 0 {
		return "XT60", []string{"XT60"}
	}

	return "", nil
}

// GenerateCelFilter creates a CEL query filter for a target hardware collection based on selected components.
func GenerateCelFilter(targetCollection string, comp *Components) string {
	if comp == nil {
		return ""
	}

	col := strings.TrimPrefix(targetCollection, "components/hardware/")
	col = strings.TrimPrefix(col, "hardware/")
	col = strings.TrimPrefix(col, "/")

	var conditions []string

	switch col {
	case "propellers":
		if comp.Frame != nil && comp.Frame.MaxPropSizeMm > 0 {
			conditions = append(conditions, fmt.Sprintf("diameter_mm <= %.1f", comp.Frame.MaxPropSizeMm))
		}
	case "frames":
		if comp.Propeller != nil && comp.Propeller.DiameterMm > 0 {
			conditions = append(conditions, fmt.Sprintf("max_prop_size_mm >= %.1f", comp.Propeller.DiameterMm))
		}
	case "batteries":
		minV, maxV, maxA := CalculateElectricalLimits(comp)
		if minV > 0 && maxV > 0 && minV > maxV {
			return ""
		}

		if minV > 0 {
			conditions = append(conditions, fmt.Sprintf("min_voltage >= %.2f", minV))
		}
		if maxV > 0 {
			conditions = append(conditions, fmt.Sprintf("max_voltage <= %.2f", maxV))
		}
		if maxA > 0 {
			conditions = append(conditions, fmt.Sprintf("max_current_a >= %.2f", maxA))
		}

		_, compatConns := DetermineBatteryConnector(comp, minV, maxV, maxA)
		if len(compatConns) == 1 {
			conditions = append(conditions, fmt.Sprintf("connector == %q", compatConns[0]))
		} else if len(compatConns) > 1 {
			var connParts []string
			for _, c := range compatConns {
				connParts = append(connParts, fmt.Sprintf("connector == %q", c))
			}
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(connParts, " || ")))
		}
	case "electronic_speed_controllers":
		if comp.Motor != nil && comp.Motor.StatorDiameterMm > 20 {
			conditions = append(conditions, "motor_current_max_a >= 20.0")
		}
		if comp.Motor != nil && comp.Motor.MinVoltage > 0 {
			conditions = append(conditions, fmt.Sprintf("max_voltage >= %.2f", comp.Motor.MinVoltage))
		}
	case "motors":
		if len(comp.ElectronicSpeedControllers) > 0 {
			var minEscCurrent float32 = 0
			var minEscMaxV float32 = 0
			for _, esc := range comp.ElectronicSpeedControllers {
				if esc != nil {
					if esc.MotorCurrentMaxA > 0 && (minEscCurrent == 0 || esc.MotorCurrentMaxA < minEscCurrent) {
						minEscCurrent = esc.MotorCurrentMaxA
					}
					if esc.MaxVoltage > 0 && (minEscMaxV == 0 || esc.MaxVoltage < minEscMaxV) {
						minEscMaxV = esc.MaxVoltage
					}
				}
			}
			if minEscCurrent > 0 {
				conditions = append(conditions, fmt.Sprintf("max_current_a <= %.1f", minEscCurrent))
			}
			if minEscMaxV > 0 {
				conditions = append(conditions, fmt.Sprintf("min_voltage <= %.2f", minEscMaxV))
			}
		}
	case "cameras":
		if comp.VideoTransmitter != nil && comp.VideoTransmitter.Protocol != "" {
			conditions = append(conditions, fmt.Sprintf("protocol == %q", comp.VideoTransmitter.Protocol))
		}
	case "video_transmitters":
		if len(comp.Cameras) > 0 && comp.Cameras[0].Protocol != "" {
			conditions = append(conditions, fmt.Sprintf("protocol == %q", comp.Cameras[0].Protocol))
		}
	}

	return strings.Join(conditions, " && ")
}
