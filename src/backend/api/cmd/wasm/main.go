//go:build js && wasm

package main

import (
	"syscall/js"

	"google.golang.org/protobuf/proto"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/internal/engines/compatibility"
	"quadsmith/api/internal/engines/evaluator"
)

func evaluateComponents(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return js.ValueOf("missing argument")
	}

	uint8Array := args[0]
	length := uint8Array.Get("byteLength").Int()
	bytes := make([]byte, length)
	js.CopyBytesToGo(bytes, uint8Array)

	var req pb.EvaluateComponentsRequest
	if err := proto.Unmarshal(bytes, &req); err != nil {
		return js.ValueOf("unmarshal error: " + err.Error())
	}

	res, err := evaluator.EvaluateComponentsDirect(req.GetComponents(), req.GetPayloadWeightG())
	if err != nil {
		return js.ValueOf("evaluation error: " + err.Error())
	}

	outBytes, err := proto.Marshal(res)
	if err != nil {
		return js.ValueOf("marshal error: " + err.Error())
	}

	dst := js.Global().Get("Uint8Array").New(len(outBytes))
	js.CopyBytesToJS(dst, outBytes)
	return dst
}

func computeElectricalLimits(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return js.ValueOf("missing argument")
	}

	uint8Array := args[0]
	length := uint8Array.Get("byteLength").Int()
	bytes := make([]byte, length)
	js.CopyBytesToGo(bytes, uint8Array)

	var req pb.GetComponentsElectricalLimitsRequest
	if err := proto.Unmarshal(bytes, &req); err != nil {
		return js.ValueOf("unmarshal error: " + err.Error())
	}

	comps := req.GetComponents()
	var motorCount uint32 = 4
	if comps != nil && comps.Frame != nil && comps.Frame.MotorCount > 0 {
		motorCount = comps.Frame.MotorCount
	}

	var fc *pb.FlightController
	var escs []*pb.ElectronicSpeedController
	var motor *pb.Motor
	if comps != nil {
		fc = comps.FlightController
		escs = comps.ElectronicSpeedControllers
		motor = comps.Motor
	}

	limits := evaluator.ComputeElectricalLimits(fc, escs, motor, motorCount)
	if len(req.GetCandidateBatteries()) > 0 {
		limits.DefaultBatteryId = evaluator.FindLightestCompatibleBatteryFromList(
			req.GetCandidateBatteries(),
			limits.MinVoltage,
			limits.MaxVoltage,
			limits.MaxCurrentA,
		)
	}

	outBytes, err := proto.Marshal(limits)
	if err != nil {
		return js.ValueOf("marshal error: " + err.Error())
	}

	dst := js.Global().Get("Uint8Array").New(len(outBytes))
	js.CopyBytesToJS(dst, outBytes)
	return dst
}

func checkCompatibility(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return js.ValueOf("missing argument")
	}

	uint8Array := args[0]
	length := uint8Array.Get("byteLength").Int()
	bytes := make([]byte, length)
	js.CopyBytesToGo(bytes, uint8Array)

	var req pb.CheckComponentsCompatibilityRequest
	if err := proto.Unmarshal(bytes, &req); err != nil {
		return js.ValueOf("unmarshal error: " + err.Error())
	}

	messages := compatibility.CheckCompatibility(req.GetComponents())
	res := &pb.CheckCompatibilityResponse{
		Messages: messages,
	}

	outBytes, err := proto.Marshal(res)
	if err != nil {
		return js.ValueOf("marshal error: " + err.Error())
	}

	dst := js.Global().Get("Uint8Array").New(len(outBytes))
	js.CopyBytesToJS(dst, outBytes)
	return dst
}

func generateCelFilter(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return js.ValueOf("")
	}

	targetCollection := args[0].String()
	uint8Array := args[1]
	length := uint8Array.Get("byteLength").Int()
	bytes := make([]byte, length)
	js.CopyBytesToGo(bytes, uint8Array)

	var comps pb.AssembledComponents
	if err := proto.Unmarshal(bytes, &comps); err != nil {
		return js.ValueOf("")
	}

	filter := compatibility.GenerateCelFilter(targetCollection, &comps)
	return js.ValueOf(filter)
}

func main() {
	engine := js.Global().Get("Object").New()
	engine.Set("evaluateComponents", js.FuncOf(evaluateComponents))
	engine.Set("computeElectricalLimits", js.FuncOf(computeElectricalLimits))
	engine.Set("checkCompatibility", js.FuncOf(checkCompatibility))
	engine.Set("generateCelFilter", js.FuncOf(generateCelFilter))
	engine.Set("ready", true)

	js.Global().Set("__quadsmith_engine", engine)

	// Notify any listeners waiting for the engine
	window := js.Global().Get("window")
	if !window.IsUndefined() && !window.IsNull() {
		event := js.Global().Get("CustomEvent").New("quadsmith:engine-ready")
		window.Call("dispatchEvent", event)
	}

	// Keep runtime running
	select {}
}
