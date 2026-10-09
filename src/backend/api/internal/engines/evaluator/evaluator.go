package evaluator

import (
	"context"
	"fmt"
	"math"
	"strings"

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
	buildId := strings.TrimSpace(req.Msg.GetBuildId())
	if buildId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("build_id is required"))
	}
	var b *pb.Build
	if s.db != nil {
		fetched, err := pb.GetBuild(ctx, s.db, buildId, nil)
		if err != nil || fetched == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %s", buildId))
		}
		b = fetched
	}
	if b == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %s", buildId))
	}

	// Validate required fields in the request
	batteryId := strings.TrimSpace(req.Msg.GetBatteryId())
	if batteryId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("battery_id is required"))
	}

	// Validate required components in the build
	if err := ValidateBuildComponents(b); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var baseWeight float32 = 0
	var systemMessages []*pb.SystemMessage

	// 1. Fetch Battery
	var battery *pb.Battery
	if s.db != nil {
		bat, err := pb.GetBattery(ctx, s.db, batteryId, nil)
		if err != nil || bat == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("battery not found: %s", batteryId))
		}
		battery = bat
	}
	if battery != nil {
		if battery.CellCountS == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("battery is missing required cell count"))
		}
		baseWeight += battery.WeightG
	}

	// 2. Fetch Frame
	var frame *pb.Frame
	if s.db != nil {
		f, err := pb.GetFrame(ctx, s.db, b.FrameUuid, nil)
		if err != nil || f == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("frame not found: %s", b.FrameUuid))
		}
		frame = f
	}
	if frame != nil {
		baseWeight += frame.WeightG
	}

	// 3. Fetch Motor
	var motor *pb.Motor
	if s.db != nil {
		m, err := pb.GetMotor(ctx, s.db, b.MotorUuid, nil)
		if err != nil || m == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("motor not found: %s", b.MotorUuid))
		}
		motor = m
	}
	if motor != nil {
		baseWeight += (motor.WeightG * 4) // Quadcopter = 4 motors
	}

	// 4. Fetch Propeller
	var prop *pb.Propeller
	if s.db != nil {
		p, err := pb.GetPropeller(ctx, s.db, b.PropellerUuid, nil)
		if err != nil || p == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("propeller not found: %s", b.PropellerUuid))
		}
		prop = p
	}
	if prop != nil {
		baseWeight += (prop.WeightG * 4) // Quadcopter = 4 props
	}

	// 5. Fetch Flight Controller
	var fc *pb.FlightController
	if s.db != nil {
		f, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil)
		if err != nil || f == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("flight controller not found: %s", b.FlightControllerUuid))
		}
		fc = f
	}
	if fc != nil {
		baseWeight += fc.WeightG
	}

	// 6. Fetch Electronic Speed Controllers
	var totalEscs uint32 = 0
	var maxAmps float32 = 0
	var escs []*pb.ElectronicSpeedController
	if s.db != nil {
		for _, id := range b.ElectronicSpeedControllerUuids {
			esc, err := pb.GetElectronicSpeedController(ctx, s.db, id, nil)
			if err != nil || esc == nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("electronic speed controller not found: %s", id))
			}
			baseWeight += esc.WeightG
			totalEscs += esc.MaxMotors
			if esc.MotorCurrentMaxA > maxAmps {
				maxAmps = esc.MotorCurrentMaxA
			}
			escs = append(escs, esc)
		}

		if fc != nil && fc.GetInternalElectronicSpeedControllerUuid() != "" {
			esc, err := pb.GetElectronicSpeedController(ctx, s.db, fc.GetInternalElectronicSpeedControllerUuid(), nil)
			if err != nil || esc == nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("internal electronic speed controller not found: %s", fc.GetInternalElectronicSpeedControllerUuid()))
			}
			totalEscs += esc.MaxMotors
			if esc.MotorCurrentMaxA > maxAmps {
				maxAmps = esc.MotorCurrentMaxA
			}
			escs = append(escs, esc)
		}
	}

	if totalEscs < 4 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("build is missing required ESCs (quadcopter requires at least 4 ESC channels, found %d)", totalEscs))
	}

	// 7. Calculate build electrical limits & compatibility checks
	electrical := ComputeElectricalLimits(fc, escs, motor)
	if electrical.MinVoltage > 0 && electrical.MaxVoltage > 0 && electrical.MinVoltage > electrical.MaxVoltage {
		systemMessages = append(systemMessages, &pb.SystemMessage{
			Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_ERROR,
			Message:  fmt.Sprintf("Build electrical conflict: Minimum component voltage requirement (%.1fV) exceeds maximum component voltage rating (%.1fV)", electrical.MinVoltage, electrical.MaxVoltage),
		})
	}

	if battery != nil {
		// Voltage and current safety / compatibility checks
		if electrical.MaxVoltage > 0 && battery.MaxVoltage > electrical.MaxVoltage {
			systemMessages = append(systemMessages, &pb.SystemMessage{
				Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_ERROR,
				Message:  fmt.Sprintf("Battery max voltage (%.1fV) exceeds build component limit (%.1fV)", battery.MaxVoltage, electrical.MaxVoltage),
			})
		}
		if electrical.MinVoltage > 0 && battery.MinVoltage < electrical.MinVoltage {
			systemMessages = append(systemMessages, &pb.SystemMessage{
				Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
				Message:  fmt.Sprintf("Battery min voltage (%.1fV) is below build minimum operating voltage (%.1fV)", battery.MinVoltage, electrical.MinVoltage),
			})
		}
		if electrical.MaxCurrentA > 0 && battery.MaxCurrentA > 0 && battery.MaxCurrentA < electrical.MaxCurrentA {
			systemMessages = append(systemMessages, &pb.SystemMessage{
				Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
				Message:  fmt.Sprintf("Battery max current (%.1fA) is below build recommended current rating (%.1fA)", battery.MaxCurrentA, electrical.MaxCurrentA),
			})
		}
	}

	// 8. Fetch Video Transmitter (Standalone or Internal to FC)
	if b.VideoTransmitterUuid != "" && s.db != nil {
		videoTransmitter, err := pb.GetVideoTransmitter(ctx, s.db, b.VideoTransmitterUuid, nil)
		if err == nil && videoTransmitter != nil {
			baseWeight += videoTransmitter.WeightG
		}
	} else if fc != nil && fc.GetInternalVideoTransmitterUuid() != "" && s.db != nil {
		videoTransmitter, err := pb.GetVideoTransmitter(ctx, s.db, fc.GetInternalVideoTransmitterUuid(), nil)
		if err == nil && videoTransmitter != nil {
			baseWeight += videoTransmitter.WeightG
		}
	}

	// 9. Fetch Cameras
	if s.db != nil {
		for _, cid := range b.CameraUuids {
			cam, err := pb.GetCamera(ctx, s.db, cid, nil)
			if err == nil && cam != nil {
				baseWeight += cam.WeightG
			}
		}

		// 10. Fetch Receivers
		for _, rid := range b.ReceiverUuids {
			rx, err := pb.GetReceiver(ctx, s.db, rid, nil)
			if err == nil && rx != nil {
				baseWeight += rx.WeightG
			}
		}

		// 11. Fetch Antennas
		for _, aid := range b.AntennaUuids {
			ant, err := pb.GetAntenna(ctx, s.db, aid, nil)
			if err == nil && ant != nil {
				baseWeight += ant.WeightG
			}
		}

		// 12. Fetch GPS Receiver
		if b.GetGpsReceiverUuid() != "" {
			gps, err := pb.GetGpsReceiver(ctx, s.db, b.GetGpsReceiverUuid(), nil)
			if err == nil && gps != nil {
				baseWeight += gps.WeightG
			}
		}
	}

	payloadWeight := req.Msg.GetPayloadWeightG()
	if payloadWeight < 0 {
		payloadWeight = 0
	}
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
	systemMessages = append(systemMessages, phys.SystemMessages...)

	if b.Id != "" {
		buildId = b.Id
	} else if b.Uuid != "" {
		buildId = b.Uuid
	}

	evaluatedBatteryId := batteryId
	if battery != nil {
		if battery.Id != "" {
			evaluatedBatteryId = battery.Id
		} else if battery.Uuid != "" {
			evaluatedBatteryId = battery.Uuid
		}
	}

	res := &pb.EvaluateBuildResponse{
		BuildId:              buildId,
		PayloadWeightG:       payloadWeight,
		BatteryId:            evaluatedBatteryId,
		AllUpWeightG:         totalWeight,
		ThrustToWeightRatio:  phys.ThrustToWeightRatio,
		HoverThrottlePercent: phys.HoverThrottlePercent,
		HoverRpm:             phys.HoverRpm,
		MinFlightTimeMin:     phys.MinFlightTimeMin,
		MaxFlightTimeMin:     phys.MaxFlightTimeMin,
		MaxAccelerationMps2:  phys.MaxAccelerationMps2,
		TopSpeedKmh:          phys.TopSpeedKmh,
		SystemMessages:       systemMessages,
		MinVoltage:           electrical.MinVoltage,
		MaxVoltage:           electrical.MaxVoltage,
		MaxCurrentA:          electrical.MaxCurrentA,
	}

	return connect.NewResponse(res), nil
}

// PhysicsResult contains the calculated aerodynamic, electrical, and flight performance metrics.
type PhysicsResult struct {
	ThrustToWeightRatio  float32
	HoverThrottlePercent float32
	HoverRpm             uint32
	MinFlightTimeMin     float32
	MaxFlightTimeMin     float32
	MaxAccelerationMps2  float32
	TopSpeedKmh          float32
	SystemMessages       []*pb.SystemMessage
}

func (r PhysicsResult) Errors() []string {
	var errs []string
	for _, m := range r.SystemMessages {
		if m.Severity == pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_ERROR {
			errs = append(errs, m.Message)
		}
	}
	return errs
}

func (r PhysicsResult) Warnings() []string {
	var warns []string
	for _, m := range r.SystemMessages {
		if m.Severity == pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING {
			warns = append(warns, m.Message)
		}
	}
	return warns
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
			SystemMessages: []*pb.SystemMessage{
				{
					Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
					Message:  "Need a Motor, Propeller, and Battery to run physics estimation",
				},
			},
		}
	}

	var (
		thrustToWeight      float32
		hoverThrottle       float32
		hoverRpm            uint32
		minFlightTime       float32
		maxFlightTime       float32
		maxAccelerationMps2 float32
		topSpeedKmh         float32
		systemMessages      []*pb.SystemMessage
	)

	safeBase := float64(baseWeight)
	if safeBase <= 0 {
		safeBase = float64(baseWeight + payloadWeight)
	}
	totalWeight := baseWeight + payloadWeight

	if battery.CellCountS == 0 {
		return PhysicsResult{
			SystemMessages: []*pb.SystemMessage{
				{
					Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
					Message:  "Battery cell count is required to run physics estimation",
				},
			},
		}
	}

	cellCount := float32(battery.CellCountS)

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

	// Motor maximum burst mechanical shaft power (Watts):
	// Brushless multirotor outrunners produce ~25-30 W per gram of motor weight,
	// or ~0.30-0.35 W per mm^3 of stator volume.
	motorMaxPowerWatts := float64(statorVol) * 0.32
	if motor.WeightG > 0 {
		motorMaxPowerWatts = math.Min(motorMaxPowerWatts, float64(motor.WeightG)*30.0)
	}
	if motorMaxPowerWatts < 10.0 {
		motorMaxPowerWatts = 10.0
	}

	// 4. Propeller aerodynamic torque demand scale: D^4 * P * sqrt(blades / 2)
	propTorqueScale := math.Pow(float64(diaIn), 4) * float64(pitchIn) * math.Sqrt(float64(propBlades)/2.0)
	torqueRatio := float64(statorVol) / math.Max(1.0, propTorqueScale)

	// Aerodynamic power coefficient Cp:
	// Cp ≈ Ct * (P / D) * 1.15 (clamped to realistic minimum)
	cp := float64(ct) * float64(pOverD) * 1.15
	if cp < 0.04 {
		cp = 0.04
	}

	// Maximum RPM allowed by motor mechanical shaft power capacity:
	// P_aero = Cp * rho * n^3 * D^5  =>  n_power = (P_max / (Cp * rho * D^5))^(1/3)
	const rho = 1.225 // kg/m^3 standard sea-level air density
	dM := float64(propDiaMm) / 1000.0
	pFactor := cp * rho * math.Pow(dM, 5)
	var maxRpmByPower float64
	if pFactor > 0 {
		maxNByPower := math.Pow(motorMaxPowerWatts/pFactor, 1.0/3.0)
		maxRpmByPower = maxNByPower * 60.0
	}

	// Maximum RPM allowed by propeller tip speed compressibility limit:
	// Around Mach 0.70 (~240 m/s) at sea level, transonic compressibility drag divergence
	// causes drag to spike exponentially, capping physical prop speed.
	const maxTipSpeedMps = 240.0
	maxRpmByTip := (maxTipSpeedMps / (math.Pi * dM)) * 60.0

	// Full-throttle loaded RPM under static bollard condition (J = 0):
	// Aerodynamic torque limits motor to ~72% of no-load (Kv * V)
	rpmLoadFactor := 0.72 * math.Min(1.05, math.Max(0.55, math.Pow(torqueRatio/1.0, 0.15)))
	loadedRpm := float32(float64(float32(motor.Kv)*voltage) * rpmLoadFactor)

	// Cap loaded RPM by motor shaft power and propeller tip speed
	if maxRpmByPower > 0 && float64(loadedRpm) > maxRpmByPower {
		loadedRpm = float32(maxRpmByPower)
	}
	if float64(loadedRpm) > maxRpmByTip {
		loadedRpm = float32(maxRpmByTip)
	}

	// 5. Static thrust in open air (momentum / blade element theory):
	// T = Ct * rho * n^2 * D^4
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
		systemMessages = append(systemMessages, &pb.SystemMessage{
			Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_ERROR,
			Message:  "Drone is too heavy to take off (Hover throttle > 100%)",
		})
	} else if hoverThrottle > 50.0 {
		systemMessages = append(systemMessages, &pb.SystemMessage{
			Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
			Message:  "Drone will be very sluggish (Hover throttle > 50%)",
		})
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
		systemMessages = append(systemMessages, &pb.SystemMessage{
			Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
			Message:  "Cruise amps exceeds ESC continuous rating",
		})
	}

	if totalCruiseAmps > 0 && battery.CapacityMah > 0 {
		usableAh := (float32(battery.CapacityMah) / 1000.0) * 0.80 // 80% usable capacity (20% safety margin)
		maxFlightTime = (usableAh / totalCruiseAmps) * 60.0        // minutes (smooth cruise / cinematic)

		// Aggressive freestyle / acro / sustained punchouts:
		// High throttle bursts, dynamic braking, and PID stabilization draw ~1.9x cruise power
		aggressiveWatts := cruiseWatts * 1.9
		aggressiveAmps := aggressiveWatts / nominalVoltage
		minFlightTime = (usableAh / aggressiveAmps) * 60.0 // minutes (aggressive freestyle)
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
	if totalThrust > 0 && propPitchMm > 0 && thrustToWeight >= 1.0 {
		loadedRpmFwd := float64(motor.Kv) * float64(voltage) * 0.86
		if maxRpmByPower > 0 && loadedRpmFwd > maxRpmByPower*1.15 {
			loadedRpmFwd = maxRpmByPower * 1.15
		}
		if loadedRpmFwd > maxRpmByTip {
			loadedRpmFwd = maxRpmByTip
		}
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

	// 11. Electrical and thermal safety checks:
	// Burst mechanical power at full throttle:
	nBurst := float64(loadedRpm) / 60.0
	pMechBurst := cp * rho * (nBurst * nBurst * nBurst) * math.Pow(dM, 5)
	pElecBurst := pMechBurst / 0.75 // ~75% motor efficiency at burst
	burstAmpsPerMotor := pElecBurst / float64(voltage)

	if maxEscAmps > 0 && burstAmpsPerMotor > float64(maxEscAmps)*1.25 {
		systemMessages = append(systemMessages, &pb.SystemMessage{
			Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
			Message:  fmt.Sprintf("Full-throttle current (%.1fA/motor) exceeds ESC burst rating (%.1fA)", burstAmpsPerMotor, float64(maxEscAmps)*1.25),
		})
	}

	if thrustToWeight >= 1.0 && hoverRpm > 0 {
		nHover := float64(hoverRpm) / 60.0
		pMechHover := cp * rho * (nHover * nHover * nHover) * math.Pow(dM, 5)
		pElecHover := pMechHover / 0.80 // ~80% motor efficiency at hover
		hoverAmpsPerMotor := pElecHover / float64(nominalVoltage)

		if maxEscAmps > 0 && hoverAmpsPerMotor > float64(maxEscAmps) {
			systemMessages = append(systemMessages, &pb.SystemMessage{
				Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
				Message:  fmt.Sprintf("Hover current (%.1fA/motor) exceeds ESC continuous rating (%.1fA)", hoverAmpsPerMotor, maxEscAmps),
			})
		}

		if motor.WeightG > 0 && pElecHover > float64(motor.WeightG)*20.0 {
			systemMessages = append(systemMessages, &pb.SystemMessage{
				Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
				Message:  fmt.Sprintf("Hover power (%.1fW/motor) exceeds motor thermal dissipation limit (%.1fW)", pElecHover, float64(motor.WeightG)*20.0),
			})
		}
	}

	return PhysicsResult{
		ThrustToWeightRatio:  thrustToWeight,
		HoverThrottlePercent: hoverThrottle,
		HoverRpm:             hoverRpm,
		MinFlightTimeMin:     minFlightTime,
		MaxFlightTimeMin:     maxFlightTime,
		MaxAccelerationMps2:  maxAccelerationMps2,
		TopSpeedKmh:          topSpeedKmh,
		SystemMessages:       systemMessages,
	}
}

// GetBuildElectricalLimits calculates electrical limits (min/max voltage and max current)
// and returns the lightest compatible battery.
func (s *EvaluatorServiceHandler) GetBuildElectricalLimits(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
	buildId := strings.TrimSpace(req.Msg.GetBuildId())
	if buildId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("build_id is required"))
	}
	var b *pb.Build
	if s.db != nil {
		fetched, err := pb.GetBuild(ctx, s.db, buildId, nil)
		if err != nil || fetched == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %s", buildId))
		}
		b = fetched
	}
	if b == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build not found: %s", buildId))
	}

	limits, err := s.CalculateElectricalLimits(ctx, b)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if b.Id != "" {
		limits.BuildId = b.Id
	} else if b.Uuid != "" {
		limits.BuildId = b.Uuid
	} else {
		limits.BuildId = buildId
	}

	return connect.NewResponse(limits), nil
}

// CalculateElectricalLimits queries referenced components and computes electrical constraints.
func (s *EvaluatorServiceHandler) CalculateElectricalLimits(ctx context.Context, b *pb.Build) (*pb.GetBuildElectricalLimitsResponse, error) {
	var fc *pb.FlightController
	var escs []*pb.ElectronicSpeedController
	var motor *pb.Motor

	if b.FlightControllerUuid != "" && s.db != nil {
		f, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil)
		if err == nil {
			fc = f
			if fc.GetInternalElectronicSpeedControllerUuid() != "" {
				esc, err := pb.GetElectronicSpeedController(ctx, s.db, fc.GetInternalElectronicSpeedControllerUuid(), nil)
				if err == nil {
					escs = append(escs, esc)
				}
			}
		}
	}

	if s.db != nil {
		for _, escUuid := range b.ElectronicSpeedControllerUuids {
			esc, err := pb.GetElectronicSpeedController(ctx, s.db, escUuid, nil)
			if err == nil {
				escs = append(escs, esc)
			}
		}
	}

	if b.MotorUuid != "" && s.db != nil {
		m, err := pb.GetMotor(ctx, s.db, b.MotorUuid, nil)
		if err == nil {
			motor = m
		}
	}

	limits := ComputeElectricalLimits(fc, escs, motor)
	limits.DefaultBatteryId = s.FindLightestCompatibleBattery(ctx, limits.MinVoltage, limits.MaxVoltage, limits.MaxCurrentA)

	return limits, nil
}

// ComputeElectricalLimits determines min_voltage, max_voltage, and max_current_a from hardware specs.
func ComputeElectricalLimits(fc *pb.FlightController, escs []*pb.ElectronicSpeedController, motor *pb.Motor) *pb.GetBuildElectricalLimitsResponse {
	var minV float32 = 0
	var maxV float32 = 0

	if fc != nil && fc.MinVoltage > minV {
		minV = fc.MinVoltage
	}
	for _, esc := range escs {
		if esc.MinVoltage > minV {
			minV = esc.MinVoltage
		}
	}
	if motor != nil && motor.MinVoltage > minV {
		minV = motor.MinVoltage
	}

	updateMaxV := func(v float32) {
		if v > 0 {
			if maxV == 0 || v < maxV {
				maxV = v
			}
		}
	}

	if fc != nil {
		updateMaxV(fc.MaxVoltage)
	}
	for _, esc := range escs {
		updateMaxV(esc.MaxVoltage)
	}
	if motor != nil {
		updateMaxV(motor.MaxVoltage)
	}

	// Calculate max_current_a required by the build
	var maxCurrentA float32 = 0
	if motor != nil && motor.MaxCurrentA > 0 {
		maxCurrentA = motor.MaxCurrentA * 4.0
	} else if len(escs) > 0 {
		var escTotal float32 = 0
		for _, esc := range escs {
			if esc.MotorCurrentMaxA > 0 {
				multiplier := float32(esc.MaxMotors)
				if multiplier == 0 {
					multiplier = 1.0
				}
				escTotal += esc.MotorCurrentMaxA * multiplier
			}
		}
		maxCurrentA = escTotal
	}

	return &pb.GetBuildElectricalLimitsResponse{
		MinVoltage:  float32(math.Round(float64(minV)*100) / 100),
		MaxVoltage:  float32(math.Round(float64(maxV)*100) / 100),
		MaxCurrentA: float32(math.Round(float64(maxCurrentA)*10) / 10),
	}
}

// FindLightestCompatibleBattery finds the lightest compatible battery in the database.
func (s *EvaluatorServiceHandler) FindLightestCompatibleBattery(ctx context.Context, minV, maxV, maxA float32) string {
	if s.db == nil || (minV > 0 && maxV > 0 && minV > maxV) {
		return ""
	}

	var batteryId string
	// Strict match: min_voltage >= minV, max_voltage <= maxV, max_current_a >= maxA
	err := s.db.QueryRow(ctx, `
		SELECT id FROM batteries
		WHERE min_voltage >= $1 AND max_voltage <= $2 AND max_current_a >= $3
		ORDER BY weight_g ASC, id ASC
		LIMIT 1
	`, minV, maxV, maxA).Scan(&batteryId)
	if err != nil {
		return ""
	}

	return batteryId
}

// ValidateBuildComponents checks that a build has all required component IDs.
func ValidateBuildComponents(b *pb.Build) error {
	if b == nil {
		return fmt.Errorf("build is required")
	}
	if strings.TrimSpace(b.FrameUuid) == "" {
		return fmt.Errorf("build is missing required frame")
	}
	if strings.TrimSpace(b.MotorUuid) == "" {
		return fmt.Errorf("build is missing required motor")
	}
	if strings.TrimSpace(b.PropellerUuid) == "" {
		return fmt.Errorf("build is missing required propeller")
	}
	if strings.TrimSpace(b.FlightControllerUuid) == "" {
		return fmt.Errorf("build is missing required flight controller")
	}
	return nil
}
