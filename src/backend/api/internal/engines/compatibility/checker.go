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
	}

	return strings.Join(conditions, " && ")
}
