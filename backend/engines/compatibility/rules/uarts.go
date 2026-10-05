package rules

import (
	"fmt"
	pb "quadsmith/api/gen/quadsmith"
)

type UartRule struct{}

func (r *UartRule) Name() string {
	return "UART & Serial Ports"
}

func (r *UartRule) Check(components []*pb.Component) *pb.CompatibilityResult {
	var fc *pb.Component
	var peripherals []*pb.Component

	requiredUarts := 0

	for _, c := range components {
		if isFC(c) {
			fc = c
		} else if isVTX(c) || 
		          isRX(c) || 
		          isGPS(c) {
			peripherals = append(peripherals, c)
			// For this MVP, we assume each of these major peripherals consumes exactly 1 full UART.
			// In the future, we can do pin-level resolution (e.g. SmartAudio only needs 1 TX pin).
			requiredUarts++
		}
	}

	// If there's no FC, we can't evaluate the UART count
	if fc == nil || fc.GetFlightController() == nil {
		return nil
	}

	availableUarts := len(fc.GetFlightController().GetUartConnections())

	var messages []*pb.CompatibilityMessage
	var involvedIds []string
	involvedIds = append(involvedIds, fc.GetResource().GetId())

	for _, p := range peripherals {
		involvedIds = append(involvedIds, p.GetResource().GetId())
	}

	if requiredUarts > availableUarts {
		msg := &pb.CompatibilityMessage{}
		msg.SetSeverity(pb.CompatibilityMessage_DEFINITE_INCOMPATIBILITY)
		msg.SetMessage(fmt.Sprintf("Peripherals require %d UARTs, but the Flight Controller only has %d available.", requiredUarts, availableUarts))
		msg.SetResolution("Choose a Flight Controller with more UARTs, or remove a peripheral (like GPS).")
		messages = append(messages, msg)
	}

	// We always return the metric, even if it's compatible, so the UI can show a "UART Usage Gauge"
	res := &pb.CompatibilityResult{}
	res.SetCheckerName(r.Name())
	res.SetComponentIds(involvedIds)
	res.SetMessages(messages)
	um := &pb.UartMetric{}
	um.SetRequired(int32(requiredUarts))
	um.SetAvailable(int32(availableUarts))
	cm := &pb.CompatibilityMetric{}
	cm.SetUart(um)
	res.SetMetrics(cm)
	return res
}








