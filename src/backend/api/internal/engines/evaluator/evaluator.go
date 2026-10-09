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

	var baseWeight float32 = 0
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
			baseWeight += frame.WeightG
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
			baseWeight += (motor.WeightG * 4) // Quadcopter = 4 motors
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
			baseWeight += battery.WeightG
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
			baseWeight += (prop.WeightG * 4) // Quadcopter = 4 props
		}
	}

	// 5. Fetch Electronic Speed Controllers
	var totalEscs uint32 = 0
	var maxAmps float32 = 0
	for _, id := range b.ElectronicSpeedControllerUuids {
		esc, err := pb.GetElectronicSpeedController(ctx, s.db, id, nil)
		if err == nil {
			baseWeight += esc.WeightG
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
			baseWeight += fc.WeightG
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
			baseWeight += videoTransmitter.WeightG
		}
	} else if b.FlightControllerUuid != "" {
		fc, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil)
		if err == nil && fc.GetInternalVideoTransmitterUuid() != "" {
			videoTransmitter, err := pb.GetVideoTransmitter(ctx, s.db, fc.GetInternalVideoTransmitterUuid(), nil)
			if err == nil {
				baseWeight += videoTransmitter.WeightG
			}
		}
	}

	// 8. Fetch Cameras
	for _, cid := range b.CameraUuids {
		cam, err := pb.GetCamera(ctx, s.db, cid, nil)
		if err == nil {
			baseWeight += cam.WeightG
		}
	}

	// 9. Fetch Receivers
	for _, rid := range b.ReceiverUuids {
		rx, err := pb.GetReceiver(ctx, s.db, rid, nil)
		if err == nil {
			baseWeight += rx.WeightG
		}
	}

	// 10. Fetch Antennas
	for _, aid := range b.AntennaUuids {
		ant, err := pb.GetAntenna(ctx, s.db, aid, nil)
		if err == nil {
			baseWeight += ant.WeightG
		}
	}

	// 11. Fetch GPS Receiver
	if b.GetGpsReceiverUuid() != "" {
		gps, err := pb.GetGpsReceiver(ctx, s.db, b.GetGpsReceiverUuid(), nil)
		if err == nil {
			baseWeight += gps.WeightG
		}
	}

	// System Validation
	// TODO: Implement mechanical compatibility checks (e.g. Flight Controller mounting hole spacing vs Frame mounts).
	// TODO: Implement electrical compatibility checks (e.g. Battery Voltage vs FC max voltage, Receiver protocol vs FC UARTs).
	if totalEscs < 4 && totalEscs > 0 {
		errors = append(errors, fmt.Sprintf("Not enough ESCs: need 4, have %d", totalEscs))
	}

	payloadWeight := req.Msg.GetPayloadWeightG()
	totalWeight := baseWeight + payloadWeight

	// Aerodynamic Physics Estimation
	phys := CalculatePhysics(
		motor,
		prop,
		battery,
		baseWeight,
		payloadWeight,
		maxAmps,
	)
	errors = append(errors, phys.Errors...)
	warnings = append(warnings, phys.Warnings...)

	res := &pb.EvaluateBuildResponse{
		TotalWeightG:           totalWeight,
		ThrustToWeightRatio:    phys.ThrustToWeightRatio,
		HoverThrottlePercent:   phys.HoverThrottlePercent,
		HoverRpm:               phys.HoverRpm,
		EstimatedFlightTimeMin: phys.EstimatedFlightTimeMin,
		MinFlightTimeMin:       phys.MinFlightTimeMin,
		MaxFlightTimeMin:       phys.MaxFlightTimeMin,
		MaxAccelerationMps2:    phys.MaxAccelerationMps2,
		TopSpeedKmh:            phys.TopSpeedKmh,
		Warnings:               warnings,
		Errors:                 errors,
	}

	return connect.NewResponse(res), nil
}

// PhysicsResult contains the calculated aerodynamic, electrical, and flight performance metrics.
type PhysicsResult struct {
	ThrustToWeightRatio    float32
	HoverThrottlePercent   float32
	HoverRpm               uint32
	EstimatedFlightTimeMin float32
	MinFlightTimeMin       float32
	MaxFlightTimeMin       float32
	MaxAccelerationMps2    float32
	TopSpeedKmh            float32
	Errors                 []string
	Warnings               []string
}

// CalculatePhysics computes aerodynamic static thrust, thrust-to-weight ratio,
// hover throttle percentage, hover propeller RPM, estimated flight time range
// (min, max, and mixed), max vertical acceleration, and terminal top speed.
func CalculatePhysics(
	motor *pb.Motor,
	prop *pb.Propeller,
	battery *pb.Battery,
	baseWeight float32,
	payloadWeight float32,
	maxEscAmps float32,
) PhysicsResult {
	if motor == nil || prop == nil || battery == nil {
		return PhysicsResult{
			Warnings: []string{"Need a Motor, Propeller, and Battery to run physics estimation"},
		}
	}

	var (
		thrustToWeight      float32
		hoverThrottle       float32
		hoverRpm            uint32
		flightTime          float32
		minFlightTime       float32
		maxFlightTime       float32
		maxAccelerationMps2 float32
		topSpeedKmh         float32
		errors              []string
		warnings            []string
	)

	safeBase := float64(baseWeight)
	if safeBase <= 0 {
		safeBase = float64(baseWeight + payloadWeight)
	}
	totalWeight := baseWeight + payloadWeight

	cellCount := float32(battery.CellCountS)
	if cellCount == 0 {
		cellCount = 4
	}

	// 1. Full-throttle burst voltage with realistic high-C sag and payload draw:
	// A high-C LiPo cell delivers ~3.55V under 100% punchout on a bare quadcopter.
	// Additional payload increases baseline drain and raises internal resistance sag down to ~3.40V.
	payloadRatio := float32(math.Min(1.2, float64(payloadWeight)/math.Max(1.0, safeBase)))
	burstCellVoltage := float32(3.55) - 0.12*payloadRatio
	voltage := cellCount * burstCellVoltage

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

	// 2. Non-dimensional thrust coefficient Ct based on momentum and blade element theory:
	// Low Reynolds number multirotor blade element theory:
	// Ct_2blade = 0.046 + 0.072 * (P / D)
	// Multi-blade solidity factor: (blades / 2)^0.38
	pOverD := pitchIn / diaIn
	ct2Blade := 0.046 + 0.072*pOverD
	bladeFactor := float32(math.Pow(float64(propBlades)/2.0, 0.38))
	ct := ct2Blade * bladeFactor

	// 3. Motor stator volume (mm^3) as proxy for torque capability
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

	// 4. Propeller aerodynamic torque demand scale: D^4 * P * sqrt(blades / 2)
	propTorqueScale := math.Pow(float64(diaIn), 4) * float64(pitchIn) * math.Sqrt(float64(propBlades)/2.0)
	torqueRatio := float64(statorVol) / math.Max(1.0, propTorqueScale)

	// Full-throttle loaded RPM under static bollard condition (J = 0):
	// Aerodynamic torque limits motor to ~72% of no-load (Kv * V)
	rpmLoadFactor := 0.72 * math.Min(1.05, math.Max(0.55, math.Pow(torqueRatio/1.0, 0.15)))
	loadedRpm := float32(float64(float32(motor.Kv)*voltage) * rpmLoadFactor)

	// 5. Static thrust in open air (momentum / blade element theory):
	// T = Ct * rho * n^2 * D^4
	const rho = 1.225 // kg/m^3 standard sea-level air density
	dM := float64(propDiaMm) / 1000.0
	n := float64(loadedRpm) / 60.0
	thrustNewtons := float64(ct) * rho * (n * n) * math.Pow(dM, 4)
	rawThrustPerMotor := float32(thrustNewtons * 101.97162) // 1 N = 101.97162 g

	// 6. Airframe installation loss (arm shadow / frame obstruction) & payload aerodynamic blockage:
	// Multirotor arm blockage and prop wash impingement reduce net thrust by ~14% vs isolated test bench.
	// Additional payload (cameras, mounts) adds inflow/outflow aerodynamic blockage.
	const frameEfficiency = 0.86
	payloadObstruction := float32(1.0 / (1.0 + 0.08*float64(payloadRatio)))
	installedThrustPerMotor := rawThrustPerMotor * frameEfficiency * payloadObstruction
	totalThrust := installedThrustPerMotor * 4.0

	// 7. Thrust-to-weight ratio and realistic hover throttle directly coupled to TWR:
	if totalWeight > 0 {
		thrustToWeight = totalThrust / totalWeight
	}

	if thrustToWeight > 0 {
		// Aerodynamic hover throttle position:
		// Required thrust fraction is 1 / TWR. In multirotor flight dynamics and Betaflight,
		// the throttle curve maps to (1 / TWR)^gamma, transitioning from gamma=0.68 near stall
		// to gamma=0.78 at high TWR due to low disk loading.
		twrFactor := float32(math.Min(1.0, math.Max(0.0, float64(thrustToWeight-1.0)/5.0)))
		gamma := 0.68 + 0.10*twrFactor
		hoverThrottle = float32(math.Pow(1.0/float64(thrustToWeight), float64(gamma)) * 100.0)
	} else {
		hoverThrottle = 100.0
	}

	if hoverThrottle > 100.0 {
		errors = append(errors, "Drone is too heavy to take off (Hover throttle > 100%)")
	} else if hoverThrottle > 50.0 {
		warnings = append(warnings, "Drone will be very sluggish (Hover throttle > 50%)")
	}

	// Average propeller RPM at hover:
	// Assumes zero wind, perfect horizontal stability, sea-level air density (rho = 1.225 kg/m^3),
	// and standard humidity. Since static thrust scales with RPM^2 (T = k * RPM^2), at steady hover
	// where T_hover = TotalWeight:
	// (RPM_hover / RPM_loaded)^2 = T_hover / T_total = 1 / TWR
	// => RPM_hover = RPM_loaded / sqrt(TWR)
	// (Equivalently: n_hover = sqrt(T_raw_hover_N / (Ct * rho * D^4)))
	if thrustToWeight >= 1.0 && totalWeight > 0 {
		rpm := float64(loadedRpm) / math.Sqrt(float64(thrustToWeight))
		if rpm > 0 && !math.IsNaN(rpm) && !math.IsInf(rpm, 0) {
			hoverRpm = uint32(math.Round(rpm))
		}
	}

	// 8. Realistic flight time range across flight styles:
	// Multirotor flight efficiency in forward flight / cruising:
	// Larger props have higher efficiency due to lower disk loading (1.8 - 3.2 g/W).
	// Base electronics (VTX, Camera, FC, RX) consume ~10W.
	effFlight := float32(math.Min(4.5, math.Max(1.8, 1.8+0.18*float64(diaIn))))
	const pElectronics = 10.0 // Watts
	weightRatio := float64(totalWeight) / safeBase
	nominalVoltage := cellCount * 3.7
	cruiseWatts := (float32(safeBase)/effFlight)*float32(math.Pow(weightRatio, 1.35)) + pElectronics
	totalCruiseAmps := cruiseWatts / nominalVoltage

	if totalCruiseAmps > float32(maxEscAmps*4) && maxEscAmps > 0 {
		warnings = append(warnings, "Cruise amps exceeds ESC continuous rating")
	}

	if totalCruiseAmps > 0 && battery.CapacityMah > 0 {
		usableAh := (float32(battery.CapacityMah) / 1000.0) * 0.80 // 80% usable capacity (20% safety margin)
		maxFlightTime = (usableAh / totalCruiseAmps) * 60.0        // minutes (smooth cruise / cinematic)

		// Aggressive freestyle / acro / sustained punchouts:
		// High throttle bursts, dynamic braking, and PID stabilization draw ~1.9x cruise power
		aggressiveWatts := cruiseWatts * 1.9
		aggressiveAmps := aggressiveWatts / nominalVoltage
		minFlightTime = (usableAh / aggressiveAmps) * 60.0 // minutes (aggressive freestyle)

		// Mixed / moderate flight profile (average between aggressive and cruise)
		flightTime = (minFlightTime + maxFlightTime) / 2.0
	}

	// 9. Maximum vertical punchout acceleration (m/s^2):
	// Under 100% throttle punchout: F_net = F_thrust - F_gravity = (TWR - 1.0) * m * g
	// Upward vertical acceleration: a_max = (TWR - 1.0) * g (in m/s^2)
	const gAccel = 9.80665 // m/s^2 standard gravity
	if thrustToWeight > 1.0 {
		maxAccelerationMps2 = (thrustToWeight - 1.0) * float32(gAccel)
	} else {
		maxAccelerationMps2 = 0.0
	}

	// 10. Terminal forward top speed (km/h) in high-tilt forward flight:
	// In forward flight, propeller aerodynamic inflow unloads the motor to ~86% Kv*V.
	// Equilibrium forward velocity balances thrust against frontal parasitic drag and induced drag:
	// T_fwd = 0.5 * rho * CdA * V^2 + T_fwd * (V / V_pitch)^2
	if totalThrust > 0 && propPitchMm > 0 {
		loadedRpmFwd := float64(motor.Kv) * float64(voltage) * 0.86
		pitchSpeedMps := (loadedRpmFwd / 60.0) * (float64(propPitchMm) / 1000.0)

		cdABase := 0.005 + 0.0012*float64(diaIn)
		cdAPayload := 0.006 * math.Min(1.0, float64(payloadWeight)/math.Max(1.0, safeBase))
		cdA := cdABase + cdAPayload

		forwardThrustN := (float64(totalThrust) * 0.95) / 101.97162

		if pitchSpeedMps > 0 && forwardThrustN > 0 {
			denom := 0.5*rho*cdA + (forwardThrustN / (pitchSpeedMps * pitchSpeedMps))
			if denom > 0 {
				vMps := math.Sqrt(forwardThrustN / denom)
				topSpeedKmh = float32(vMps * 3.6)
			}
		}
	}

	return PhysicsResult{
		ThrustToWeightRatio:    thrustToWeight,
		HoverThrottlePercent:   hoverThrottle,
		HoverRpm:               hoverRpm,
		EstimatedFlightTimeMin: flightTime,
		MinFlightTimeMin:       minFlightTime,
		MaxFlightTimeMin:       maxFlightTime,
		MaxAccelerationMps2:    maxAccelerationMps2,
		TopSpeedKmh:            topSpeedKmh,
		Errors:                 errors,
		Warnings:               warnings,
	}
}
