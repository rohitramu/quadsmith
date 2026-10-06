package compatibility

import (
	"testing"
	pb "quadsmith/api/gen/quadsmith"
)

func TestMotorEscCompatibility(t *testing.T) {
	comp := &Components{
		Motor: &pb.Motor{
			StatorDiameterMm: 22,
		},
		Escs: []*pb.Esc{
			{ContinuousAmps: 15},
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
			MaxPropSizeInches: 5.0,
		},
		Propeller: &pb.Propeller{
			DiameterInches: 5.1,
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
			MaxPropSizeInches: 5.1,
		},
		Propeller: &pb.Propeller{
			DiameterInches: 5.1,
		},
		Motor: &pb.Motor{
			StatorDiameterMm: 22,
		},
		Escs: []*pb.Esc{
			{ContinuousAmps: 45},
		},
	}

	messages := CheckCompatibility(comp)
	
	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages, got %d", len(messages))
	}
}
