import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { wasmEngine } from "../wasmEngine";
import { create } from "@bufbuild/protobuf";
import { AssembledComponentsSchema } from "../../gen/quadsmith/components_pb";

describe("wasmEngine Client", () => {
  beforeEach(() => {
    delete (window as any).Go;
    delete (window as any).__quadsmith_engine;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("has correct initial idle state", () => {
    expect(wasmEngine.getStatus()).toBe("idle");
    expect(wasmEngine.getIsReady()).toBe(false);
    expect(wasmEngine.getIsDelayed()).toBe(false);
    expect(wasmEngine.getProgress()).toBe(null);
    expect(wasmEngine.getError()).toBe(null);
  });

  it("notifies subscribers and records error when fetch fails", async () => {
    // Provide Mock Go so it doesn't try to append real DOM script
    (window as any).Go = class MockGo {
      importObject = {};
      run = vi.fn().mockResolvedValue(undefined);
    };

    const subscriber = vi.fn();
    const unsubscribe = wasmEngine.subscribe(subscriber);

    vi.spyOn(window, "fetch").mockImplementationOnce(() =>
      Promise.reject(new Error("Network test failure")),
    );

    await expect(wasmEngine.load()).rejects.toThrow("Network test failure");
    expect(subscriber).toHaveBeenCalled();
    expect(wasmEngine.getStatus()).toBe("error");
    expect(wasmEngine.getError()).toBe("Network test failure");

    unsubscribe();
  });

  it("bridges generateCelFilter when engine is initialized on window", () => {
    (window as any).__quadsmith_engine = {
      evaluateComponents: vi.fn(),
      computeElectricalLimits: vi.fn(),
      checkCompatibility: vi.fn(),
      generateCelFilter: vi.fn().mockReturnValue("diameter_mm <= 127.0"),
      ready: true,
    };

    const comps = create(AssembledComponentsSchema, {});
    const filter = wasmEngine.generateCelFilter("propellers", comps);

    expect((window as any).__quadsmith_engine.generateCelFilter).toHaveBeenCalled();
    expect(filter).toBe("diameter_mm <= 127.0");
  });
});
