package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func TestSandboxBuilds(t *testing.T) {
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Sandbox not running at 127.0.0.1:8080: %v", err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		"http://127.0.0.1:8080",
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
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Sandbox not running at 127.0.0.1:8080: %v", err)
	}

	frameClient := quadsmithconnect.NewFrameServiceClient(
		http.DefaultClient,
		"http://127.0.0.1:8080",
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
