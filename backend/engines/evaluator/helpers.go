package evaluator

import pb "quadsmith/api/gen/quadsmith"

func isMotor(c *pb.Component) bool { return c.WhichType() == pb.Component_Motor_case }
func isProp(c *pb.Component) bool { return c.WhichType() == pb.Component_Propeller_case }
func isFrame(c *pb.Component) bool { return c.WhichType() == pb.Component_Frame_case }
