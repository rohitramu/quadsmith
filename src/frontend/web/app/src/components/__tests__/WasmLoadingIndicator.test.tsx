import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { WasmLoadingIndicator } from "../WasmLoadingIndicator";

describe("WasmLoadingIndicator Component", () => {
  it("renders progress percentage and MB metrics", () => {
    const mockProgress = {
      loadedBytes: 10 * 1024 * 1024,
      totalBytes: 16 * 1024 * 1024,
      percent: 63,
    };

    render(<WasmLoadingIndicator progress={mockProgress} />);

    expect(screen.getByText("Build Evaluator")).toBeInTheDocument();
    expect(screen.getByText("Loading physics engine...")).toBeInTheDocument();
    expect(screen.getByText("> 3s Delayed")).toBeInTheDocument();
    expect(screen.getByText("Downloading Physics Engine")).toBeInTheDocument();
    expect(screen.getByText("63%")).toBeInTheDocument();
    expect(screen.getByText(/10.00 MB \/ 16.00 MB/)).toBeInTheDocument();
    expect(screen.getByText("Physics Engine Core")).toBeInTheDocument();
  });

  it("handles null progress gracefully with zero defaults", () => {
    render(<WasmLoadingIndicator progress={null} />);

    expect(screen.getByText("0%")).toBeInTheDocument();
    expect(screen.getByText(/0.00 MB \/ 16.00 MB/)).toBeInTheDocument();
  });
});
