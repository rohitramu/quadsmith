package evaluator

import (
	pb "quadsmith/api/gen/quadsmith"
)

type Evaluator struct {}

func NewEvaluator() *Evaluator { return &Evaluator{} }

func ptrFloat64(v float64) *float64 { return &v }

func (e *Evaluator) Evaluate(buildComponents []*pb.Component, battery *pb.Component, payloadWeightG float64) (*pb.EvaluateBuildResponse, error) {
	totalMassG := CalculateTotalMass(buildComponents, battery, payloadWeightG)

	var motor *pb.Motor
	var prop *pb.Propeller
	var bat *pb.Battery
	var frame *pb.Frame

	if battery != nil { bat = battery.GetBattery() }
	for _, comp := range buildComponents {
		if isMotor(comp) { motor = comp.GetMotor() }
		if isProp(comp) { prop = comp.GetPropeller() }
		if isFrame(comp) { frame = comp.GetFrame() }
	}

	twr, hoverS, mixedS, topSpeed := 0.0, 0.0, 0.0, 0.0

	if motor != nil && prop != nil && bat != nil && totalMassG > 0 {
		singleMotorThrust := EstimateMaxThrustGrams(motor, prop, bat)
		totalThrust := singleMotorThrust * 4.0 
		twr = totalThrust / totalMassG
		hoverS, mixedS = EstimateFlightTimeSeconds(totalMassG, motor, prop, bat)
		if frame != nil {
			topSpeed = EstimateTopSpeedKph(totalMassG, motor, prop, bat, frame)
		}
	}

	res := (&pb.EvaluatorResult_builder{
		TotalMassG:           ptrFloat64(totalMassG),
		ThrustToWeightRatio:  ptrFloat64(twr),
		HoverFlightTimeS:     ptrFloat64(hoverS),
		FreestyleFlightTimeS: ptrFloat64(mixedS),
		MaxSpeedKph:          ptrFloat64(topSpeed),
	}).Build()

	return (&pb.EvaluateBuildResponse_builder{
		Result: res,
	}).Build(), nil
}
