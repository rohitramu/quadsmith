package evaluator

import (
	"testing"

	pb "quadsmith/api/gen/quadsmith"
)

func TestEvaluatePhysics_5InchFreestyle(t *testing.T) {
	motor := &pb.Motor{
		StatorDiameterMm: 22,
		StatorHeightMm:   7.5,
		Kv:               1950,
		WeightG:          33.9,
	}
	prop := &pb.Propeller{
		DiameterMm: 131.83, // 5.19 in
		PitchMm:    91.44,  // 3.6 in
		Blades:     3,
		WeightG:    4.2,
	}
	battery := &pb.Battery{
		CellCountS:  6,
		CapacityMah: 1500,
		WeightG:     255,
	}
	totalWeight := float32(581.0) // 5" AUW
	maxEscAmps := float32(50.0)

	twr, hover, flightTime, errs, warns := CalculatePhysics(motor, prop, battery, totalWeight, maxEscAmps)

	if len(errs) > 0 {
		t.Fatalf("Unexpected physics errors: %v", errs)
	}
	if len(warns) > 0 {
		t.Fatalf("Unexpected physics warnings: %v", warns)
	}

	// 5-inch 6S freestyle drone should have realistic TWR between 8:1 and 12:1
	if twr < 8.0 || twr > 12.0 {
		t.Errorf("Expected 5-inch TWR between 8.0 and 12.0, got %.2f", twr)
	}

	// Hover throttle should follow aerodynamic quadratic curve ~28% - 36%
	if hover < 28.0 || hover > 36.0 {
		t.Errorf("Expected hover throttle between 28%% and 36%%, got %.1f%%", hover)
	}

	// Hover flight time should be plausible (> 10 min)
	if flightTime < 10.0 || flightTime > 25.0 {
		t.Errorf("Expected hover flight time between 10 and 25 min, got %.1f", flightTime)
	}
}

func TestEvaluatePhysics_7InchLongRange(t *testing.T) {
	motor := &pb.Motor{
		StatorDiameterMm: 28,
		StatorHeightMm:   6.5,
		Kv:               1300,
		WeightG:          46.0,
	}
	prop := &pb.Propeller{
		DiameterMm: 190.5, // 7.5 in
		PitchMm:    88.9,  // 3.5 in
		Blades:     3,
		WeightG:    8.5,
	}
	battery := &pb.Battery{
		CellCountS:  6,
		CapacityMah: 3000,
		WeightG:     390,
	}
	totalWeight := float32(850.0) // 7" long range AUW
	maxEscAmps := float32(50.0)

	twr, hover, flightTime, errs, warns := CalculatePhysics(motor, prop, battery, totalWeight, maxEscAmps)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}
	if len(warns) > 0 {
		t.Fatalf("Unexpected warnings: %v", warns)
	}

	// 7-inch cruiser should have TWR ~6.0 - 8.5
	if twr < 6.0 || twr > 8.5 {
		t.Errorf("Expected 7-inch TWR between 6.0 and 8.5, got %.2f", twr)
	}

	// Hover throttle should be ~34% - 41%
	if hover < 34.0 || hover > 41.0 {
		t.Errorf("Expected hover throttle between 34%% and 41%%, got %.1f%%", hover)
	}

	// Long range battery should provide 20+ min endurance
	if flightTime < 20.0 {
		t.Errorf("Expected long range flight time >= 20 min, got %.1f", flightTime)
	}
}

func TestEvaluatePhysics_OverloadedDrone(t *testing.T) {
	// Tiny motor on 1S with huge payload
	motor := &pb.Motor{
		StatorDiameterMm: 11,
		StatorHeightMm:   3.0,
		Kv:               11000,
	}
	prop := &pb.Propeller{
		DiameterMm: 50.8, // 2 in
		PitchMm:    38.1,
		Blades:     2,
	}
	battery := &pb.Battery{
		CellCountS:  1,
		CapacityMah: 450,
	}
	totalWeight := float32(300.0) // severely overloaded (300g on 2" 1S)

	twr, hover, _, errs, _ := CalculatePhysics(motor, prop, battery, totalWeight, 10.0)

	if twr >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for overloaded drone, got %.2f", twr)
	}
	if hover < 100.0 {
		t.Errorf("Expected hover throttle >= 100%% for overloaded drone, got %.1f%%", hover)
	}
	if len(errs) == 0 {
		t.Errorf("Expected 'too heavy to take off' error, got none")
	}
}

func TestEvaluatePhysics_SluggishWarning(t *testing.T) {
	motor := &pb.Motor{
		StatorDiameterMm: 22,
		StatorHeightMm:   7.0,
		Kv:               1950,
	}
	prop := &pb.Propeller{
		DiameterMm: 127.0,
		PitchMm:    80.0,
		Blades:     3,
	}
	battery := &pb.Battery{
		CellCountS:  6,
		CapacityMah: 1500,
	}
	// Weight chosen to put TWR around 3:1 (Hover throttle ~57% > 50%)
	totalWeight := float32(1800.0)

	twr, hover, _, _, warns := CalculatePhysics(motor, prop, battery, totalWeight, 45.0)

	if twr > 3.5 || twr < 2.5 {
		t.Errorf("Expected TWR around 3:1, got %.2f", twr)
	}
	if hover <= 50.0 {
		t.Errorf("Expected hover throttle > 50%%, got %.1f%%", hover)
	}
	foundSluggish := false
	for _, w := range warns {
		if w == "Drone will be very sluggish (Hover throttle > 50%)" {
			foundSluggish = true
			break
		}
	}
	if !foundSluggish {
		t.Errorf("Expected sluggish warning, got %v", warns)
	}
}

func TestEvaluatePhysics_MissingInputs(t *testing.T) {
	_, _, _, _, warns := CalculatePhysics(nil, nil, nil, 500, 40)
	if len(warns) == 0 {
		t.Errorf("Expected missing components warning, got none")
	}
}
