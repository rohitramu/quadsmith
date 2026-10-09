package evaluator

import (
	"strings"
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

	res := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, maxEscAmps)

	if len(res.Errors) > 0 {
		t.Fatalf("Unexpected physics errors: %v", res.Errors)
	}
	if len(res.Warnings) > 0 {
		t.Fatalf("Unexpected physics warnings: %v", res.Warnings)
	}

	// 5-inch 6S freestyle drone should have realistic TWR between 5.5:1 and 7.5:1
	if res.ThrustToWeightRatio < 5.5 || res.ThrustToWeightRatio > 7.5 {
		t.Errorf("Expected 5-inch TWR between 5.5 and 7.5, got %.2f", res.ThrustToWeightRatio)
	}

	// Bare hover throttle should be in typical Betaflight 20% - 26% range
	if res.HoverThrottlePercent < 20.0 || res.HoverThrottlePercent > 26.0 {
		t.Errorf("Expected bare hover throttle between 20%% and 26%%, got %.1f%%", res.HoverThrottlePercent)
	}

	// 5-inch 6S hover RPM should be between 10,000 and 14,000 RPM
	if res.HoverRpm < 10000 || res.HoverRpm > 14000 {
		t.Errorf("Expected 5-inch hover RPM between 10,000 and 14,000, got %d", res.HoverRpm)
	}

	// Max vertical acceleration: (TWR - 1.0) * 9.8 m/s^2 -> ~44 to 64 m/s^2 (~4.5G - 6.5G punchout)
	if res.MaxAccelerationMps2 < 44.0 || res.MaxAccelerationMps2 > 64.0 {
		t.Errorf("Expected max acceleration between 44.0 and 64.0 m/s^2, got %.1f m/s^2", res.MaxAccelerationMps2)
	}

	// Terminal top speed in forward flight: ~140 to 180 km/h
	if res.TopSpeedKmh < 140.0 || res.TopSpeedKmh > 180.0 {
		t.Errorf("Expected top speed between 140.0 and 180.0 km/h, got %.1f km/h", res.TopSpeedKmh)
	}

	// Flight time range should be realistic:
	// Aggressive freestyle: 3.0 to 4.8 min
	if res.MinFlightTimeMin < 3.0 || res.MinFlightTimeMin > 4.8 {
		t.Errorf("Expected min flight time between 3.0 and 4.8 min, got %.1f min", res.MinFlightTimeMin)
	}
	// Smooth cruising: 6.0 to 8.5 min
	if res.MaxFlightTimeMin < 6.0 || res.MaxFlightTimeMin > 8.5 {
		t.Errorf("Expected max flight time between 6.0 and 8.5 min, got %.1f min", res.MaxFlightTimeMin)
	}
	if res.MinFlightTimeMin >= res.EstimatedFlightTimeMin || res.EstimatedFlightTimeMin >= res.MaxFlightTimeMin {
		t.Errorf("Expected minFlightTime (%.1f) < flightTime (%.1f) < maxFlightTime (%.1f)",
			res.MinFlightTimeMin, res.EstimatedFlightTimeMin, res.MaxFlightTimeMin)
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
	res0 := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	// Evaluate with GoPro (+133g payload)
	resGoPro := CalculatePhysics(motor, prop, battery, baseWeight, 133, maxEscAmps)

	// Thrust-to-weight ratio should scale down noticeably (at least 1.0 point drop)
	twrDiff := res0.ThrustToWeightRatio - resGoPro.ThrustToWeightRatio
	if twrDiff < 1.0 {
		t.Errorf("Expected TWR to scale down by at least 1.0 with +133g payload, but only decreased by %.2f (from %.2f to %.2f)",
			twrDiff, res0.ThrustToWeightRatio, resGoPro.ThrustToWeightRatio)
	}

	// Max vertical acceleration should scale down significantly (at least 8 m/s^2 drop)
	accelDiff := res0.MaxAccelerationMps2 - resGoPro.MaxAccelerationMps2
	if accelDiff < 8.0 {
		t.Errorf("Expected max acceleration to decrease by at least 8 m/s^2 with +133g payload, got %.1f to %.1f", res0.MaxAccelerationMps2, resGoPro.MaxAccelerationMps2)
	}

	// Forward top speed should decrease slightly due to extra camera drag
	if resGoPro.TopSpeedKmh >= res0.TopSpeedKmh {
		t.Errorf("Expected top speed with payload (%.1f km/h) to be lower than bare (%.1f km/h)", resGoPro.TopSpeedKmh, res0.TopSpeedKmh)
	}

	// Hover throttle should scale up visibly (at least +6 percentage points)
	hoverDiff := resGoPro.HoverThrottlePercent - res0.HoverThrottlePercent
	if hoverDiff < 6.0 {
		t.Errorf("Expected hover throttle to increase by at least 6%% with +133g payload, but only increased by %.1f%% (from %.1f%% to %.1f%%)",
			hoverDiff, res0.HoverThrottlePercent, resGoPro.HoverThrottlePercent)
	}

	// Hover RPM must increase with payload to generate additional lift
	if resGoPro.HoverRpm <= res0.HoverRpm {
		t.Errorf("Expected hover RPM with payload (%d) to be higher than bare (%d)", resGoPro.HoverRpm, res0.HoverRpm)
	}

	// Flight time range should decrease noticeably
	if res0.MinFlightTimeMin-resGoPro.MinFlightTimeMin < 0.6 {
		t.Errorf("Expected min flight time to decrease by at least 0.6 min with +133g payload, got %.1f to %.1f", res0.MinFlightTimeMin, resGoPro.MinFlightTimeMin)
	}
	if res0.MaxFlightTimeMin-resGoPro.MaxFlightTimeMin < 1.2 {
		t.Errorf("Expected max flight time to decrease by at least 1.2 min with +133g payload, got %.1f to %.1f", res0.MaxFlightTimeMin, resGoPro.MaxFlightTimeMin)
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
	res0 := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)
	if res0.ThrustToWeightRatio <= 3.5 {
		t.Errorf("Expected bare toothpick TWR > 3.5, got %.2f", res0.ThrustToWeightRatio)
	}
	if res0.HoverThrottlePercent >= 40.0 {
		t.Errorf("Expected bare toothpick hover < 40%%, got %.1f%%", res0.HoverThrottlePercent)
	}
	if res0.HoverRpm < 13000 || res0.HoverRpm > 22000 {
		t.Errorf("Expected bare toothpick hover RPM between 13,000 and 22,000, got %d", res0.HoverRpm)
	}
	if len(res0.Errors) > 0 {
		t.Errorf("Unexpected errors for bare toothpick: %v", res0.Errors)
	}

	// 2. Toothpick with +200g heavy payload (TWR ~1.8 > 1.0)
	// Must NOT exceed 100% hover throttle because TWR is still > 1.0
	res200 := CalculatePhysics(motor, prop, battery, baseWeight, 200, maxEscAmps)
	if res200.ThrustToWeightRatio <= 1.0 {
		t.Errorf("Expected TWR > 1.0 with +200g payload, got %.2f", res200.ThrustToWeightRatio)
	}
	if res200.HoverThrottlePercent >= 100.0 {
		t.Errorf("Hover throttle must be < 100%% when TWR > 1.0, got %.1f%% (TWR=%.2f)", res200.HoverThrottlePercent, res200.ThrustToWeightRatio)
	}
	if res200.HoverRpm <= res0.HoverRpm {
		t.Errorf("Expected hover RPM with 200g payload (%d) > bare (%d)", res200.HoverRpm, res0.HoverRpm)
	}
	if len(res200.Errors) > 0 {
		t.Errorf("Expected no 'too heavy to take off' error when TWR > 1.0, got: %v", res200.Errors)
	}
	if len(res200.Warnings) == 0 {
		t.Errorf("Expected sluggish warning for +200g payload on 139g toothpick, got none")
	}

	// 3. Severely overloaded toothpick (+550g payload, total ~689g)
	// TWR < 1.0, must exceed 100% hover throttle and report takeoff error
	resOver := CalculatePhysics(motor, prop, battery, baseWeight, 550, maxEscAmps)
	if resOver.ThrustToWeightRatio >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for +550g payload on toothpick, got %.2f", resOver.ThrustToWeightRatio)
	}
	if resOver.HoverRpm != 0 {
		t.Errorf("Expected hover RPM = 0 for overloaded drone, got %d", resOver.HoverRpm)
	}
	if resOver.MaxAccelerationMps2 != 0.0 {
		t.Errorf("Expected max acceleration = 0 for overloaded drone, got %.1f", resOver.MaxAccelerationMps2)
	}
	if resOver.HoverThrottlePercent < 100.0 {
		t.Errorf("Hover throttle must be >= 100%% when TWR < 1.0, got %.1f%%", resOver.HoverThrottlePercent)
	}
	if len(resOver.Errors) == 0 {
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

	res := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	if len(res.Errors) > 0 {
		t.Fatalf("Unexpected errors: %v", res.Errors)
	}
	if len(res.Warnings) > 0 {
		t.Fatalf("Unexpected warnings: %v", res.Warnings)
	}

	// 7-inch cruiser should have TWR ~3.8 - 5.5
	if res.ThrustToWeightRatio < 3.8 || res.ThrustToWeightRatio > 5.5 {
		t.Errorf("Expected 7-inch TWR between 3.8 and 5.5, got %.2f", res.ThrustToWeightRatio)
	}

	// 7-inch cruiser hover RPM should be between 5,000 and 8,500 RPM
	if res.HoverRpm < 5000 || res.HoverRpm > 8500 {
		t.Errorf("Expected 7-inch hover RPM between 5,000 and 8,500, got %d", res.HoverRpm)
	}

	// Max vertical acceleration: ~25.0 to 45.0 m/s^2 (~2.5G - 4.5G)
	if res.MaxAccelerationMps2 < 25.0 || res.MaxAccelerationMps2 > 45.0 {
		t.Errorf("Expected 7-inch max acceleration between 25.0 and 45.0 m/s^2, got %.1f m/s^2", res.MaxAccelerationMps2)
	}

	// Terminal top speed: ~95 to 135 km/h
	if res.TopSpeedKmh < 95.0 || res.TopSpeedKmh > 135.0 {
		t.Errorf("Expected 7-inch top speed between 95.0 and 135.0 km/h, got %.1f km/h", res.TopSpeedKmh)
	}

	// Hover throttle should be ~25% - 35%
	if res.HoverThrottlePercent < 25.0 || res.HoverThrottlePercent > 35.0 {
		t.Errorf("Expected hover throttle between 25%% and 35%%, got %.1f%%", res.HoverThrottlePercent)
	}

	// Long range battery should provide endurance flight time >= 9.5 min
	if res.MaxFlightTimeMin < 9.5 {
		t.Errorf("Expected long range max flight time >= 9.5 min, got %.1f", res.MaxFlightTimeMin)
	}
	if res.MinFlightTimeMin >= res.EstimatedFlightTimeMin || res.EstimatedFlightTimeMin >= res.MaxFlightTimeMin {
		t.Errorf("Expected min < mid < max for 7-inch, got %.1f < %.1f < %.1f", res.MinFlightTimeMin, res.EstimatedFlightTimeMin, res.MaxFlightTimeMin)
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

	res := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 10.0)

	if res.ThrustToWeightRatio >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for overloaded drone, got %.2f", res.ThrustToWeightRatio)
	}
	if res.HoverRpm != 0 {
		t.Errorf("Expected hover RPM = 0 for overloaded drone, got %d", res.HoverRpm)
	}
	if res.MaxAccelerationMps2 != 0.0 {
		t.Errorf("Expected max acceleration 0 for overloaded drone, got %.1f", res.MaxAccelerationMps2)
	}
	if res.HoverThrottlePercent < 100.0 {
		t.Errorf("Expected hover throttle >= 100%% for overloaded drone, got %.1f%%", res.HoverThrottlePercent)
	}
	if len(res.Errors) == 0 {
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

	res := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, 45.0)

	if res.HoverThrottlePercent <= 50.0 {
		t.Errorf("Expected hover throttle > 50%%, got %.1f%%", res.HoverThrottlePercent)
	}
	foundSluggish := false
	for _, w := range res.Warnings {
		if w == "Drone will be very sluggish (Hover throttle > 50%)" {
			foundSluggish = true
			break
		}
	}
	if !foundSluggish {
		t.Errorf("Expected sluggish warning, got %v", res.Warnings)
	}
}

func TestEvaluatePhysics_MissingInputs(t *testing.T) {
	res := CalculatePhysics(nil, nil, nil, 500, 0, 40)
	if len(res.Warnings) == 0 {
		t.Errorf("Expected missing components warning, got none")
	}
}

func TestEvaluatePhysics_ToothpickExtremeOverload(t *testing.T) {
	// Flywoo ROBO 1202.5 5500KV motor
	motor := &pb.Motor{
		StatorDiameterMm: 12,
		StatorHeightMm:   2.5,
		Kv:               5500,
		WeightG:          4.2,
	}
	// 3-inch propeller
	prop := &pb.Propeller{
		DiameterMm: 76.2,
		PitchMm:    76.2,
		Blades:     3,
		WeightG:    1.5,
	}
	// 4S 650mAh LiPo battery
	battery := &pb.Battery{
		CellCountS:  4,
		CapacityMah: 650,
		WeightG:     65.0,
	}
	baseWeight := float32(139.0)     // Toothpick AUW bare
	payloadWeight := float32(1300.0) // Massive +1300g payload (Total: 1439g)
	maxEscAmps := float32(12.0)

	res := CalculatePhysics(motor, prop, battery, baseWeight, payloadWeight, maxEscAmps)

	// Motors must saturate on shaft power limit (~90W) rather than reporting unphysical 53,000+ RPM hover.
	// TWR should be ~0.40 (< 1.0)
	if res.ThrustToWeightRatio >= 1.0 {
		t.Errorf("Expected TWR < 1.0 for 1439g payload on 1202.5 toothpick, got %.2f", res.ThrustToWeightRatio)
	}

	// Cannot hover: hover RPM must be 0
	if res.HoverRpm != 0 {
		t.Errorf("Expected hover RPM = 0 for overloaded toothpick, got %d (should not report unphysical RPM)", res.HoverRpm)
	}

	// Hover throttle must exceed 100%
	if res.HoverThrottlePercent <= 100.0 {
		t.Errorf("Expected hover throttle > 100%%, got %.1f%%", res.HoverThrottlePercent)
	}

	// Must report takeoff error
	foundTakeoffError := false
	for _, e := range res.Errors {
		if e == "Drone is too heavy to take off (Hover throttle > 100%)" {
			foundTakeoffError = true
			break
		}
	}
	if !foundTakeoffError {
		t.Errorf("Expected takeoff error, got %v", res.Errors)
	}

	// Max acceleration and top speed must be 0
	if res.MaxAccelerationMps2 != 0.0 {
		t.Errorf("Expected max acceleration 0.0 m/s^2, got %.1f", res.MaxAccelerationMps2)
	}
	if res.TopSpeedKmh != 0.0 {
		t.Errorf("Expected top speed 0.0 km/h, got %.1f", res.TopSpeedKmh)
	}
}

func TestEvaluatePhysics_ElectricalWarnings(t *testing.T) {
	// Undersized 5A ESC on a 5-inch 6S freestyle drone drawing ~32A burst
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
	baseWeight := float32(581.0)
	maxEscAmps := float32(5.0) // Extremely undersized ESC

	res := CalculatePhysics(motor, prop, battery, baseWeight, 0, maxEscAmps)

	foundEscWarning := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "exceeds ESC burst rating") {
			foundEscWarning = true
			break
		}
	}
	if !foundEscWarning {
		t.Errorf("Expected ESC burst overload warning for 5A ESC on 5-inch build, got warnings: %v", res.Warnings)
	}
}
