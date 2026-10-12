package quadsmith

import (
	"testing"
)

func TestScoreSimilarity_ExactMatch(t *testing.T) {
	target := &Build{
		FrameUuid:            "frame-1",
		VideoTransmitterUuid: "vtx-1",
		CameraUuids:          []string{"cam-1"},
		MotorUuid:            "motor-1",
	}
	targetFrame := &Frame{Uuid: "frame-1", Name: "Source One V5", MaxPropSizeMm: 130, Geometry: "True X"}
	targetVtx := &VideoTransmitter{Uuid: "vtx-1", Name: "TBS Unify", Protocol: "Analog"}
	targetCam := &Camera{Uuid: "cam-1", Name: "Ratel 2", Protocol: "Analog", WidthMm: 19}
	targetMotor := &Motor{Uuid: "motor-1", Name: "F60 PRO V", StatorDiameterMm: 22, StatorHeightMm: 7.5, Kv: 2020}

	cand := &Build{
		FrameUuid:            "frame-1",
		VideoTransmitterUuid: "vtx-1",
		CameraUuids:          []string{"cam-1"},
		MotorUuid:            "motor-1",
	}

	score, details := ScoreSimilarity(
		target, targetFrame, targetVtx, targetCam, targetMotor,
		cand, targetFrame, targetVtx, targetCam, targetMotor,
	)

	if score != 100.0 {
		t.Fatalf("expected score 100.0 for identical build, got %f", score)
	}

	if len(details) != 4 {
		t.Fatalf("expected 4 match details, got %d", len(details))
	}

	for _, d := range details {
		if !d.ExactMatch {
			t.Errorf("expected exact match for component %s, got false", d.Component)
		}
	}
}

func TestScoreSimilarity_TieredPartialMatches(t *testing.T) {
	target := &Build{
		FrameUuid:            "frame-target",
		VideoTransmitterUuid: "vtx-target",
		CameraUuids:          []string{"cam-target"},
		MotorUuid:            "motor-target",
	}
	targetFrame := &Frame{Uuid: "frame-target", Name: "Target Frame", MaxPropSizeMm: 130, Geometry: "True X"}
	targetVtx := &VideoTransmitter{Uuid: "vtx-target", Name: "Target VTX", Protocol: "Walksnail"}
	targetCam := &Camera{Uuid: "cam-target", Name: "Target Cam", Protocol: "Walksnail", WidthMm: 19}
	targetMotor := &Motor{Uuid: "motor-target", Name: "Target Motor", StatorDiameterMm: 22, StatorHeightMm: 7, Kv: 1950}

	cand := &Build{
		FrameUuid:            "frame-cand",
		VideoTransmitterUuid: "vtx-cand",
		CameraUuids:          []string{"cam-cand"},
		MotorUuid:            "motor-cand",
	}
	candFrame := &Frame{Uuid: "frame-cand", Name: "Cand Frame", MaxPropSizeMm: 132, Geometry: "true x"}
	candVtx := &VideoTransmitter{Uuid: "vtx-cand", Name: "Cand VTX", Protocol: "walksnail"}
	candCam := &Camera{Uuid: "cam-cand", Name: "Cand Cam", Protocol: "walksnail", WidthMm: 14}
	candMotor := &Motor{Uuid: "motor-cand", Name: "Cand Motor", StatorDiameterMm: 22, StatorHeightMm: 7, Kv: 2020}

	score, details := ScoreSimilarity(
		target, targetFrame, targetVtx, targetCam, targetMotor,
		cand, candFrame, candVtx, candCam, candMotor,
	)

	expectedScore := 55.0 // 25 + 15 + 9 + 6
	if score != expectedScore {
		t.Fatalf("expected score %f, got %f", expectedScore, score)
	}

	if len(details) != 4 {
		t.Fatalf("expected 4 match details, got %d", len(details))
	}

	for _, d := range details {
		if d.ExactMatch {
			t.Errorf("expected partial match for component %s, got exact", d.Component)
		}
	}
}

func TestScoreSimilarity_ThresholdFiltering(t *testing.T) {
	target := &Build{
		FrameUuid:            "frame-5inch",
		VideoTransmitterUuid: "vtx-dji",
		CameraUuids:          []string{"cam-dji"},
		MotorUuid:            "motor-2207",
	}
	targetFrame := &Frame{Uuid: "frame-5inch", MaxPropSizeMm: 130, Geometry: "True X"}
	targetVtx := &VideoTransmitter{Uuid: "vtx-dji", Protocol: "DJI O3"}
	targetCam := &Camera{Uuid: "cam-dji", Protocol: "DJI O3", WidthMm: 20}
	targetMotor := &Motor{Uuid: "motor-2207", StatorDiameterMm: 22, StatorHeightMm: 7, Kv: 1950}

	cand := &Build{
		FrameUuid:            "frame-whoop",
		VideoTransmitterUuid: "vtx-analog",
		CameraUuids:          []string{"cam-analog"},
		MotorUuid:            "motor-1103",
	}
	candFrame := &Frame{Uuid: "frame-whoop", MaxPropSizeMm: 52, Geometry: "Whoop"}
	candVtx := &VideoTransmitter{Uuid: "vtx-analog", Protocol: "Analog"}
	candCam := &Camera{Uuid: "cam-analog", Protocol: "Analog", WidthMm: 14}
	candMotor := &Motor{Uuid: "motor-1103", StatorDiameterMm: 11, StatorHeightMm: 3, Kv: 10000}

	score, details := ScoreSimilarity(
		target, targetFrame, targetVtx, targetCam, targetMotor,
		cand, candFrame, candVtx, candCam, candMotor,
	)

	if score >= 25.0 {
		t.Fatalf("expected score below 25.0 threshold, got %f", score)
	}

	if len(details) != 0 {
		t.Fatalf("expected 0 match details for completely distinct build, got %d", len(details))
	}
}
