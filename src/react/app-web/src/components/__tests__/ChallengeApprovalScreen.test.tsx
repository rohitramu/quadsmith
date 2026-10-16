import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test/test-utils";
import { ChallengeApprovalScreen } from "../ChallengeApprovalScreen";

describe("ChallengeApprovalScreen Component", () => {
  it("renders login challenge details and approves login successfully", async () => {
    const user = userEvent.setup();
    const onApproved = vi.fn();

    renderWithProviders(
      <ChallengeApprovalScreen token="valid-login-token" onApproved={onApproved} />,
      { transportOptions: { isAuthenticated: false } },
    );

    expect(
      await screen.findByRole("heading", { name: /approve login request/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/a device is requesting to sign in to your quadsmith account/i),
    ).toBeInTheDocument();
    expect(screen.getByText("alex@fpv.build")).toBeInTheDocument();
    expect(screen.getByText("Chrome on macOS")).toBeInTheDocument();
    expect(screen.getByText(/192.168.1.42 \(Local Network\)/i)).toBeInTheDocument();

    const approveBtn = screen.getByTestId("approve-login-btn");
    await user.click(approveBtn);

    expect(await screen.findByTestId("approval-success-card")).toBeInTheDocument();
    expect(screen.getByText(/login approved!/i)).toBeInTheDocument();
    expect(screen.getByText(/you can safely close this window now/i)).toBeInTheDocument();
    expect(onApproved).toHaveBeenCalledTimes(1);
  });

  it("renders denial state and triggers onDenied callback when denied", async () => {
    const user = userEvent.setup();
    const onDenied = vi.fn();

    renderWithProviders(<ChallengeApprovalScreen token="valid-login-token" onDenied={onDenied} />, {
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", { name: /approve login request/i }),
    ).toBeInTheDocument();

    const denyBtn = screen.getByTestId("deny-login-btn");
    await user.click(denyBtn);

    expect(await screen.findByTestId("approval-denied-card")).toBeInTheDocument();
    expect(screen.getByText(/login request denied/i)).toBeInTheDocument();
    expect(screen.getByText(/you can safely close this window now/i)).toBeInTheDocument();
    expect(onDenied).toHaveBeenCalledTimes(1);
  });

  it("renders custom headings and does not offer fallback download button on approving device for EXPORT_DATA purpose", async () => {
    const user = userEvent.setup();
    const onApproved = vi.fn();

    renderWithProviders(
      <ChallengeApprovalScreen token="valid-export-token" onApproved={onApproved} />,
      { transportOptions: { isAuthenticated: false } },
    );

    expect(
      await screen.findByRole("heading", { name: /approve data export/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/a device is requesting to export your quadsmith account data/i),
    ).toBeInTheDocument();

    const approveBtn = screen.getByRole("button", { name: /approve data export/i });
    await user.click(approveBtn);

    expect(await screen.findByTestId("approval-success-card")).toBeInTheDocument();
    expect(screen.getByText(/export approved!/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /your data export has been approved and the download has been initiated on your other device/i,
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /download data archive/i })).not.toBeInTheDocument();
    expect(onApproved).toHaveBeenCalledTimes(1);
  });

  it("renders custom headings for ADD_EMAIL purpose", async () => {
    renderWithProviders(<ChallengeApprovalScreen token="valid-add-email-token" />, {
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", { name: /approve email verification/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/a device is requesting to link this email address to your account/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /approve email/i })).toBeInTheDocument();
  });

  it("renders custom headings for DELETE_ACCOUNT purpose", async () => {
    renderWithProviders(<ChallengeApprovalScreen token="valid-delete-token" />, {
      transportOptions: { isAuthenticated: false },
    });

    expect(
      await screen.findByRole("heading", { name: /approve account deletion/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/a device is requesting to delete your quadsmith account/i),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /confirm account deletion/i })).toBeInTheDocument();
  });

  it("displays error card and triggers onError on invalid or expired token", async () => {
    const onError = vi.fn();

    renderWithProviders(<ChallengeApprovalScreen token="invalid-token" onError={onError} />, {
      transportOptions: { isAuthenticated: false },
    });

    expect(await screen.findByTestId("approval-error-card")).toBeInTheDocument();
    expect(screen.getByText(/invalid or expired link/i)).toBeInTheDocument();
    expect(onError).toHaveBeenCalledTimes(1);
  });
});
