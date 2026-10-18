import { useState } from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ErrorBoundary } from "@quadsmith/shared";

function ThrowingChild({
  shouldThrow,
  message = "Simulated render failure",
}: {
  shouldThrow: boolean;
  message?: string;
}) {
  if (shouldThrow) {
    throw new Error(message);
  }
  return <div data-testid="healthy-child">Healthy Content</div>;
}

describe("ErrorBoundary Component", () => {
  it("renders children normally when no error occurs", () => {
    render(
      <ErrorBoundary>
        <ThrowingChild shouldThrow={false} />
      </ErrorBoundary>,
    );

    expect(screen.getByTestId("healthy-child")).toBeInTheDocument();
    expect(screen.queryByTestId("error-boundary-fallback")).toBeNull();
  });

  it("catches render errors in page variant, displays diagnostic message, and recovers on Try Again", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const user = userEvent.setup();

    function RecoverableHarness() {
      const [shouldThrow, setShouldThrow] = useState(true);
      return (
        <ErrorBoundary
          variant="page"
          title="Page failed to render"
          onReset={() => setShouldThrow(false)}
        >
          <ThrowingChild shouldThrow={shouldThrow} message="Unexpected null reference" />
        </ErrorBoundary>
      );
    }

    render(<RecoverableHarness />);

    const alert = screen.getByTestId("error-boundary-fallback");
    expect(alert).toBeInTheDocument();
    expect(screen.getByText("Page failed to render")).toBeInTheDocument();
    expect(screen.getByText("Unexpected null reference")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /try again/i }));

    expect(screen.getByTestId("healthy-child")).toBeInTheDocument();
    expect(screen.queryByTestId("error-boundary-fallback")).toBeNull();

    consoleErrorSpy.mockRestore();
  });

  it("isolates panel errors in panel variant and automatically resets when resetKeys change", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const user = userEvent.setup();

    function StepHarness() {
      const [step, setStep] = useState(1);
      return (
        <div>
          <div data-testid="outer-shell">Persistent Page Header</div>
          <button type="button" onClick={() => setStep(2)}>
            Switch to Step 2
          </button>
          <ErrorBoundary variant="panel" title="Build Evaluator Error" resetKeys={[step]}>
            <ThrowingChild shouldThrow={step === 1} message="Evaluator panel crashed" />
          </ErrorBoundary>
        </div>
      );
    }

    render(<StepHarness />);

    // Outer shell remains rendered while panel shows inline error
    expect(screen.getByTestId("outer-shell")).toBeInTheDocument();
    expect(screen.getByText("Build Evaluator Error")).toBeInTheDocument();
    expect(screen.getByText("Evaluator panel crashed")).toBeInTheDocument();

    // Changing resetKeys (switching to Step 2) automatically recovers the boundary
    await user.click(screen.getByRole("button", { name: /switch to step 2/i }));

    expect(screen.getByTestId("healthy-child")).toBeInTheDocument();
    expect(screen.queryByTestId("error-boundary-fallback")).toBeNull();

    consoleErrorSpy.mockRestore();
  });
});
