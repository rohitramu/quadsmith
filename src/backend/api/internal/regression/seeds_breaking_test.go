package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	pb "quadsmith/api/gen/quadsmith"
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
		unmarshal func(data []byte) (count int, ids []string, uuids []string, err error)
	}{
		{
			filename: "motors.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListMotorsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Motors))
				uuids := make([]string, len(res.Motors))
				for i, m := range res.Motors {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Motors), ids, uuids, nil
			},
		},
		{
			filename: "frames.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListFramesResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Frames))
				uuids := make([]string, len(res.Frames))
				for i, m := range res.Frames {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Frames), ids, uuids, nil
			},
		},
		{
			filename: "batteries.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListBatteriesResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Batteries))
				uuids := make([]string, len(res.Batteries))
				for i, m := range res.Batteries {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Batteries), ids, uuids, nil
			},
		},
		{
			filename: "electronic_speed_controllers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListElectronicSpeedControllersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.ElectronicSpeedControllers))
				uuids := make([]string, len(res.ElectronicSpeedControllers))
				for i, m := range res.ElectronicSpeedControllers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.ElectronicSpeedControllers), ids, uuids, nil
			},
		},
		{
			filename: "antennas.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListAntennasResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Antennas))
				uuids := make([]string, len(res.Antennas))
				for i, m := range res.Antennas {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Antennas), ids, uuids, nil
			},
		},
		{
			filename: "cameras.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListCamerasResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Cameras))
				uuids := make([]string, len(res.Cameras))
				for i, m := range res.Cameras {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Cameras), ids, uuids, nil
			},
		},
		{
			filename: "propellers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListPropellersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Propellers))
				uuids := make([]string, len(res.Propellers))
				for i, m := range res.Propellers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Propellers), ids, uuids, nil
			},
		},
		{
			filename: "receivers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListReceiversResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Receivers))
				uuids := make([]string, len(res.Receivers))
				for i, m := range res.Receivers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Receivers), ids, uuids, nil
			},
		},
		{
			filename: "video_transmitters.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListVideoTransmittersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.VideoTransmitters))
				uuids := make([]string, len(res.VideoTransmitters))
				for i, m := range res.VideoTransmitters {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.VideoTransmitters), ids, uuids, nil
			},
		},
		{
			filename: "flight_controllers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListFlightControllersResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.FlightControllers))
				uuids := make([]string, len(res.FlightControllers))
				for i, m := range res.FlightControllers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.FlightControllers), ids, uuids, nil
			},
		},
		{
			filename: "gps_receivers.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListGpsReceiversResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.GpsReceivers))
				uuids := make([]string, len(res.GpsReceivers))
				for i, m := range res.GpsReceivers {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.GpsReceivers), ids, uuids, nil
			},
		},
		{
			filename: "builds.textproto",
			unmarshal: func(data []byte) (int, []string, []string, error) {
				res := &pb.ListBuildsResponse{}
				if err := prototext.Unmarshal(data, res); err != nil {
					return 0, nil, nil, err
				}
				ids := make([]string, len(res.Builds))
				uuids := make([]string, len(res.Builds))
				for i, m := range res.Builds {
					ids[i] = m.Id
					uuids[i] = m.Uuid
				}
				return len(res.Builds), ids, uuids, nil
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

			count, ids, uuids, err := tt.unmarshal(data)
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
			}
		})
	}
}
