package compatibility

import (
	"context"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith"
)

type CompatibilityServiceHandler struct {
	db *pgxpool.Pool
}

func NewCompatibilityServiceHandler(db *pgxpool.Pool) *CompatibilityServiceHandler {
	return &CompatibilityServiceHandler{db: db}
}

func (s *CompatibilityServiceHandler) CheckCompatibility(ctx context.Context, req *connect.Request[pb.CheckCompatibilityRequest]) (*connect.Response[pb.CheckCompatibilityResponse], error) {
	b := req.Msg.GetBuild()
	if b == nil {
		return connect.NewResponse(&pb.CheckCompatibilityResponse{}), nil
	}

	comps := &Components{}
	if s.db != nil {
		if b.FrameUuid != "" {
			if f, err := pb.GetFrame(ctx, s.db, b.FrameUuid, nil); err == nil {
				comps.Frame = f
			}
		}
		if b.MotorUuid != "" {
			if m, err := pb.GetMotor(ctx, s.db, b.MotorUuid, nil); err == nil {
				comps.Motor = m
			}
		}
		if b.PropellerUuid != "" {
			if p, err := pb.GetPropeller(ctx, s.db, b.PropellerUuid, nil); err == nil {
				comps.Propeller = p
			}
		}
		if b.FlightControllerUuid != "" {
			if fc, err := pb.GetFlightController(ctx, s.db, b.FlightControllerUuid, nil); err == nil {
				comps.FlightController = fc
				if fc.GetInternalElectronicSpeedControllerUuid() != "" {
					if esc, err := pb.GetElectronicSpeedController(ctx, s.db, fc.GetInternalElectronicSpeedControllerUuid(), nil); err == nil {
						comps.ElectronicSpeedControllers = append(comps.ElectronicSpeedControllers, esc)
					}
				}
			}
		}
		for _, escUuid := range b.ElectronicSpeedControllerUuids {
			if esc, err := pb.GetElectronicSpeedController(ctx, s.db, escUuid, nil); err == nil {
				comps.ElectronicSpeedControllers = append(comps.ElectronicSpeedControllers, esc)
			}
		}
		if b.VideoTransmitterUuid != "" {
			if vtx, err := pb.GetVideoTransmitter(ctx, s.db, b.VideoTransmitterUuid, nil); err == nil {
				comps.VideoTransmitter = vtx
			}
		}
		for _, cid := range b.CameraUuids {
			if cam, err := pb.GetCamera(ctx, s.db, cid, nil); err == nil {
				comps.Cameras = append(comps.Cameras, cam)
			}
		}
		for _, rid := range b.ReceiverUuids {
			if rx, err := pb.GetReceiver(ctx, s.db, rid, nil); err == nil {
				comps.Receivers = append(comps.Receivers, rx)
			}
		}
		for _, aid := range b.AntennaUuids {
			if ant, err := pb.GetAntenna(ctx, s.db, aid, nil); err == nil {
				comps.Antennas = append(comps.Antennas, ant)
			}
		}
	}

	messages := CheckCompatibility(comps)
	return connect.NewResponse(&pb.CheckCompatibilityResponse{
		Messages: messages,
	}), nil
}
