package evaluator

import (
	"context"
	"fmt"
	"math"

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
		f, err := pb.GetFrame(ctx, s.db, b.FrameUuid, nil)
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
		m, err := pb.GetMotor(ctx, s.db, b.MotorUuid, nil)
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
		bat, err := pb.GetBattery(ctx, s.db, b.BatteryUuid, nil)
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
		p, err := pb.GetPropeller(ctx, s.db, b.PropellerUuid, nil)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Propeller not found: %s", b.PropellerUuid))
		} else {
			prop = p
			totalWeight += (prop.WeightG * 4) // Quadcopter = 4 props
		}
	}

	// 5. Fetch Electronic Speed Controllers
	var totalEscs uint32 = 0
	var maxAmps float32 = 0
	for _, id := range b.ElectronicSpeedControllerUuids {
		esc, err := pb.GetElectronicSpeedController(ctx, s.db, id, nil)
		if err == nil {
			totalWeight += esc.WeightG
			totalEscs += esc.MaxMotors
			if esc.MotorCurrentMaxA > maxAmps {
				maxAmps = esc.MotorCurrentMaxA
			}
		}
	}

	// 6. Fetch Flight Controller (and Internal Electronic Speed Controller)
	if b.FlightControllerUuid != "" {
		fc, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil)
		if err == nil {
			totalWeight += fc.WeightG
			if fc.GetInternalElectronicSpeedControllerUuid() != "" {
				esc, err := pb.GetElectronicSpeedController(ctx, s.db, fc.GetInternalElectronicSpeedControllerUuid(), nil)
				if err == nil {
					totalEscs += esc.MaxMotors
					if esc.MotorCurrentMaxA > maxAmps {
						maxAmps = esc.MotorCurrentMaxA
					}
				}
			}
		}
	}

	// 7. Fetch Video Transmitter (Standalone or Internal to FC)
	if b.VideoTransmitterUuid != "" {
		videoTransmitter, err := pb.GetVideoTransmitter(ctx, s.db, b.VideoTransmitterUuid, nil)
		if err == nil {
			totalWeight += videoTransmitter.WeightG
		}
	} else if b.FlightControllerUuid != "" {
		fc, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil)
		if err == nil && fc.GetInternalVideoTransmitterUuid() != "" {
			videoTransmitter, err := pb.GetVideoTransmitter(ctx, s.db, fc.GetInternalVideoTransmitterUuid(), nil)
			if err == nil {
				totalWeight += videoTransmitter.WeightG
			}
		}
	}

	// 8. Fetch Cameras
	for _, cid := range b.CameraUuids {
		cam, err := pb.GetCamera(ctx, s.db, cid, nil)
		if err == nil {
			totalWeight += cam.WeightG
		}
	}

	// 9. Fetch Receivers
	for _, rid := range b.ReceiverUuids {
		rx, err := pb.GetReceiver(ctx, s.db, rid, nil)
		if err == nil {
			totalWeight += rx.WeightG
		}
	}

	// 10. Fetch Antennas
	for _, aid := range b.AntennaUuids {
		ant, err := pb.GetAntenna(ctx, s.db, aid, nil)
		if err == nil {
			totalWeight += ant.WeightG
		}
	}

	// 11. Fetch GPS Receiver
	if b.GetGpsReceiverUuid() != "" {
		gps, err := pb.GetGpsReceiver(ctx, s.db, b.GetGpsReceiverUuid(), nil)
		if err == nil {
			totalWeight += gps.WeightG
		}
	}

	// System Validation
	// TODO: Implement mechanical compatibility checks (e.g. Flight Controller mounting hole spacing vs Frame mounts).
	// TODO: Implement electrical compatibility checks (e.g. Battery Voltage vs FC max voltage, Receiver protocol vs FC UARTs).
	if totalEscs < 4 && totalEscs > 0 {
		errors = append(errors, fmt.Sprintf("Not enough ESCs: need 4, have %d", totalEscs))
	}

	payloadWeight := req.Msg.GetPayloadWeightG()
	baseWeight := totalWeight - payloadWeight

	// Aerodynamic Physics Estimation
	thrustToWeight, hoverThrottle, flightTime, physErrors, physWarnings := CalculatePhysics(
		motor,
		prop,
		battery,
		baseWeight,
		payloadWeight,
		maxAmps,
	)
	errors = append(errors, physErrors...)
	warnings = append(warnings, physWarnings...)

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

// CalculatePhysics computes aerodynamic static thrust, thrust-to-weight ratio,
// quadratic hover throttle percentage, and estimated flight time.
func CalculatePhysics(
	motor *pb.Motor,
	prop *pb.Propeller,
	battery *pb.Battery,
	baseWeight float32,
	payloadWeight float32,
	maxEscAmps float32,
) (thrustToWeight float32, hoverThrottle float32, flightTime float32, errors []string, warnings []string) {
	if motor == nil || prop == nil || battery == nil {
		warnings = append(warnings, "Need a Motor, Propeller, and Battery to run physics estimation")
		return 0, 0, 0, errors, warnings
	}

	totalWeight := baseWeight + payloadWeight

	cellCount := float32(battery.CellCountS)
	if cellCount == 0 {
		cellCount = 4
	}
	// Nominal loaded voltage under moderate load (~3.7V per cell for LiPo)
	voltage := cellCount * 3.7

	propDiaMm := prop.DiameterMm
	if propDiaMm <= 0 {
		propDiaMm = 127.0 // default 5-inch
	}
	propPitchMm := prop.PitchMm
	if propPitchMm <= 0 {
		propPitchMm = propDiaMm * 0.7
	}
	propBlades := prop.Blades
	if propBlades < 2 {
		propBlades = 3
	}

	diaIn := propDiaMm / 25.4
	pitchIn := propPitchMm / 25.4

	// 1. Non-dimensional thrust coefficient Ct based on momentum and blade element theory:
	// Ct_2blade = 0.045 + 0.090 * (P / D)
	// Blade solidity scaling: (blades / 2)^0.45
	pOverD := pitchIn / diaIn
	ct2Blade := 0.045 + 0.090*pOverD
	bladeFactor := float32(math.Pow(float64(propBlades)/2.0, 0.45))
	ct := ct2Blade * bladeFactor

	// 2. Motor stator volume (mm^3) as proxy for torque capability
	statorD := motor.StatorDiameterMm
	statorH := motor.StatorHeightMm
	statorVol := float32(math.Pi/4.0) * statorD * statorD * statorH
	if statorVol <= 0 {
		// Fallback estimate from motor weight if stator dimensions missing
		if motor.WeightG > 0 {
			statorVol = motor.WeightG * 80.0
		} else {
			statorVol = 2600.0 // standard 2207 size
		}
	}

	// 3. Propeller aerodynamic torque demand scale: D^4 * P * sqrt(blades / 2)
	propTorqueScale := math.Pow(float64(diaIn), 4) * float64(pitchIn) * math.Sqrt(float64(propBlades)/2.0)
	torqueRatio := float64(statorVol) / math.Max(1.0, propTorqueScale)

	// Full-throttle loaded RPM: well-matched motor reaches ~76% of no-load (Kv * V)
	rpmLoadFactor := 0.76 * math.Min(1.08, math.Max(0.60, math.Pow(torqueRatio/1.08, 0.15)))
	loadedRpm := float32(float64(float32(motor.Kv)*voltage) * rpmLoadFactor)

	// 4. Static thrust (momentum / blade element theory):
	// T = Ct * rho * n^2 * D^4
	const rho = 1.225 // kg/m^3 standard sea-level air density
	dM := float64(propDiaMm) / 1000.0
	n := float64(loadedRpm) / 60.0
	thrustNewtons := float64(ct) * rho * (n * n) * math.Pow(dM, 4)
	thrustPerMotor := float32(thrustNewtons * 101.97162) // 1 N = 101.97162 g
	totalThrust := thrustPerMotor * 4.0

	// 5. Thrust-to-weight ratio and realistic hover throttle with payload scaling
	if totalWeight > 0 {
		thrustToWeight = totalThrust / totalWeight
	}

	safeBase := float64(baseWeight)
	if safeBase <= 0 {
		safeBase = float64(totalWeight)
	}

	if safeBase > 0 && totalThrust > 0 {
		baseTwr := float64(totalThrust) / safeBase
		baseHover := math.Pow(1.0/math.Max(0.1, baseTwr), 0.65) * 100.0
		weightRatio := float64(totalWeight) / safeBase
		hoverThrottle = float32(baseHover * math.Pow(weightRatio, 1.6))
	} else {
		hoverThrottle = 100.0
	}

	if thrustToWeight <= 1.0 && hoverThrottle < 100.0 {
		hoverThrottle = 100.0
	}

	if hoverThrottle > 100.0 {
		errors = append(errors, "Drone is too heavy to take off (Hover throttle > 100%)")
	} else if hoverThrottle > 50.0 {
		warnings = append(warnings, "Drone will be very sluggish (Hover throttle > 50%)")
	}

	// 6. Realistic mixed/cruising flight time:
	// Multirotor flight efficiency in mixed forward flight / cruising:
	// Larger props have higher efficiency due to lower disk loading (2.2 - 3.2 g/W).
	// Base electronics (VTX, Camera, FC, RX) consume ~10W.
	effFlight := float32(math.Min(4.5, math.Max(1.8, 1.8+0.18*float64(diaIn))))
	const pElectronics = 10.0 // Watts
	weightRatio := float64(totalWeight) / safeBase
	flightWatts := (float32(safeBase)/effFlight)*float32(math.Pow(weightRatio, 1.35)) + pElectronics
	totalFlightAmps := flightWatts / voltage

	if totalFlightAmps > float32(maxEscAmps*4) && maxEscAmps > 0 {
		warnings = append(warnings, "Hover amps exceeds ESC continuous rating")
	}

	if totalFlightAmps > 0 && battery.CapacityMah > 0 {
		usableAh := (float32(battery.CapacityMah) / 1000.0) * 0.80 // 80% usable capacity (20% safety margin)
		flightTime = (usableAh / totalFlightAmps) * 60.0           // minutes
	}

	return thrustToWeight, hoverThrottle, flightTime, errors, warnings
}
