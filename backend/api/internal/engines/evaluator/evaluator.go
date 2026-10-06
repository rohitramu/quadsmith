package evaluator

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
)

type EvaluatorServiceHandler struct {
	db *pgxpool.Pool
}

func NewEvaluatorServiceHandler(db *pgxpool.Pool) *EvaluatorServiceHandler {
	return &EvaluatorServiceHandler{db: db}
}

func (s *EvaluatorServiceHandler) EvaluateBuild(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
	b := req.Msg.GetBuild()
	if b == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("build is required"))
	}
	
	totalWeight := req.Msg.GetPayloadWeightG()
	var errors []string
	var warnings []string

	// 1. Fetch Frame
	var frame *pb.Frame
	if b.FrameUuid != "" {
		f, err := pb.GetFrame(ctx, s.db, b.FrameUuid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Frame not found: %s", b.FrameUuid))
		} else {
			frame = f
			totalWeight += frame.WeightG
		}
	}

	// 2. Fetch Motor
	var motor *pb.Motor
	if b.MotorUuid != "" {
		m, err := pb.GetMotor(ctx, s.db, b.MotorUuid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Motor not found: %s", b.MotorUuid))
		} else {
			motor = m
			totalWeight += (motor.WeightG * 4) // Quadcopter = 4 motors
		}
	}

	// 3. Fetch Battery
	var battery *pb.Battery
	if b.BatteryUuid != "" {
		bat, err := pb.GetBattery(ctx, s.db, b.BatteryUuid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Battery not found: %s", b.BatteryUuid))
		} else {
			battery = bat
			totalWeight += battery.WeightG
		}
	}

	// 4. Fetch Propeller
	var prop *pb.Propeller
	if b.PropellerUuid != "" {
		p, err := pb.GetPropeller(ctx, s.db, b.PropellerUuid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Propeller not found: %s", b.PropellerUuid))
		} else {
			prop = p
			totalWeight += (prop.WeightG * 4) // Quadcopter = 4 props
		}
	}

	// 5. Fetch ESCs
	var totalEscs int32 = 0
	var maxAmps int32 = 0
	for _, id := range b.EscUuids {
		esc, err := pb.GetEsc(ctx, s.db, id)
		if err == nil {
			totalWeight += esc.WeightG
			totalEscs += esc.MaxMotors
			if esc.ContinuousAmps > maxAmps {
				maxAmps = esc.ContinuousAmps
			}
		}
	}

	// 6. Fetch Flight Controller (and Internal ESC)
	if b.FlightControllerUuid != "" {
		fc, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid)
		if err == nil {
			totalWeight += fc.WeightG
			if fc.InternalEscUuid != "" {
				esc, err := pb.GetEsc(ctx, s.db, fc.InternalEscUuid)
				if err == nil {
					totalEscs += esc.MaxMotors
					if esc.ContinuousAmps > maxAmps {
						maxAmps = esc.ContinuousAmps
					}
				}
			}
		}
	}

	// System Validation
	if totalEscs < 4 && totalEscs > 0 {
		errors = append(errors, fmt.Sprintf("Not enough ESCs: need 4, have %d", totalEscs))
	}
	
	// Physics Estimation (Placeholder for MVP)
	var thrustToWeight float32 = 0
	var hoverThrottle float32 = 0
	var flightTime float32 = 0

	if motor != nil && prop != nil && battery != nil {
		voltage := float32(battery.CellCountS) * 3.7
		
		// Naive thrust formula
		statorVol := motor.StatorDiameterMm * motor.StatorHeightMm
		thrustPerMotor := (statorVol * float32(motor.Kv) * voltage * prop.DiameterInches * prop.PitchInches) / 1500.0
		totalThrust := thrustPerMotor * 4
		
		if totalWeight > 0 {
			thrustToWeight = totalThrust / totalWeight
			hoverThrottle = (1.0 / thrustToWeight) * 100.0
		}
		
		if hoverThrottle > 100 {
			errors = append(errors, "Drone is too heavy to take off (Hover throttle > 100%)")
		} else if hoverThrottle > 50 {
			warnings = append(warnings, "Drone will be very sluggish (Hover throttle > 50%)")
		}
		
		// Flight time estimation
		// Assume hover takes (totalWeight / 4) grams of thrust per motor
		// Amps = thrust / efficiency(g/W) / voltage
		hoverAmpsPerMotor := (totalWeight / 4.0) / 3.0 / voltage
		totalHoverAmps := hoverAmpsPerMotor * 4.0
		
		if totalHoverAmps > float32(maxAmps*4) && maxAmps > 0 {
			warnings = append(warnings, "Hover amps exceeds ESC continuous rating")
		}

		if totalHoverAmps > 0 {
			flightTime = (float32(battery.CapacityMah) / 1000.0) / totalHoverAmps * 60.0 // minutes
		}
	} else {
		warnings = append(warnings, "Need a Motor, Propeller, and Battery to run physics estimation")
	}

	res := &pb.EvaluateBuildResponse{
		TotalWeightG:           totalWeight,
		ThrustToWeightRatio:    thrustToWeight,
		HoverThrottlePercent:   hoverThrottle,
		EstimatedFlightTimeMin: flightTime,
		Warnings:               warnings,
		Errors:                 errors,
	}

	return connect.NewResponse(res), nil
}
