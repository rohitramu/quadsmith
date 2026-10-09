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
		BuildId: "ultralight-toothpick",
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
		BuildId:   "ultralight-toothpick",
		BatteryId: "non-existent-battery",
	}))
	if err == nil {
		t.Fatal("expected error when battery does not exist, got nil")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 3. Build missing required component (e.g. motor) should return InvalidArgument
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		Build: &pb.Build{
			FrameUuid:            "0e04d336-1756-4612-88a9-a617635b59d4",
			PropellerUuid:        "01923019-3002-7001-8001-000000000007",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000004",
		},
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err == nil {
		t.Fatal("expected error when build is missing motor, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "build is missing required motor") {
		t.Errorf("expected 'build is missing required motor' error, got: %v", err)
	}

	// 4. Valid build with battery evaluates successfully
	res, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildId:   "ultralight-toothpick",
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
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
	if res.Msg.TotalWeightG <= 0 {
		t.Errorf("expected positive total_weight_g, got: %v", res.Msg.TotalWeightG)
	}
	if res.Msg.ThrustToWeightRatio <= 0 {
		t.Errorf("expected positive thrust_to_weight_ratio, got: %v", res.Msg.ThrustToWeightRatio)
	}
}
