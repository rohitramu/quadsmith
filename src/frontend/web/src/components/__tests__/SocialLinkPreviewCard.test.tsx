import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import {
  SocialLinkPreviewCard,
  validateReferenceLink,
  validateReferenceLinks,
} from "../SocialLinkPreviewCard";
import { ReferenceLinkType } from "../../gen/quadsmith/reference_link_pb";

describe("SocialLinkPreviewCard", () => {
  it("renders preview with mock transport data", async () => {
    const link = {
      types: [ReferenceLinkType.PURCHASE],
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
      types: [ReferenceLinkType.DOCUMENTATION],
      url: "https://github.com/betaflight/betaflight",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Documentation")).toBeInTheDocument();
    });
  });

  it("renders multiple badges when a link belongs to multiple categories", async () => {
    const link = {
      types: [ReferenceLinkType.PRODUCT_PAGE, ReferenceLinkType.PURCHASE],
      url: "https://emax-usa.com/products/eco-ii-2207",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Official Product Page")).toBeInTheDocument();
      expect(screen.getByText("Purchase")).toBeInTheDocument();
    });
  });

  it("infers 'Other' category when types array is empty", async () => {
    const link = {
      types: [],
      url: "https://example.com/other-link",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Other")).toBeInTheDocument();
    });
  });

  it("infers 'Other' category when types is not provided", async () => {
    const link = {
      url: "https://example.com/no-types",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Other")).toBeInTheDocument();
    });
  });

  it("throws validation error when UNSPECIFIED type is provided", () => {
    const link = {
      types: [ReferenceLinkType.UNSPECIFIED],
      url: "https://example.com/unspecified",
    };

    expect(() => validateReferenceLink(link)).toThrow(
      "ReferenceLinkType.UNSPECIFIED is not permitted",
    );
  });

  it("throws validation error when UNSPECIFIED is included among other types", () => {
    const link = {
      types: [ReferenceLinkType.PURCHASE, ReferenceLinkType.UNSPECIFIED],
      url: "https://example.com/multi-unspecified",
    };

    expect(() => validateReferenceLink(link)).toThrow(
      "ReferenceLinkType.UNSPECIFIED is not permitted",
    );
  });

  it("throws validation error on duplicate reference link URLs", () => {
    const links = [
      {
        types: [ReferenceLinkType.PRODUCT_PAGE],
        url: "https://example.com/product",
      },
      {
        types: [ReferenceLinkType.PURCHASE],
        url: "https://example.com/product/",
      },
    ];

    expect(() => validateReferenceLinks(links)).toThrow("Duplicate reference link URL");
  });

  it("permits multiple links when URLs are distinct", () => {
    const links = [
      {
        types: [ReferenceLinkType.PRODUCT_PAGE],
        url: "https://example.com/product",
      },
      {
        types: [ReferenceLinkType.PURCHASE],
        url: "https://example.com/store/item",
      },
    ];

    expect(() => validateReferenceLinks(links)).not.toThrow();
  });

  it("gracefully falls back when legacy type field is provided", async () => {
    const link = {
      type: ReferenceLinkType.REVIEW,
      url: "https://youtube.com/watch?v=123",
    };

    renderWithProviders(<SocialLinkPreviewCard link={link} />);

    await waitFor(() => {
      expect(screen.getByText("Review")).toBeInTheDocument();
    });
  });
});
