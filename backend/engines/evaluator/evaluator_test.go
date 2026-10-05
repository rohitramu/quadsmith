package evaluator

import (
	"google.golang.org/protobuf/proto"
	"testing"
	pb "quadsmith/api/gen/quadsmith"
)



func TestEvaluator_AeroPhysics(t *testing.T) {
	eval := NewEvaluator()
	
	f := pb.Component_builder{WeightG: proto.Float64(120.0), Frame: pb.Frame_builder{WheelbaseMm: proto.Int32(220)}.Build()}.Build()
	m := pb.Component_builder{WeightG: proto.Float64(30.0), Motor: pb.Motor_builder{KvRating: proto.Int32(2400)}.Build()}.Build()
	p := pb.Component_builder{WeightG: proto.Float64(5.0), Propeller: pb.Propeller_builder{DiameterMm: proto.Float64(5.1 * 25.4), PitchMm: proto.Float64(3.0 * 25.4)}.Build()}.Build()

	components := []*pb.Component{f, m, p}

	battery := pb.Component_builder{WeightG: proto.Float64(180.0), Battery: pb.Battery_builder{CellCountS: proto.Int32(4), CapacityMah: proto.Int32(1300)}.Build()}.Build()

	res, err := eval.Evaluate(components, battery, 80.0) // 530g total
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	r := res.GetResult()

	t.Logf("Mass: %.1fg", r.GetTotalMassG())
	t.Logf("TWR: %.2f", r.GetThrustToWeightRatio())
	t.Logf("Hover: %.1f mins", r.GetHoverFlightTimeS() / 60.0)
	t.Logf("Mixed: %.1f mins", r.GetFreestyleFlightTimeS() / 60.0)
	t.Logf("Top Speed: %.1f kph", r.GetMaxSpeedKph())

	if r.GetThrustToWeightRatio() < 5.0 || r.GetThrustToWeightRatio() > 12.0 {
		t.Errorf("TWR unrealistic: %.2f", r.GetThrustToWeightRatio())
	}

	if r.GetMaxSpeedKph() < 100.0 || r.GetMaxSpeedKph() > 250.0 {
		t.Errorf("Top Speed unrealistic for 5-inch 4S: %.1f kph", r.GetMaxSpeedKph())
	}
}
