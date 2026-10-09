package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resolveQSPath(t *testing.T) string {
	t.Helper()
	// Try relative from test/api/
	qsPath, err := filepath.Abs("../../bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	// Try relative from test/
	qsPath, err = filepath.Abs("../bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	// Try relative from repo root
	qsPath, err = filepath.Abs("bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	t.Fatalf("Failed to resolve qs binary path (checked ../../bin/qs, ../bin/qs, and bin/qs)")
	return ""
}

func TestCLI_ListFrames(t *testing.T) {
	apiUrl := getAPIURL()
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	qsPath := resolveQSPath(t)

	cmd := exec.Command(qsPath, "components", "hardware", "frames", "list", "--json")
	cmd.Env = append(cmd.Env, "QS_API_URL="+apiUrl)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse JSON
	var frames []map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &frames); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	if len(frames) == 0 {
		t.Log("CLI returned empty frames list")
	} else {
		t.Logf("CLI returned %d frames", len(frames))
	}
}

func TestCLI_ListBuilds(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	qsPath := resolveQSPath(t)

	cmd := exec.Command(qsPath, "builds", "list", "--json")
	cmd.Env = append(cmd.Env, "QS_API_URL="+apiUrl)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	var builds []interface{} // can be empty or list of maps
	if err := json.Unmarshal(stdout.Bytes(), &builds); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	t.Logf("CLI returned %d builds", len(builds))
}

func TestCLI_BuildsEvaluate(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	qsPath := resolveQSPath(t)

	// 1. Test default YAML output
	cmdYAML := exec.Command(qsPath, "builds", "evaluate", "bando-basher-5-inch")
	cmdYAML.Env = append(cmdYAML.Env, "QS_API_URL="+apiUrl)
	var stdoutYAML, stderrYAML bytes.Buffer
	cmdYAML.Stdout = &stdoutYAML
	cmdYAML.Stderr = &stderrYAML
	if err := cmdYAML.Run(); err != nil {
		t.Fatalf("CLI builds evaluate failed: %v\nStderr: %s", err, stderrYAML.String())
	}
	outputYAML := stdoutYAML.String()
	if !strings.Contains(outputYAML, "thrust_to_weight_ratio:") {
		t.Errorf("Expected YAML output with 'thrust_to_weight_ratio:', got:\n%s", outputYAML)
	}
	if !strings.Contains(outputYAML, "build_id: bando-basher-5-inch") {
		t.Errorf("Expected YAML output with 'build_id: bando-basher-5-inch', got:\n%s", outputYAML)
	}
	if !strings.Contains(outputYAML, "battery_id:") {
		t.Errorf("Expected YAML output with 'battery_id:', got:\n%s", outputYAML)
	}

	// 2. Test --json flag
	cmd := exec.Command(qsPath, "builds", "evaluate", "bando-basher-5-inch", "--json")
	cmd.Env = append(cmd.Env, "QS_API_URL="+apiUrl)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	var eval map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &eval); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	if eval["build_id"] != "bando-basher-5-inch" {
		t.Errorf("Expected build_id == 'bando-basher-5-inch', got: %v", eval["build_id"])
	}
	if eval["payload_weight_g"] != float64(0) {
		t.Errorf("Expected payload_weight_g == 0, got: %v", eval["payload_weight_g"])
	}
	if batId, ok := eval["battery_id"].(string); !ok || batId == "" {
		t.Errorf("Expected non-empty battery_id in evaluation response, got: %v", eval["battery_id"])
	}
	if _, ok := eval["thrust_to_weight_ratio"]; !ok {
		t.Errorf("Expected thrust_to_weight_ratio in evaluation response, got: %v", eval)
	}
	if _, ok := eval["hover_rpm"]; !ok {
		t.Errorf("Expected hover_rpm in evaluation response, got: %v", eval)
	}
	if _, ok := eval["system_messages"]; !ok {
		t.Errorf("Expected system_messages in evaluation response, got: %v", eval)
	}
	if eval["all_up_weight_g"] == nil || eval["all_up_weight_g"].(float64) <= 0 {
		t.Errorf("Expected positive all_up_weight_g in evaluation response, got: %v", eval["all_up_weight_g"])
	}
	t.Logf("CLI returned evaluation: %v", eval)

	// 3. Test --payload and --battery flags
	cmdCustom := exec.Command(qsPath, "builds", "evaluate", "bando-basher-5-inch", "--payload", "75", "--battery", "tattu-r-line-v5-1400mah-6s-150c", "--json")
	cmdCustom.Env = append(cmdCustom.Env, "QS_API_URL="+apiUrl)
	var stdoutCustom, stderrCustom bytes.Buffer
	cmdCustom.Stdout = &stdoutCustom
	cmdCustom.Stderr = &stderrCustom
	if err := cmdCustom.Run(); err != nil {
		t.Fatalf("CLI evaluate with custom payload/battery failed: %v\nStderr: %s", err, stderrCustom.String())
	}
	var evalCustom map[string]interface{}
	if err := json.Unmarshal(stdoutCustom.Bytes(), &evalCustom); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	if evalCustom["payload_weight_g"] != float64(75) {
		t.Errorf("Expected payload_weight_g == 75, got: %v", evalCustom["payload_weight_g"])
	}
	if evalCustom["battery_id"] != "tattu-r-line-v5-1400mah-6s-150c" {
		t.Errorf("Expected battery_id == 'tattu-r-line-v5-1400mah-6s-150c', got: %v", evalCustom["battery_id"])
	}

	// 4. Test that --yaml flag is NOT accepted since YAML is the default
	cmdRejectYAML := exec.Command(qsPath, "builds", "evaluate", "bando-basher-5-inch", "--yaml")
	cmdRejectYAML.Env = append(cmdRejectYAML.Env, "QS_API_URL="+apiUrl)
	var stderrReject bytes.Buffer
	cmdRejectYAML.Stderr = &stderrReject
	if err := cmdRejectYAML.Run(); err == nil {
		t.Fatalf("Expected CLI builds evaluate --yaml to fail, but it succeeded")
	} else if !strings.Contains(stderrReject.String(), "unknown flag: --yaml") {
		t.Errorf("Expected 'unknown flag: --yaml', got: %s", stderrReject.String())
	}

	// 5. Test ultralight-toothpick auto-selects lightest compatible battery
	cmdToothpick := exec.Command(qsPath, "builds", "evaluate", "ultralight-toothpick", "--json")
	cmdToothpick.Env = append(cmdToothpick.Env, "QS_API_URL="+apiUrl)
	var stdoutTP, stderrTP bytes.Buffer
	cmdToothpick.Stdout = &stdoutTP
	cmdToothpick.Stderr = &stderrTP
	if err := cmdToothpick.Run(); err != nil {
		t.Fatalf("CLI evaluate ultralight-toothpick failed: %v\nStderr: %s", err, stderrTP.String())
	}
	var evalTP map[string]interface{}
	if err := json.Unmarshal(stdoutTP.Bytes(), &evalTP); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}
	if evalTP["build_id"] != "ultralight-toothpick" {
		t.Errorf("Expected build_id == 'ultralight-toothpick', got: %v", evalTP["build_id"])
	}
	if batId, ok := evalTP["battery_id"].(string); !ok || batId == "" {
		t.Errorf("Expected auto-selected battery_id for ultralight-toothpick, got: %v", evalTP["battery_id"])
	}
}

func TestCLI_BuildsGet(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	qsPath := resolveQSPath(t)

	// 1. Test YAML output by default
	cmdYAML := exec.Command(qsPath, "builds", "get", "bando-basher-5-inch")
	cmdYAML.Env = append(cmdYAML.Env, "QS_API_URL="+apiUrl)
	var stdoutYAML, stderrYAML bytes.Buffer
	cmdYAML.Stdout = &stdoutYAML
	cmdYAML.Stderr = &stderrYAML
	if err := cmdYAML.Run(); err != nil {
		t.Fatalf("CLI builds get failed: %v\nStderr: %s", err, stderrYAML.String())
	}
	outputYAML := stdoutYAML.String()
	if !strings.Contains(outputYAML, "id: bando-basher-5-inch") {
		t.Errorf("Expected YAML output with 'id: bando-basher-5-inch', got:\n%s", outputYAML)
	}

	// 2. Test --json flag
	cmdJSON := exec.Command(qsPath, "builds", "get", "bando-basher-5-inch", "--json")
	cmdJSON.Env = append(cmdJSON.Env, "QS_API_URL="+apiUrl)
	var stdoutJSON, stderrJSON bytes.Buffer
	cmdJSON.Stdout = &stdoutJSON
	cmdJSON.Stderr = &stderrJSON
	if err := cmdJSON.Run(); err != nil {
		t.Fatalf("CLI builds get --json failed: %v\nStderr: %s", err, stderrJSON.String())
	}
	var resJSON map[string]interface{}
	if err := json.Unmarshal(stdoutJSON.Bytes(), &resJSON); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, stdoutJSON.String())
	}
	if resJSON["id"] != "bando-basher-5-inch" {
		t.Errorf("Expected id == 'bando-basher-5-inch', got: %v", resJSON["id"])
	}

	// 3. Test that --yaml flag is NOT accepted since YAML is the default
	cmdGetRejectYAML := exec.Command(qsPath, "builds", "get", "bando-basher-5-inch", "--yaml")
	cmdGetRejectYAML.Env = append(cmdGetRejectYAML.Env, "QS_API_URL="+apiUrl)
	var stderrGetReject bytes.Buffer
	cmdGetRejectYAML.Stderr = &stderrGetReject
	if err := cmdGetRejectYAML.Run(); err == nil {
		t.Fatalf("Expected CLI builds get --yaml to fail, but it succeeded")
	} else if !strings.Contains(stderrGetReject.String(), "unknown flag: --yaml") {
		t.Errorf("Expected 'unknown flag: --yaml', got: %s", stderrGetReject.String())
	}
}
