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
	baseWeight := float32(581.0) // 5" AUW bare
	payloadWeight := float32(0.0)
	maxEscAmps := float32(50.0)

	twr, hover, flightTime, minFlightTime, maxFlightTime, errs, warns := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, maxEscAmps)

	if len(errs) > 0 {
		t.Fatalf("Unexpected physics errors: %v", errs)
	}
	if len(warns) > 0 {
		t.Fatalf("Unexpected physics warnings: %v", warns)
	}

	// 5-inch 6S freestyle drone should have realistic TWR between 5.5:1 and 7.5:1
	if twr < 5.5 || twr > 7.5 {
		t.Errorf("Expected 5-inch TWR between 5.5 and 7.5, got %.2f", twr)
	}

	// Bare hover throttle should be in typical Betaflight 20% - 26% range
	if hover < 20.0 || hover > 26.0 {
		t.Errorf("Expected bare hover throttle between 20%% and 26%%, got %.1f%%", hover)
	}

	// Flight time range should be realistic:
	// Aggressive freestyle: 3.0 to 4.8 min
	if minFlightTime < 3.0 || minFlightTime > 4.8 {
		t.Errorf("Expected min flight time between 3.0 and 4.8 min, got %.1f min", minFlightTime)
	}
	// Smooth cruising: 6.0 to 8.5 min
	if maxFlightTime < 6.0 || maxFlightTime > 8.5 {
		t.Errorf("Expected max flight time between 6.0 and 8.5 min, got %.1f min", maxFlightTime)
	}
	if minFlightTime >= flightTime || flightTime >= maxFlightTime {
		t.Errorf("Expected minFlightTime (%.1f) < flightTime (%.1f) < maxFlightTime (%.1f)",
			minFlightTime, flightTime, maxFlightTime)
	}
}

func TestEvaluatePhysics_PayloadScaling(t *testing.T) {
	motor := &pb.Motor{
		StatorDiameterMm: 22,
		StatorHeightMm:   7.5,
		Kv:               1950,
		WeightG:          33.9,
	}
	prop := &pb.Propeller{
		DiameterMm: 131.83,
		PitchMm:    91.44,
		Blades:     3,
		WeightG:    4.2,
	}
	battery := &pb.Battery{
		CellCountS:  6,
		CapacityMah: 1500,
		WeightG:     255,
	}
	baseWeight := float32(580.0)
	maxEscAmps := float32(50.0)

	// Evaluate at 0g payload
	twr0, hover0, _, min0, max0, _, _ := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	// Evaluate with GoPro (+133g payload)
	twrGoPro, hoverGoPro, _, minGoPro, maxGoPro, _, _ := CalculatePhysics(motor, prop, battery, baseWeight, 133, maxEscAmps)

	// Thrust-to-weight ratio should scale down noticeably (at least 1.0 point drop)
	twrDiff := twr0 - twrGoPro
	if twrDiff < 1.0 {
		t.Errorf("Expected TWR to scale down by at least 1.0 with +133g payload, but only decreased by %.2f (from %.2f to %.2f)",
			twrDiff, twr0, twrGoPro)
	}

	// Hover throttle should scale up visibly (at least +7 percentage points)
	hoverDiff := hoverGoPro - hover0
	if hoverDiff < 7.0 {
		t.Errorf("Expected hover throttle to increase by at least 7%% with +133g payload, but only increased by %.1f%% (from %.1f%% to %.1f%%)",
			hoverDiff, hover0, hoverGoPro)
	}

	// Flight time range should decrease noticeably
	if min0-minGoPro < 0.6 {
		t.Errorf("Expected min flight time to decrease by at least 0.6 min with +133g payload, got %.1f to %.1f", min0, minGoPro)
	}
	if max0-maxGoPro < 1.2 {
		t.Errorf("Expected max flight time to decrease by at least 1.2 min with +133g payload, got %.1f to %.1f", max0, maxGoPro)
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
	baseWeight := float32(850.0) // 7" long range AUW
	maxEscAmps := float32(50.0)

	twr, hover, flightTime, minFlightTime, maxFlightTime, errs, warns := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}
	if len(warns) > 0 {
		t.Fatalf("Unexpected warnings: %v", warns)
	}

	// 7-inch cruiser should have TWR ~3.8 - 5.5
	if twr < 3.8 || twr > 5.5 {
		t.Errorf("Expected 7-inch TWR between 3.8 and 5.5, got %.2f", twr)
	}

	// Hover throttle should be ~25% - 35%
	if hover < 25.0 || hover > 35.0 {
		t.Errorf("Expected hover throttle between 25%% and 35%%, got %.1f%%", hover)
	}

	// Long range battery should provide endurance flight time >= 9.5 min
	if maxFlightTime < 9.5 {
		t.Errorf("Expected long range max flight time >= 9.5 min, got %.1f", maxFlightTime)
	}
	if minFlightTime >= flightTime || flightTime >= maxFlightTime {
		t.Errorf("Expected min < mid < max for 7-inch, got %.1f < %.1f < %.1f", minFlightTime, flightTime, maxFlightTime)
	}
}

func TestEvaluatePhysics_OverloadedDrone(t *testing.T) {
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
	baseWeight := float32(100.0)
	payloadWeight := float32(200.0) // 300g on 2" 1S

	twr, hover, _, _, _, errs, _ := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 10.0)

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
	// Base weight 580g + 450g heavy payload -> triggers sluggish warning (> 50%)
	baseWeight := float32(580.0)
	payloadWeight := float32(450.0)

	_, hover, _, _, _, _, warns := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 45.0)

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
	_, _, _, _, _, _, warns := CalculatePhysics(nil, nil, nil, 500, 0, 40)
	if len(warns) == 0 {
		t.Errorf("Expected missing components warning, got none")
	}
}
