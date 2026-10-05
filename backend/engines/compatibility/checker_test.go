package compatibility_test

import (
	"google.golang.org/protobuf/proto"
	"testing"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/engines/compatibility"
	"quadsmith/engines/compatibility/rules"
)

func TestVoltageRule_WithBec(t *testing.T) {
	engine := compatibility.NewEngine(&rules.VoltageRule{})

	b_bldr := pb.Battery_builder{CellCountS: proto.Int32(6)}
	battery := pb.Component_builder{Id: proto.String("bat-1"), Battery: b_bldr.Build()}.Build()

	fc_bldr := pb.FlightController_builder{BecOutputs: []*pb.BecOutput{pb.BecOutput_builder{VoltageV: proto.Float64(9.0)}.Build()}}
	fc := pb.Component_builder{Id: proto.String("fc-1"), FlightController: fc_bldr.Build()}.Build()

	vtx5v := pb.Component_builder{Id: proto.String("vtx-5v"), Vtx: pb.Vtx_builder{InputVoltageMinV: proto.Float64(4.5), InputVoltageMaxV: proto.Float64(5.5)}.Build()}.Build()

	vtxO3 := pb.Component_builder{Id: proto.String("vtx-o3"), Vtx: pb.Vtx_builder{InputVoltageMinV: proto.Float64(7.0), InputVoltageMaxV: proto.Float64(26.0)}.Build()}.Build()
	
	vtxStrict9v := pb.Component_builder{Id: proto.String("vtx-strict9v"), Vtx: pb.Vtx_builder{InputVoltageMinV: proto.Float64(7.0), InputVoltageMaxV: proto.Float64(12.0)}.Build()}.Build()

	resp1, _ := engine.Evaluate(&pb.CheckCompatibilityRequest{}, []*pb.Component{battery, fc, vtx5v})
	if len(resp1.GetResults()) == 0 {
		t.Fatalf("Expected voltage incompatibility for 5V VTX without 5V BEC")
	}

	resp2, _ := engine.Evaluate(&pb.CheckCompatibilityRequest{}, []*pb.Component{battery, fc, vtxStrict9v})
	if len(resp2.GetResults()) != 0 {
		t.Fatalf("Expected NO incompatibility for strict 9V VTX with 9V BEC, got an error")
	}

	resp3, _ := engine.Evaluate(&pb.CheckCompatibilityRequest{}, []*pb.Component{battery, vtxO3})
	if len(resp3.GetResults()) != 0 {
		t.Fatalf("Expected NO incompatibility for 26V VTX running directly on 25.2V Vbat")
	}
}

func TestUartRule(t *testing.T) {
	engine := compatibility.NewEngine(&rules.UartRule{})

	fc := pb.Component_builder{
		Id: proto.String("fc-1"), 
		FlightController: pb.FlightController_builder{
			UartConnections: []*pb.UartPort{
				pb.UartPort_builder{Id: proto.String("UART1"), SupportsRx: proto.Bool(true), SupportsTx: proto.Bool(true)}.Build(),
				pb.UartPort_builder{Id: proto.String("UART2"), SupportsRx: proto.Bool(true), SupportsTx: proto.Bool(true)}.Build(),
			},
		}.Build(),
	}.Build()

	vtx := pb.Component_builder{Id: proto.String("vtx-1"), Vtx: pb.Vtx_builder{}.Build()}.Build()
	rx := pb.Component_builder{Id: proto.String("rx-1"), Receiver: pb.Receiver_builder{}.Build()}.Build()
	gps := pb.Component_builder{Id: proto.String("gps-1"), Gps: pb.Gps_builder{}.Build()}.Build()

	resp1, _ := engine.Evaluate(&pb.CheckCompatibilityRequest{}, []*pb.Component{fc, vtx, rx})
	
	if len(resp1.GetResults()) == 0 {
		t.Fatalf("Expected a result containing the UART Metric")
	}
	if len(resp1.GetResults()[0].GetMessages()) != 0 {
		t.Fatalf("Expected NO incompatibility messages, got %v", resp1.GetResults()[0].GetMessages())
	}

	metric := resp1.GetResults()[0].GetMetrics().GetUart()
	if metric.GetRequired() != 2 || metric.GetAvailable() != 2 {
		t.Errorf("Expected 2 required and 2 available, got %d and %d", metric.GetRequired(), metric.GetAvailable())
	}

	resp2, _ := engine.Evaluate(&pb.CheckCompatibilityRequest{}, []*pb.Component{fc, vtx, rx, gps})
	if len(resp2.GetResults()[0].GetMessages()) == 0 {
		t.Fatalf("Expected a UART shortage incompatibility")
	}
}
