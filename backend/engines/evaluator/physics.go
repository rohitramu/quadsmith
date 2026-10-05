package evaluator

import (
	pb "quadsmith/api/gen/quadsmith"
	"math"
)

const (
	AirDensityKgPerM3 = 1.225
	GravityMPerS2     = 9.81
	KinematicViscosity = 1.48e-5
)

func CalculateTotalMass(buildComponents []*pb.Component, battery *pb.Component, payloadWeightG float64) float64 {
	total := payloadWeightG
	if battery != nil && battery.GetWeightG() > 0 { total += battery.GetWeightG() }
	for _, comp := range buildComponents {
		if comp.GetWeightG() > 0 {
			multiplier := 1.0
			if isMotor(comp) || isProp(comp) { multiplier = 4.0 }
			total += (comp.GetWeightG()) * multiplier
		}
	}
	return total
}

type PropAeroData struct {
	DiameterMeters float64
	PitchMeters    float64
	ChordMeters    float64
	PitchRatio     float64
}

func getPropAeroData(prop *pb.Propeller) PropAeroData {
	d := prop.GetDiameterMm() / 1000.0
	p := prop.GetPitchMm() / 1000.0
	return PropAeroData{DiameterMeters: d, PitchMeters: p, ChordMeters: d / 10.0, PitchRatio: p / d}
}

func getLoadedRps(motor *pb.Motor, prop *pb.Propeller, battery *pb.Battery) float64 {
	aero := getPropAeroData(prop)
	maxVoltage := float64(battery.GetCellCountS()) * 4.2
	noLoadRpm := maxVoltage * float64(motor.GetKvRating())
	loadFactor := (math.Pow(aero.DiameterMeters, 4) * aero.PitchMeters) * 1e6
	rpmEfficiency := math.Max(0.5, 1.0 - (loadFactor * 0.005))
	return (noLoadRpm * rpmEfficiency) / 60.0
}

func EstimateMaxThrustGrams(motor *pb.Motor, prop *pb.Propeller, battery *pb.Battery) float64 {
	if motor == nil || prop == nil || battery == nil { return 0.0 }
	aero := getPropAeroData(prop)
	rps := getLoadedRps(motor, prop, battery)

	tipSpeedMps := rps * math.Pi * aero.DiameterMeters
	reynolds := (tipSpeedMps * aero.ChordMeters) / KinematicViscosity
	reynoldsEfficiency := 1.0
	if reynolds < 50000 { reynoldsEfficiency = math.Pow(reynolds/50000.0, 0.25) }

	cT := 0.12 * aero.PitchRatio * reynoldsEfficiency
	thrustNewtons := cT * AirDensityKgPerM3 * math.Pow(rps, 2) * math.Pow(aero.DiameterMeters, 4)
	return (thrustNewtons / GravityMPerS2) * 1000.0
}

func EstimateFlightTimeSeconds(totalMassG float64, motor *pb.Motor, prop *pb.Propeller, battery *pb.Battery) (hoverS float64, mixedS float64) {
	if motor == nil || prop == nil || battery == nil || totalMassG <= 0 { return 0, 0 }

	hoverThrustN := (totalMassG / 4000.0) * GravityMPerS2
	aero := getPropAeroData(prop)
	cT := math.Max(0.1, 0.12 * aero.PitchRatio)

	rpsHover := math.Sqrt(hoverThrustN / (cT * AirDensityKgPerM3 * math.Pow(aero.DiameterMeters, 4)))
	cP := cT * 0.5 
	mechPowerWatts := cP * AirDensityKgPerM3 * math.Pow(rpsHover, 3) * math.Pow(aero.DiameterMeters, 5)

	electricalPowerWatts := mechPowerWatts / 0.85
	hoverCurrentPerMotorA := electricalPowerWatts / (float64(battery.GetCellCountS()) * 3.7)
	totalHoverCurrentA := (hoverCurrentPerMotorA * 4.0) + 1.0 

	usableCapacityAh := (float64(battery.GetCapacityMah()) / 1000.0) * 0.80
	hoverS = (usableCapacityAh / totalHoverCurrentA) * 3600.0
	mixedS = (usableCapacityAh / (totalHoverCurrentA * 3.0)) * 3600.0
	return hoverS, mixedS
}

func EstimateTopSpeedKph(totalMassG float64, motor *pb.Motor, prop *pb.Propeller, battery *pb.Battery, frame *pb.Frame) float64 {
	if motor == nil || prop == nil || battery == nil || frame == nil || totalMassG <= 0 { return 0.0 }
	
	singleThrustG := EstimateMaxThrustGrams(motor, prop, battery)
	maxTotalThrustN := ((singleThrustG * 4.0) / 1000.0) * GravityMPerS2
	weightN := (totalMassG / 1000.0) * GravityMPerS2

	if maxTotalThrustN <= weightN { return 0.0 }

	// Calculate Max Pitch Speed (Absolute theoretical speed limit of the propeller)
	rps := getLoadedRps(motor, prop, battery)
	aero := getPropAeroData(prop)
	pitchSpeedMps := rps * aero.PitchMeters

	// Calculate Pitch Angle
	pitchAngleRad := math.Acos(weightN / maxTotalThrustN)
	if pitchAngleRad > 1.22 { pitchAngleRad = 1.22 }
	forwardThrustN := maxTotalThrustN * math.Sin(pitchAngleRad)

	// Calculate Drag Area
	wheelbaseMeters := float64(frame.GetWheelbaseMm()) / 1000.0
	aEff := ((wheelbaseMeters / 2.0 * wheelbaseMeters / 3.0) * math.Sin(pitchAngleRad)) + 
	        ((wheelbaseMeters / 3.0 * 0.05) * math.Cos(pitchAngleRad))
	aEff += math.Pow(wheelbaseMeters/2.0, 2) * 0.3
	cD := 1.4

	// Dynamic Thrust decreases linearly as speed approaches Pitch Speed
	// T(v) = T_forward * (1 - v / v_pitch)
	// We solve for v where T(v) = Drag = 0.5 * rho * Cd * A * v^2
	// 0.5 * rho * Cd * A * v^2 + (T_forward / v_pitch) * v - T_forward = 0
	
	a := 0.5 * AirDensityKgPerM3 * cD * aEff
	b := forwardThrustN / pitchSpeedMps
	c := -forwardThrustN

	// Quadratic formula: v = (-b + sqrt(b^2 - 4ac)) / 2a
	discriminant := (b * b) - (4.0 * a * c)
	if discriminant < 0 { return 0.0 }
	
	maxVelocityMps := (-b + math.Sqrt(discriminant)) / (2.0 * a)

	return maxVelocityMps * 3.6
}
