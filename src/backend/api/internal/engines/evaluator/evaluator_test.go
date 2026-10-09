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

	twr, hover, hoverRpm, flightTime, minFlightTime, maxFlightTime, maxAccel, topSpeed, errs, warns := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, maxEscAmps)

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

	// 5-inch 6S hover RPM should be between 10,000 and 14,000 RPM
	if hoverRpm < 10000 || hoverRpm > 14000 {
		t.Errorf("Expected 5-inch hover RPM between 10,000 and 14,000, got %d", hoverRpm)
	}

	// Max vertical acceleration: (TWR - 1.0) * 9.8 m/s^2 -> ~44 to 64 m/s^2 (~4.5G - 6.5G punchout)
	if maxAccel < 44.0 || maxAccel > 64.0 {
		t.Errorf("Expected max acceleration between 44.0 and 64.0 m/s^2, got %.1f m/s^2", maxAccel)
	}

	// Terminal top speed in forward flight: ~140 to 180 km/h
	if topSpeed < 140.0 || topSpeed > 180.0 {
		t.Errorf("Expected top speed between 140.0 and 180.0 km/h, got %.1f km/h", topSpeed)
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
	twr0, hover0, rpm0, _, min0, max0, accel0, speed0, _, _ := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	// Evaluate with GoPro (+133g payload)
	twrGoPro, hoverGoPro, rpmGoPro, _, minGoPro, maxGoPro, accelGoPro, speedGoPro, _, _ := CalculatePhysics(motor, prop, battery, baseWeight, 133, maxEscAmps)

	// Thrust-to-weight ratio should scale down noticeably (at least 1.0 point drop)
	twrDiff := twr0 - twrGoPro
	if twrDiff < 1.0 {
		t.Errorf("Expected TWR to scale down by at least 1.0 with +133g payload, but only decreased by %.2f (from %.2f to %.2f)",
			twrDiff, twr0, twrGoPro)
	}

	// Max vertical acceleration should scale down significantly (at least 8 m/s^2 drop)
	accelDiff := accel0 - accelGoPro
	if accelDiff < 8.0 {
		t.Errorf("Expected max acceleration to decrease by at least 8 m/s^2 with +133g payload, got %.1f to %.1f", accel0, accelGoPro)
	}

	// Forward top speed should decrease slightly due to extra camera drag
	if speedGoPro >= speed0 {
		t.Errorf("Expected top speed with payload (%.1f km/h) to be lower than bare (%.1f km/h)", speedGoPro, speed0)
	}

	// Hover throttle should scale up visibly (at least +6 percentage points)
	hoverDiff := hoverGoPro - hover0
	if hoverDiff < 6.0 {
		t.Errorf("Expected hover throttle to increase by at least 6%% with +133g payload, but only increased by %.1f%% (from %.1f%% to %.1f%%)",
			hoverDiff, hover0, hoverGoPro)
	}

	// Hover RPM must increase with payload to generate additional lift
	if rpmGoPro <= rpm0 {
		t.Errorf("Expected hover RPM with payload (%d) to be higher than bare (%d)", rpmGoPro, rpm0)
	}

	// Flight time range should decrease noticeably
	if min0-minGoPro < 0.6 {
		t.Errorf("Expected min flight time to decrease by at least 0.6 min with +133g payload, got %.1f to %.1f", min0, minGoPro)
	}
	if max0-maxGoPro < 1.2 {
		t.Errorf("Expected max flight time to decrease by at least 1.2 min with +133g payload, got %.1f to %.1f", max0, maxGoPro)
	}
}

func TestEvaluatePhysics_ToothpickPayloadConsistency(t *testing.T) {
	// Flywoo ROBO 1202.5 4500KV motor
	motor := &pb.Motor{
		StatorDiameterMm: 12,
		StatorHeightMm:   2.5,
		Kv:               4500,
		WeightG:          4.5,
	}
	// HQProp 3x3x3 3-inch propeller
	prop := &pb.Propeller{
		DiameterMm: 76.2,
		PitchMm:    76.2,
		Blades:     3,
		WeightG:    1.5,
	}
	// 3S 450mAh LiPo battery
	battery := &pb.Battery{
		CellCountS:  3,
		CapacityMah: 450,
		WeightG:     43.0,
	}
	baseWeight := float32(139.0) // Ultralight Toothpick bare AUW
	maxEscAmps := float32(12.0)

	// 1. Bare toothpick should hover comfortably
	twr0, hover0, rpm0, _, _, _, _, _, errs0, _ := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)
	if twr0 <= 3.5 {
		t.Errorf("Expected bare toothpick TWR > 3.5, got %.2f", twr0)
	}
	if hover0 >= 40.0 {
		t.Errorf("Expected bare toothpick hover < 40%%, got %.1f%%", hover0)
	}
	if rpm0 < 13000 || rpm0 > 22000 {
		t.Errorf("Expected bare toothpick hover RPM between 13,000 and 22,000, got %d", rpm0)
	}
	if len(errs0) > 0 {
		t.Errorf("Unexpected errors for bare toothpick: %v", errs0)
	}

	// 2. Toothpick with +200g heavy payload (TWR ~1.8 > 1.0)
	// Must NOT exceed 100% hover throttle because TWR is still > 1.0
	twr200, hover200, rpm200, _, _, _, _, _, errs200, warns200 := CalculatePhysics(motor, prop, battery, baseWeight, 200, maxEscAmps)
	if twr200 <= 1.0 {
		t.Errorf("Expected TWR > 1.0 with +200g payload, got %.2f", twr200)
	}
	if hover200 >= 100.0 {
		t.Errorf("Hover throttle must be < 100%% when TWR > 1.0, got %.1f%% (TWR=%.2f)", hover200, twr200)
	}
	if rpm200 <= rpm0 {
		t.Errorf("Expected hover RPM with 200g payload (%d) > bare (%d)", rpm200, rpm0)
	}
	if len(errs200) > 0 {
		t.Errorf("Expected no 'too heavy to take off' error when TWR > 1.0, got: %v", errs200)
	}
	if len(warns200) == 0 {
		t.Errorf("Expected sluggish warning for +200g payload on 139g toothpick, got none")
	}

	// 3. Severely overloaded toothpick (+550g payload, total ~689g)
	// TWR < 1.0, must exceed 100% hover throttle and report takeoff error
	twrOver, hoverOver, rpmOver, _, _, _, accelOver, _, errsOver, _ := CalculatePhysics(motor, prop, battery, baseWeight, 550, maxEscAmps)
	if twrOver >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for +550g payload on toothpick, got %.2f", twrOver)
	}
	if rpmOver != 0 {
		t.Errorf("Expected hover RPM = 0 for overloaded drone, got %d", rpmOver)
	}
	if accelOver != 0.0 {
		t.Errorf("Expected max acceleration = 0 for overloaded drone, got %.1f", accelOver)
	}
	if hoverOver < 100.0 {
		t.Errorf("Hover throttle must be >= 100%% when TWR < 1.0, got %.1f%%", hoverOver)
	}
	if len(errsOver) == 0 {
		t.Errorf("Expected takeoff error for overloaded toothpick, got none")
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

	twr, hover, hoverRpm, flightTime, minFlightTime, maxFlightTime, maxAccel, topSpeed, errs, warns := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

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

	// 7-inch cruiser hover RPM should be between 5,000 and 8,500 RPM
	if hoverRpm < 5000 || hoverRpm > 8500 {
		t.Errorf("Expected 7-inch hover RPM between 5,000 and 8,500, got %d", hoverRpm)
	}

	// Max vertical acceleration: ~25.0 to 45.0 m/s^2 (~2.5G - 4.5G)
	if maxAccel < 25.0 || maxAccel > 45.0 {
		t.Errorf("Expected 7-inch max acceleration between 25.0 and 45.0 m/s^2, got %.1f m/s^2", maxAccel)
	}

	// Terminal top speed: ~95 to 135 km/h
	if topSpeed < 95.0 || topSpeed > 135.0 {
		t.Errorf("Expected 7-inch top speed between 95.0 and 135.0 km/h, got %.1f km/h", topSpeed)
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

	twr, hover, rpm, _, _, _, accel, _, errs, _ := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 10.0)

	if twr >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for overloaded drone, got %.2f", twr)
	}
	if rpm != 0 {
		t.Errorf("Expected hover RPM = 0 for overloaded drone, got %d", rpm)
	}
	if accel != 0.0 {
		t.Errorf("Expected max acceleration 0 for overloaded drone, got %.1f", accel)
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

	_, hover, _, _, _, _, _, _, _, warns := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 45.0)

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
	_, _, _, _, _, _, _, _, _, warns := CalculatePhysics(nil, nil, nil, 500, 0, 40)
	if len(warns) == 0 {
		t.Errorf("Expected missing components warning, got none")
	}
}
