package compatibility

import (
	pb "quadsmith/api/gen/quadsmith"
)

// Components represents a resolved set of hardware components to check for compatibility.
type Components struct {
	Frame                      *pb.Frame
	Motor                      *pb.Motor
	Battery                    *pb.Battery
	FlightController           *pb.FlightController
	Propeller                  *pb.Propeller
	ElectronicSpeedControllers []*pb.ElectronicSpeedController
	Receivers                  []*pb.Receiver
	Antennas                   []*pb.Antenna
	Cameras                    []*pb.Camera
	VideoTransmitter           *pb.VideoTransmitter
}

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
