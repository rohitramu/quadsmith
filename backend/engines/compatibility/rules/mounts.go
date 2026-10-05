package rules

import (
	"fmt"
	"strings"
	pb "quadsmith/api/gen/quadsmith"
)

type MountRule struct{}

func (r *MountRule) Name() string {
	return "Mounting Holes"
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func (r *MountRule) Check(components []*pb.Component) *pb.CompatibilityResult {
	var frame *pb.Component
	var motors []*pb.Component

	for _, c := range components {
		if isFrame(c) {
			frame = c
		}
		if isMotor(c) {
			motors = append(motors, c)
		}
	}

	if frame == nil || frame.GetFrame() == nil {
		return nil
	}

	frameSpec := frame.GetFrame()
	var messages []*pb.CompatibilityMessage
	var involvedIds []string
	involvedIds = append(involvedIds, frame.GetResource().GetId())

	for _, motor := range motors {
		if motor.GetMotor() == nil {
			continue
		}
		m := motor.GetMotor()
		
		if len(m.GetMountPatternIds()) > 0 {
			supported := false
			for _, p := range m.GetMountPatternIds() {
				if containsString(frameSpec.GetMotorMountPatternIds(), p) {
					supported = true
					break
				}
			}

			if !supported {
				involvedIds = append(involvedIds, motor.GetResource().GetId())
				msg := &pb.CompatibilityMessage{}
				msg.SetSeverity(pb.CompatibilityMessage_DEFINITE_INCOMPATIBILITY)
				msg.SetMessage(fmt.Sprintf("Motor mounting patterns [%s] are not supported by the frame.", strings.Join(m.GetMountPatternIds(), ", ")))
				msg.SetResolution("Choose a frame that supports this motor's mounting pattern, or a different motor.")
				messages = append(messages, msg)
			}
		}
	}

	if len(messages) == 0 {
		return nil
	}

	res := &pb.CompatibilityResult{}
	res.SetCheckerName(r.Name())
	res.SetComponentIds(involvedIds)
	res.SetMessages(messages)
	return res
}

func isFrame(c *pb.Component) bool { return c.WhichType() == pb.Component_Frame_case }
func isMotor(c *pb.Component) bool { return c.WhichType() == pb.Component_Motor_case }
func isFC(c *pb.Component) bool { return c.WhichType() == pb.Component_FlightController_case }
func isVTX(c *pb.Component) bool { return c.WhichType() == pb.Component_Vtx_case }
func isRX(c *pb.Component) bool { return c.WhichType() == pb.Component_Receiver_case }
func isGPS(c *pb.Component) bool { return c.WhichType() == pb.Component_Gps_case }
func isBattery(c *pb.Component) bool { return c.WhichType() == pb.Component_Battery_case }
