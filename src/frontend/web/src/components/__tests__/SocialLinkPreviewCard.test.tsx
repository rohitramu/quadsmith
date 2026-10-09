import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { SocialLinkPreviewCard } from "../SocialLinkPreviewCard";
import { ReferenceLinkType } from "../../gen/quadsmith/reference_link_pb";

describe("SocialLinkPreviewCard", () => {
  it("renders preview with mock transport data", async () => {
    const link = {
      type: ReferenceLinkType.PURCHASE,
      url: "https://pyrodrone.com/products/motor",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    // Check loading skeleton appears first or immediately resolves
    await waitFor(() => {
      expect(screen.getByText("Purchase")).toBeInTheDocument();
      expect(
        screen.getByText("Preview for https://pyrodrone.com/products/motor"),
      ).toBeInTheDocument();
    });

    expect(screen.getByText("A simulated mock link preview description.")).toBeInTheDocument();
    expect(screen.getByText("example.com")).toBeInTheDocument();
  });

  it("renders correct badge for documentation link", async () => {
    const link = {
      type: ReferenceLinkType.DOCUMENTATION,
      url: "https://github.com/betaflight/betaflight",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Documentation")).toBeInTheDocument();
    });
  });
});
