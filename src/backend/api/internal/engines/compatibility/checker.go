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
				Resolution:    "Choose a camera and video transmitter that share the same transmission protocol (e.g. Analog, DJI, Walksnail, or HDZero).",
			})
		}
	}

	// Rule 4: Battery Voltage Compatibility
	if comp.Battery != nil {
		bat := comp.Battery
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
				if esc.MaxVoltage > 0 && bat.MinVoltage > esc.MaxVoltage {
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
	}

	return messages
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
		var minV, maxV, maxA float32
		if comp.FlightController != nil && comp.FlightController.MinVoltage > minV {
			minV = comp.FlightController.MinVoltage
		}
		for _, esc := range comp.ElectronicSpeedControllers {
			if esc.MinVoltage > minV {
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
			updateMaxV(esc.MaxVoltage)
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
				mult := float32(esc.MaxMotors)
				if mult == 0 {
					mult = 1.0
				}
				maxA += esc.MotorCurrentMaxA * mult
			}
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
	case "electronic_speed_controllers":
		if comp.Motor != nil && comp.Motor.StatorDiameterMm > 20 {
			conditions = append(conditions, "motor_current_max_a >= 20.0")
		}
	case "motors":
		if len(comp.ElectronicSpeedControllers) > 0 {
			var minEscCurrent float32 = 0
			for _, esc := range comp.ElectronicSpeedControllers {
				if esc.MotorCurrentMaxA > 0 && (minEscCurrent == 0 || esc.MotorCurrentMaxA < minEscCurrent) {
					minEscCurrent = esc.MotorCurrentMaxA
				}
			}
			if minEscCurrent > 0 {
				conditions = append(conditions, fmt.Sprintf("max_current_a <= %.1f", minEscCurrent))
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
