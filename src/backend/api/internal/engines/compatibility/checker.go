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

	// Rule 3: Motor vs Propeller Mount Compatibility
	if comp.Motor != nil && comp.Propeller != nil {
		if len(comp.Motor.PropMountInterfaces) > 0 && comp.Propeller.PropMountInterface != "" {
			matched := false
			for _, iface := range comp.Motor.PropMountInterfaces {
				if strings.EqualFold(strings.TrimSpace(iface), strings.TrimSpace(comp.Propeller.PropMountInterface)) {
					matched = true
					break
				}
			}
			if !matched {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "MotorPropellerMountChecker",
					Components:    "Motor and Propeller",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Motor propeller mount interfaces (%s) do not match propeller mount interface (%s).", strings.Join(comp.Motor.PropMountInterfaces, ", "), comp.Propeller.PropMountInterface),
					Resolution:    "Select a propeller with a matching mount interface or a motor supporting the propeller's mounting style.",
				})
			}
		}
	}

	// Rule 4: Motor vs Frame Mount Pattern
	if comp.Motor != nil && comp.Frame != nil {
		if comp.Motor.MountPattern != "" && len(comp.Frame.MotorMountPatterns) > 0 {
			matched := false
			for _, p := range comp.Frame.MotorMountPatterns {
				if strings.EqualFold(strings.TrimSpace(comp.Motor.MountPattern), strings.TrimSpace(p)) {
					matched = true
					break
				}
			}
			if !matched {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "MotorFrameMountChecker",
					Components:    "Motor and Frame",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Motor mounting pattern (%s) is not supported by frame motor mount holes (%s).", comp.Motor.MountPattern, strings.Join(comp.Frame.MotorMountPatterns, ", ")),
					Resolution:    "Select a motor matching the frame's mounting pattern or choose a frame with compatible motor mount holes.",
				})
			}
		}
	}

	// Rule 5: Camera vs Frame Camera Mount Width
	if comp.Frame != nil && len(comp.Cameras) > 0 {
		cam := comp.Cameras[0]
		if cam.WidthMm > 0 && comp.Frame.CameraMountWidthMm > 0 {
			if float32(cam.WidthMm) > comp.Frame.CameraMountWidthMm {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "CameraFrameMountChecker",
					Components:    "Camera and Frame",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Camera width (%dmm) exceeds frame camera mount width (%.0fmm).", cam.WidthMm, comp.Frame.CameraMountWidthMm),
					Resolution:    "Select a smaller camera (e.g. nano 14mm) or a frame supporting wider camera mounting.",
				})
			}
		}
	}

	// Rule 6: Camera vs Video Transmitter Protocol
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

	// Rule 8: Flight Controller vs Frame Board Mount Pattern
	if comp.FlightController != nil && comp.Frame != nil {
		if comp.FlightController.MountPattern != "" && len(comp.Frame.BoardMountPatterns) > 0 {
			matched := false
			for _, p := range comp.Frame.BoardMountPatterns {
				if strings.EqualFold(strings.TrimSpace(comp.FlightController.MountPattern), strings.TrimSpace(p)) {
					matched = true
					break
				}
			}
			if !matched {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "FlightControllerFrameMountChecker",
					Components:    "Flight Controller and Frame",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Flight controller mounting pattern (%s) is not supported by frame board mount holes (%s).", comp.FlightController.MountPattern, strings.Join(comp.Frame.BoardMountPatterns, ", ")),
					Resolution:    "Select a flight controller matching the frame's board mounting pattern or choose a frame with compatible mounting holes.",
				})
			}
		}
	}

	// Rule 9: Video Transmitter vs Frame Board Mount Pattern
	if comp.VideoTransmitter != nil && comp.Frame != nil {
		if comp.VideoTransmitter.MountPattern != "" && len(comp.Frame.BoardMountPatterns) > 0 {
			matched := false
			for _, p := range comp.Frame.BoardMountPatterns {
				if strings.EqualFold(strings.TrimSpace(comp.VideoTransmitter.MountPattern), strings.TrimSpace(p)) {
					matched = true
					break
				}
			}
			if !matched {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "VideoTransmitterFrameMountChecker",
					Components:    "Video Transmitter and Frame",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Video transmitter mounting pattern (%s) is not supported by frame board mount holes (%s).", comp.VideoTransmitter.MountPattern, strings.Join(comp.Frame.BoardMountPatterns, ", ")),
					Resolution:    "Select a video transmitter matching the frame's board mounting pattern or choose a frame with compatible mounting holes.",
				})
			}
		}
	}

	// Rule 12: Motor vs Electronic Speed Controller Electrical / Voltage Compatibility
	if comp.Motor != nil && len(comp.ElectronicSpeedControllers) > 0 {
		m := comp.Motor
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc == nil {
				continue
			}
			if esc.MaxVoltage > 0 && m.MinVoltage > 0 && m.MinVoltage > esc.MaxVoltage {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "MotorElectronicSpeedControllerVoltageChecker",
					Components:    "Motor and Electronic Speed Controller",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Motor minimum operating voltage (%.1fV) exceeds electronic speed controller maximum voltage rating (%.1fV).", m.MinVoltage, esc.MaxVoltage),
					Resolution:    "Select an electronic speed controller supporting higher voltage or a motor with lower operating voltage requirements.",
				})
			} else if esc.MinVoltage > 0 && m.MaxVoltage > 0 && m.MaxVoltage < esc.MinVoltage {
				messages = append(messages, &pb.CompatibilityMessage{
					CheckerName:   "MotorElectronicSpeedControllerVoltageChecker",
					Components:    "Motor and Electronic Speed Controller",
					SeverityLevel: 0,
					SeverityName:  "DEFINITE_INCOMPATIBILITY",
					Message:       fmt.Sprintf("Motor maximum voltage rating (%.1fV) is below electronic speed controller minimum operating voltage (%.1fV).", m.MaxVoltage, esc.MinVoltage),
					Resolution:    "Select a motor rated for higher voltage or an electronic speed controller with lower minimum voltage requirements.",
				})
			}
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

	// Rule 17: Receiver vs Antenna (Frequency and Connector Compatibility)
	if len(comp.Receivers) > 0 && len(comp.Antennas) > 0 {
		for _, rx := range comp.Receivers {
			if rx == nil {
				continue
			}
			for _, ant := range comp.Antennas {
				if ant == nil {
					continue
				}
				// Identify if this antenna is an RX antenna (frequency <= 3000 MHz)
				if ant.FrequencyBandMhz > 0 && ant.FrequencyBandMhz <= 3000 {
					// Check frequency matching
					if rx.FrequencyBandMhz >= 2400 && rx.FrequencyBandMhz <= 2500 {
						if ant.FrequencyBandMhz < 2400 || ant.FrequencyBandMhz > 2500 {
							messages = append(messages, &pb.CompatibilityMessage{
								CheckerName:   "ReceiverAntennaFrequencyChecker",
								Components:    "Receiver and Antenna",
								SeverityLevel: 0,
								SeverityName:  "DEFINITE_INCOMPATIBILITY",
								Message:       fmt.Sprintf("Receiver operates at 2.4 GHz (%d MHz), but selected antenna operates at %d MHz.", rx.FrequencyBandMhz, ant.FrequencyBandMhz),
								Resolution:    "Select a 2.4 GHz receiver antenna.",
							})
						}
					} else if rx.FrequencyBandMhz > 0 && rx.FrequencyBandMhz <= 1000 {
						if ant.FrequencyBandMhz > 1000 {
							messages = append(messages, &pb.CompatibilityMessage{
								CheckerName:   "ReceiverAntennaFrequencyChecker",
								Components:    "Receiver and Antenna",
								SeverityLevel: 0,
								SeverityName:  "DEFINITE_INCOMPATIBILITY",
								Message:       fmt.Sprintf("Receiver operates at %d MHz sub-GHz, but selected antenna operates at %d MHz.", rx.FrequencyBandMhz, ant.FrequencyBandMhz),
								Resolution:    "Select a sub-GHz (868/900/915 MHz) receiver antenna.",
							})
						}
					}

					// Check connector matching (receivers require micro-coaxial connectors U.FL / IPEX)
					connLower := strings.ToLower(strings.TrimSpace(ant.Connector))
					if connLower != "" && !strings.Contains(connLower, "u.fl") && !strings.Contains(connLower, "ipex") && !strings.Contains(connLower, "integrated") {
						messages = append(messages, &pb.CompatibilityMessage{
							CheckerName:   "ReceiverAntennaConnectorChecker",
							Components:    "Receiver and Antenna",
							SeverityLevel: 0,
							SeverityName:  "DEFINITE_INCOMPATIBILITY",
							Message:       fmt.Sprintf("Receiver requires a U.FL / IPEX antenna connector, but selected antenna has a %s connector.", ant.Connector),
							Resolution:    "Select an antenna with a U.FL or IPEX connector.",
						})
					}
				}
			}
		}
	}

	// Rule 18: Video Transmitter vs Antenna (Frequency and Connector Compatibility)
	if comp.VideoTransmitter != nil && len(comp.Antennas) > 0 {
		vtx := comp.VideoTransmitter
		for _, ant := range comp.Antennas {
			if ant == nil {
				continue
			}
			// Identify if this antenna is intended for VTX (frequency >= 4000 MHz)
			if ant.FrequencyBandMhz >= 4000 {
				if ant.FrequencyBandMhz < 5000 || ant.FrequencyBandMhz > 6000 {
					messages = append(messages, &pb.CompatibilityMessage{
						CheckerName:   "VideoTransmitterAntennaFrequencyChecker",
						Components:    "Video Transmitter and Antenna",
						SeverityLevel: 0,
						SeverityName:  "DEFINITE_INCOMPATIBILITY",
						Message:       fmt.Sprintf("Video transmitter operates at 5.8 GHz, but selected antenna operates at %d MHz.", ant.FrequencyBandMhz),
						Resolution:    "Select a 5.8 GHz video antenna.",
					})
				}

				// Check connector matching for VTX
				vtxProtoLower := strings.ToLower(vtx.Protocol)
				vtxNameLower := strings.ToLower(vtx.Name)
				antConnLower := strings.ToLower(strings.TrimSpace(ant.Connector))

				// If DJI O3, it requires Dual U.FL or U.FL
				if strings.Contains(vtxProtoLower, "o3") || strings.Contains(vtxNameLower, "o3") {
					if !strings.Contains(antConnLower, "u.fl") && !strings.Contains(antConnLower, "ipex") {
						messages = append(messages, &pb.CompatibilityMessage{
							CheckerName:   "VideoTransmitterAntennaConnectorChecker",
							Components:    "Video Transmitter and Antenna",
							SeverityLevel: 0,
							SeverityName:  "DEFINITE_INCOMPATIBILITY",
							Message:       fmt.Sprintf("DJI O3 Air Unit requires a Dual U.FL or U.FL antenna, but selected antenna has a %s connector.", ant.Connector),
							Resolution:    "Select an antenna with a Dual U.FL or U.FL connector.",
						})
					}
				}
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
		if comp.Motor != nil && len(comp.Motor.PropMountInterfaces) > 0 {
			if len(comp.Motor.PropMountInterfaces) == 1 {
				conditions = append(conditions, fmt.Sprintf("prop_mount_interface == %q", comp.Motor.PropMountInterfaces[0]))
			} else {
				var ifaceChecks []string
				for _, iface := range comp.Motor.PropMountInterfaces {
					ifaceChecks = append(ifaceChecks, fmt.Sprintf("prop_mount_interface == %q", iface))
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(ifaceChecks, " || ")))
			}
		}
	case "frames":
		if comp.Propeller != nil && comp.Propeller.DiameterMm > 0 {
			conditions = append(conditions, fmt.Sprintf("max_prop_size_mm >= %.1f", comp.Propeller.DiameterMm))
		}
		if len(comp.Cameras) > 0 && comp.Cameras[0].WidthMm > 0 {
			conditions = append(conditions, fmt.Sprintf("camera_mount_width_mm >= %.1f", float32(comp.Cameras[0].WidthMm)))
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
		if comp.Motor != nil && comp.Motor.MaxCurrentA > 0 {
			conditions = append(conditions, fmt.Sprintf("motor_current_max_a >= %.1f", comp.Motor.MaxCurrentA))
		}
		if comp.Motor != nil && comp.Motor.MinVoltage > 0 {
			conditions = append(conditions, fmt.Sprintf("max_voltage >= %.2f", comp.Motor.MinVoltage))
		}
		if comp.Motor != nil && comp.Motor.MaxVoltage > 0 {
			conditions = append(conditions, fmt.Sprintf("min_voltage <= %.2f", comp.Motor.MaxVoltage))
		}
		if comp.FlightController != nil {
			fc := comp.FlightController
			if fc.MaxVoltage > 0 {
				conditions = append(conditions, fmt.Sprintf("min_voltage <= %.2f", fc.MaxVoltage))
			}
			if fc.MinVoltage > 0 {
				conditions = append(conditions, fmt.Sprintf("max_voltage >= %.2f", fc.MinVoltage))
			}
		}
	case "motors":
		if comp.Frame != nil && len(comp.Frame.MotorMountPatterns) > 0 {
			if len(comp.Frame.MotorMountPatterns) == 1 {
				conditions = append(conditions, fmt.Sprintf("mount_pattern == %q", comp.Frame.MotorMountPatterns[0]))
			} else {
				var parts []string
				for _, p := range comp.Frame.MotorMountPatterns {
					parts = append(parts, fmt.Sprintf("mount_pattern == %q", p))
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(parts, " || ")))
			}
		}
	case "cameras":
		if comp.VideoTransmitter != nil && comp.VideoTransmitter.Protocol != "" {
			conditions = append(conditions, fmt.Sprintf("protocol == %q", comp.VideoTransmitter.Protocol))
		}
		if comp.Frame != nil && comp.Frame.CameraMountWidthMm > 0 {
			conditions = append(conditions, fmt.Sprintf("width_mm <= %d", uint32(comp.Frame.CameraMountWidthMm)))
		}
	case "flight_controllers":
		if comp.Frame != nil && len(comp.Frame.BoardMountPatterns) > 0 {
			if len(comp.Frame.BoardMountPatterns) == 1 {
				conditions = append(conditions, fmt.Sprintf("mount_pattern == %q", comp.Frame.BoardMountPatterns[0]))
			} else {
				var parts []string
				for _, p := range comp.Frame.BoardMountPatterns {
					parts = append(parts, fmt.Sprintf("mount_pattern == %q", p))
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(parts, " || ")))
			}
		}
	case "video_transmitters":
		if comp.FlightController != nil {
			fc := comp.FlightController
			if fc.VtxVoltageMaxV > 0 {
				conditions = append(conditions, fmt.Sprintf("input_voltage_min_v <= %.1f", fc.VtxVoltageMaxV))
			}
			if fc.VtxVoltageMinV > 0 {
				conditions = append(conditions, fmt.Sprintf("input_voltage_max_v >= %.1f", fc.VtxVoltageMinV))
			}
			if len(fc.VtxPorts) == 1 {
				conditions = append(conditions, fmt.Sprintf("port_type == %q", fc.VtxPorts[0]))
			} else if len(fc.VtxPorts) > 1 {
				var parts []string
				for _, p := range fc.VtxPorts {
					parts = append(parts, fmt.Sprintf("port_type == %q", p))
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(parts, " || ")))
			}
		}
		if comp.Frame != nil && len(comp.Frame.BoardMountPatterns) > 0 {
			if len(comp.Frame.BoardMountPatterns) == 1 {
				conditions = append(conditions, fmt.Sprintf("mount_pattern == %q", comp.Frame.BoardMountPatterns[0]))
			} else {
				var parts []string
				for _, p := range comp.Frame.BoardMountPatterns {
					parts = append(parts, fmt.Sprintf("mount_pattern == %q", p))
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(parts, " || ")))
			}
		}
	case "antennas":
		if len(comp.Receivers) > 0 && comp.Receivers[0] != nil {
			rx := comp.Receivers[0]
			if rx.FrequencyBandMhz >= 2400 && rx.FrequencyBandMhz <= 2500 {
				conditions = append(conditions, "frequency_band_mhz == 2400")
			} else if rx.FrequencyBandMhz > 0 && rx.FrequencyBandMhz <= 1000 {
				conditions = append(conditions, "(frequency_band_mhz == 915 || frequency_band_mhz == 868 || frequency_band_mhz == 900)")
			}
			conditions = append(conditions, "(connector == \"U.FL\" || connector == \"IPEX\")")
		} else if comp.VideoTransmitter != nil {
			conditions = append(conditions, "frequency_band_mhz == 5800")
			vtx := comp.VideoTransmitter
			protoLower := strings.ToLower(vtx.Protocol)
			nameLower := strings.ToLower(vtx.Name)
			if strings.Contains(protoLower, "o3") || strings.Contains(nameLower, "o3") {
				conditions = append(conditions, "(connector == \"Dual U.FL\" || connector == \"U.FL\")")
			}
		}
	}

	return strings.Join(conditions, " && ")
}
