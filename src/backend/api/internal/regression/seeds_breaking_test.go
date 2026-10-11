package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/internal/validation"
	"quadsmith/api/pkg/schemacheck"
)

func TestSeeds_UnmarshalWithoutErrors(t *testing.T) {
	gh, err := schemacheck.NewGitHelper()
	if err != nil {
		t.Fatalf("Failed to initialize GitHelper: %v", err)
	}

	seedsDir := filepath.Join(gh.RepoRoot, "src/backend/db/seeds")

	tests := []struct {
		filename  string
		unmarshal func(data []byte) (count int, ids []string, uuids []string, refLinks [][]*pb.ReferenceLink, err error)
	}{
		{
			filename: "motors.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListMotorsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Motors))
				uuids := make([]string, len(res.Motors))
				refLinks := make([][]*pb.ReferenceLink, len(res.Motors))
				for i, m := range res.Motors {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Motors), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "frames.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListFramesResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Frames))
				uuids := make([]string, len(res.Frames))
				refLinks := make([][]*pb.ReferenceLink, len(res.Frames))
				for i, m := range res.Frames {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Frames), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "batteries.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListBatteriesResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Batteries))
				uuids := make([]string, len(res.Batteries))
				refLinks := make([][]*pb.ReferenceLink, len(res.Batteries))
				for i, m := range res.Batteries {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Batteries), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "electronic_speed_controllers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListElectronicSpeedControllersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.ElectronicSpeedControllers))
				uuids := make([]string, len(res.ElectronicSpeedControllers))
				refLinks := make([][]*pb.ReferenceLink, len(res.ElectronicSpeedControllers))
				for i, m := range res.ElectronicSpeedControllers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.ElectronicSpeedControllers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "antennas.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListAntennasResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Antennas))
				uuids := make([]string, len(res.Antennas))
				refLinks := make([][]*pb.ReferenceLink, len(res.Antennas))
				for i, m := range res.Antennas {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Antennas), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "cameras.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListCamerasResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Cameras))
				uuids := make([]string, len(res.Cameras))
				refLinks := make([][]*pb.ReferenceLink, len(res.Cameras))
				for i, m := range res.Cameras {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Cameras), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "propellers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListPropellersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Propellers))
				uuids := make([]string, len(res.Propellers))
				refLinks := make([][]*pb.ReferenceLink, len(res.Propellers))
				for i, m := range res.Propellers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Propellers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "receivers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListReceiversResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Receivers))
				uuids := make([]string, len(res.Receivers))
				refLinks := make([][]*pb.ReferenceLink, len(res.Receivers))
				for i, m := range res.Receivers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Receivers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "video_transmitters.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListVideoTransmittersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.VideoTransmitters))
				uuids := make([]string, len(res.VideoTransmitters))
				refLinks := make([][]*pb.ReferenceLink, len(res.VideoTransmitters))
				for i, m := range res.VideoTransmitters {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.VideoTransmitters), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "flight_controllers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListFlightControllersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.FlightControllers))
				uuids := make([]string, len(res.FlightControllers))
				refLinks := make([][]*pb.ReferenceLink, len(res.FlightControllers))
				for i, m := range res.FlightControllers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.FlightControllers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "gps_receivers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListGpsReceiversResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.GpsReceivers))
				uuids := make([]string, len(res.GpsReceivers))
				refLinks := make([][]*pb.ReferenceLink, len(res.GpsReceivers))
				for i, m := range res.GpsReceivers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.GpsReceivers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "builds.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListBuildsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Builds))
				uuids := make([]string, len(res.Builds))
				refLinks := make([][]*pb.ReferenceLink, len(res.Builds))
				for i, m := range res.Builds {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Builds), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "fc_firmwares.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListFcFirmwaresResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.FcFirmwares))
				uuids := make([]string, len(res.FcFirmwares))
				refLinks := make([][]*pb.ReferenceLink, len(res.FcFirmwares))
				for i, m := range res.FcFirmwares {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.FcFirmwares), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "esc_firmwares.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListEscFirmwaresResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.EscFirmwares))
				uuids := make([]string, len(res.EscFirmwares))
				refLinks := make([][]*pb.ReferenceLink, len(res.EscFirmwares))
				for i, m := range res.EscFirmwares {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.EscFirmwares), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "vtx_firmwares.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListVtxFirmwaresResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.VtxFirmwares))
				uuids := make([]string, len(res.VtxFirmwares))
				refLinks := make([][]*pb.ReferenceLink, len(res.VtxFirmwares))
				for i, m := range res.VtxFirmwares {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.VtxFirmwares), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "radio_firmwares.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListRadioFirmwaresResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.RadioFirmwares))
				uuids := make([]string, len(res.RadioFirmwares))
				refLinks := make([][]*pb.ReferenceLink, len(res.RadioFirmwares))
				for i, m := range res.RadioFirmwares {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.RadioFirmwares), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "software_tools.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListSoftwareToolsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.SoftwareTools))
				uuids := make([]string, len(res.SoftwareTools))
				refLinks := make([][]*pb.ReferenceLink, len(res.SoftwareTools))
				for i, m := range res.SoftwareTools {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.SoftwareTools), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "transmitters.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListTransmittersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Transmitters))
				uuids := make([]string, len(res.Transmitters))
				refLinks := make([][]*pb.ReferenceLink, len(res.Transmitters))
				for i, m := range res.Transmitters {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Transmitters), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "goggles.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListGogglesResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Goggles))
				uuids := make([]string, len(res.Goggles))
				refLinks := make([][]*pb.ReferenceLink, len(res.Goggles))
				for i, m := range res.Goggles {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Goggles), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "chargers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListChargersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.Chargers))
				uuids := make([]string, len(res.Chargers))
				refLinks := make([][]*pb.ReferenceLink, len(res.Chargers))
				for i, m := range res.Chargers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.Chargers), ids, uuids, refLinks, nil
			},
		},
		{
			filename: "field_tools.textproto",
			unmarshal: func(data []byte) (int, []string, []string, [][]*pb.ReferenceLink, error) {
				res := &pb.ListFieldToolsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, nil, err
				}
				ids := make([]string, len(res.FieldTools))
				uuids := make([]string, len(res.FieldTools))
				refLinks := make([][]*pb.ReferenceLink, len(res.FieldTools))
				for i, m := range res.FieldTools {
					ids[i] = m.Id
					uuids[i] = m.Uuid
					refLinks[i] = m.ReferenceLinks
				}
				return len(res.FieldTools), ids, uuids, refLinks, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join(seedsDir, tt.filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read seed file %s: %v", tt.filename, err)
			}

			count, ids, uuids, refLinks, err := tt.unmarshal(data)
			if err != nil {
				reportBreakingChange(t, gh, "Seed Data", "current schema", fmt.Sprintf("FAILED TO UNMARSHAL %s: %v\nExisting seed data does not conform to the Protobuf definitions.", tt.filename, err))
				return
			}

			t.Logf("%s: successfully parsed %d records", tt.filename, count)

			// Validate UUID and ID uniqueness and non-emptiness
			seenIDs := make(map[string]struct{})
			seenUUIDs := make(map[string]struct{})

			for i := 0; i < count; i++ {
				id := ids[i]
				uuid := uuids[i]

				if id == "" {
					t.Errorf("Record %d in %s has empty 'id'", i, tt.filename)
				}
				if uuid == "" {
					t.Errorf("Record %d (%s) in %s has empty 'uuid'", i, id, tt.filename)
				}

				if _, exists := seenIDs[id]; exists {
					t.Errorf("Duplicate 'id' %q found in %s", id, tt.filename)
				}
				seenIDs[id] = struct{}{}

				if _, exists := seenUUIDs[uuid]; exists {
					t.Errorf("Duplicate 'uuid' %q found in %s", uuid, tt.filename)
				}
				seenUUIDs[uuid] = struct{}{}

				if err := validation.ValidateReferenceLinks(refLinks[i]); err != nil {
					t.Errorf("Record %d (%s) in %s has invalid reference links: %v", i, id, tt.filename, err)
				}
			}
		})
	}
}
