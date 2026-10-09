package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func getAPIURL() string {
	if url := os.Getenv("QS_API_URL"); url != "" {
		return url
	}
	return "http://127.0.0.1:8080"
}

func TestSandboxBuilds(t *testing.T) {
	apiUrl := getAPIURL()
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.ListBuildsRequest{})

	res, err := buildClient.ListBuilds(ctx, req)
	if err != nil {
		t.Fatalf("Failed to list builds: %v", err)
	}

	if res.Msg.Builds == nil {
		t.Log("Got nil builds array (no builds exist yet)")
	} else {
		t.Logf("Got %d builds", len(res.Msg.Builds))
	}
}

func TestSandboxFrames(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	frameClient := quadsmithconnect.NewFrameServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.ListFramesRequest{})

	res, err := frameClient.ListFrames(ctx, req)
	if err != nil {
		t.Fatalf("Failed to list frames: %v", err)
	}

	if len(res.Msg.Frames) == 0 {
		t.Log("No frames found")
	} else {
		t.Logf("Found %d frames, e.g. %s", len(res.Msg.Frames), res.Msg.Frames[0].Name)
	}
}

func TestSandboxEvaluateBuild(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	evalClient := quadsmithconnect.NewEvaluatorServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. Missing battery_id should return InvalidArgument
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
	}))
	if err == nil {
		t.Fatal("expected error when battery_id is missing, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "battery_id is required") {
		t.Errorf("expected 'battery_id is required' error, got: %v", err)
	}

	// 2. Non-existent battery should return NotFound
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:   "non-existent-battery",
	}))
	if err == nil {
		t.Fatal("expected error when battery does not exist, got nil")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 3. Missing build_id should return InvalidArgument
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err == nil {
		t.Fatal("expected error when build_id is missing, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "either build or build_id must be provided") && !strings.Contains(err.Error(), "build_id is required") {
		t.Errorf("expected build source error, got: %v", err)
	}

	// 4. Non-existent build should return NotFound
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "non-existent-build"},
		BatteryId:   "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err == nil {
		t.Fatal("expected error when build does not exist, got nil")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 5. Valid build with battery evaluates successfully
	res, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:   "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild failed: %v", err)
	}
	if res.Msg.BuildId != "ultralight-toothpick" {
		t.Errorf("expected build_id == 'ultralight-toothpick', got: %v", res.Msg.BuildId)
	}
	if res.Msg.BatteryId != "betafpv-lava-1s-300mah-75c-lihv" {
		t.Errorf("expected battery_id == 'betafpv-lava-1s-300mah-75c-lihv', got: %v", res.Msg.BatteryId)
	}
	if res.Msg.AllUpWeightG <= 0 {
		t.Errorf("expected positive all_up_weight_g, got: %v", res.Msg.AllUpWeightG)
	}
	if res.Msg.ThrustToWeightRatio <= 0 {
		t.Errorf("expected positive thrust_to_weight_ratio, got: %v", res.Msg.ThrustToWeightRatio)
	}

	// 6. Evaluate with payload and verify all-up weight includes the payload
	resWithPayload, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource:    &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:      "betafpv-lava-1s-300mah-75c-lihv",
		PayloadWeightG: 25.0,
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild with payload failed: %v", err)
	}
	diff := resWithPayload.Msg.AllUpWeightG - res.Msg.AllUpWeightG
	if diff < 24.9 || diff > 25.1 {
		t.Errorf("expected all-up weight to increase by 25g with payload, got diff=%.2f (before=%.2f, after=%.2f)", diff, res.Msg.AllUpWeightG, resWithPayload.Msg.AllUpWeightG)
	}

	// 7. Evaluate in-memory draft Build object
	resDraft, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_Build{
			Build: &pb.Build{
				FrameUuid:     "01923019-3008-7001-8001-000000000001",
				MotorUuid:     "01923019-3001-7001-8001-000000000001",
				PropellerUuid: "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			},
		},
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild with in-memory draft build failed: %v", err)
	}
	if resDraft.Msg.AllUpWeightG <= 0 {
		t.Errorf("expected positive all-up weight for in-memory draft, got: %v", resDraft.Msg.AllUpWeightG)
	}
}
