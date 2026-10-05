package rules

import (
	"fmt"
	pb "quadsmith/api/gen/quadsmith"
)

type VoltageRule struct{}

func (r *VoltageRule) Name() string {
	return "Voltage Limits"
}

type PeripheralVoltage struct {
	ComponentId string
	Name        string
	MinV        float64
	MaxV        float64
}

func (r *VoltageRule) Check(components []*pb.Component) *pb.CompatibilityResult {
	var battery *pb.Component
	var fc *pb.Component
	var messages []*pb.CompatibilityMessage
	var involvedComponentIds []string

	for _, c := range components {
		if isBattery(c) {
			battery = c
		} else if isFC(c) {
			fc = c
		}
	}

	if battery == nil || battery.GetBattery() == nil {
		return nil
	}
	involvedComponentIds = append(involvedComponentIds, battery.GetId())

	batteryMaxV := float64(battery.GetBattery().GetCellCountS()) * 4.2

	var becVoltages []float64
	if fc != nil && fc.GetFlightController() != nil {
		involvedComponentIds = append(involvedComponentIds, fc.GetId())
		for _, bec := range fc.GetFlightController().GetBecOutputs() {
			becVoltages = append(becVoltages, bec.GetVoltageV())
		}
	}

	var peripherals []PeripheralVoltage

	for _, c := range components {
		switch c.WhichType() {
		case pb.Component_FlightController_case:
			p := c.GetFlightController()
			if batteryMaxV > float64(p.GetInputVoltageMaxV()) && p.GetInputVoltageMaxV() > 0 {
				msg := &pb.CompatibilityMessage{}
				msg.SetSeverity(pb.CompatibilityMessage_DEFINITE_INCOMPATIBILITY)
				msg.SetMessage(fmt.Sprintf("Battery max voltage (%.1fV) exceeds Flight Controller safe input voltage (%.1fV).", batteryMaxV, p.GetInputVoltageMaxV()))
				msg.SetResolution("Choose a battery with fewer cells or an FC that supports higher voltage.")
				messages = append(messages, msg)
			}
		case pb.Component_Esc_case:
			p := c.GetEsc()
			if batteryMaxV > float64(p.GetInputVoltageMaxV()) && p.GetInputVoltageMaxV() > 0 {
				msg := &pb.CompatibilityMessage{}
				msg.SetSeverity(pb.CompatibilityMessage_DEFINITE_INCOMPATIBILITY)
				msg.SetMessage(fmt.Sprintf("Battery max voltage (%.1fV) exceeds ESC safe input voltage (%.1fV).", batteryMaxV, p.GetInputVoltageMaxV()))
				msg.SetResolution("Choose a battery with fewer cells or an ESC that supports higher voltage.")
				messages = append(messages, msg)
			}
		case pb.Component_Vtx_case:
			p := c.GetVtx()
			if p != nil {
				peripherals = append(peripherals, PeripheralVoltage{ComponentId: c.GetId(), Name: "VTX", MinV: p.GetInputVoltageMinV(), MaxV: p.GetInputVoltageMaxV()})
			}
		case pb.Component_Camera_case:
			p := c.GetCamera()
			if p != nil {
				peripherals = append(peripherals, PeripheralVoltage{ComponentId: c.GetId(), Name: "Camera", MinV: p.GetInputVoltageMinV(), MaxV: p.GetInputVoltageMaxV()})
			}
		case pb.Component_Receiver_case:
			p := c.GetReceiver()
			if p != nil {
				peripherals = append(peripherals, PeripheralVoltage{ComponentId: c.GetId(), Name: "Receiver", MinV: p.GetInputVoltageMinV(), MaxV: p.GetInputVoltageMaxV()})
			}
		case pb.Component_Gps_case:
			p := c.GetGps()
			if p != nil {
				peripherals = append(peripherals, PeripheralVoltage{ComponentId: c.GetId(), Name: "GPS", MinV: p.GetInputVoltageMinV(), MaxV: p.GetInputVoltageMaxV()})
			}
		}
	}

	for _, p := range peripherals {
		if p.MaxV == 0 {
			continue 
		}

		canUseVbat := batteryMaxV <= p.MaxV
		
		canUseBec := false
		for _, becV := range becVoltages {
			if becV >= p.MinV && becV <= p.MaxV {
				canUseBec = true
				break
			}
		}

		if !canUseVbat && !canUseBec {
			involvedComponentIds = append(involvedComponentIds, p.ComponentId)
			msg := &pb.CompatibilityMessage{}
			msg.SetSeverity(pb.CompatibilityMessage_DEFINITE_INCOMPATIBILITY)
			msg.SetMessage(fmt.Sprintf("%s requires %.1fV-%.1fV. It cannot handle the battery (%.1fV) directly, and the Flight Controller lacks a compatible BEC.", p.Name, p.MinV, p.MaxV, batteryMaxV))
			msg.SetResolution(fmt.Sprintf("Select a Flight Controller with a BEC in the %.1fV-%.1fV range, or a %s that supports %.1fV.", p.MinV, p.MaxV, p.Name, batteryMaxV))
			messages = append(messages, msg)
		}
	}

	if len(messages) == 0 {
		return nil
	}

	vm := &pb.VoltageMetric{}
	vm.SetValue(batteryMaxV)
	vm.SetUnit("V")

	cm := &pb.CompatibilityMetric{}
	cm.SetVoltage(vm)

	res := &pb.CompatibilityResult{}
	res.SetCheckerName(r.Name())
	res.SetComponentIds(involvedComponentIds)
	res.SetMetrics(cm)
	res.SetMessages(messages)
	
	return res
}
