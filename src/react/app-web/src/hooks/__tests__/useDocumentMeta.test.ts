import { describe, it, expect, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useDocumentMeta } from "../useDocumentMeta";

describe("useDocumentMeta", () => {
  beforeEach(() => {
    document.title = "";
    document.head.innerHTML = "";
  });

  it("updates document.title and creates meta tags in document.head", () => {
    renderHook(() =>
      useDocumentMeta({
        title: "Custom Build — Quadsmith",
        description: "Custom build description for testing.",
        image: "/custom-image.png",
        type: "article",
      }),
    );

    expect(document.title).toBe("Custom Build — Quadsmith");

    const descMeta = document.querySelector('meta[name="description"]');
    expect(descMeta?.getAttribute("content")).toBe("Custom build description for testing.");

    const ogTitle = document.querySelector('meta[property="og:title"]');
    expect(ogTitle?.getAttribute("content")).toBe("Custom Build — Quadsmith");

    const ogType = document.querySelector('meta[property="og:type"]');
    expect(ogType?.getAttribute("content")).toBe("article");

    const twitterCard = document.querySelector('meta[name="twitter:card"]');
    expect(twitterCard?.getAttribute("content")).toBe("summary_large_image");
  });
});
