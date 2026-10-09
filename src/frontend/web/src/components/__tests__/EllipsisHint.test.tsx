import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, act, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import {
  EllipsisHint,
  isElementEllipsized,
  getElementFullText,
  findEllipsizedTarget,
} from "../EllipsisHint";

describe("EllipsisHint Component & Utilities", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  describe("isElementEllipsized utility", () => {
    it("returns false for elements without text", () => {
      const el = document.createElement("div");
      expect(isElementEllipsized(el)).toBe(false);
    });

    it("returns false for input elements", () => {
      const input = document.createElement("input");
      input.value = "Some long text";
      expect(isElementEllipsized(input)).toBe(false);
    });

    it("returns true for elements with truncate class when scrollWidth > clientWidth", () => {
      const el = document.createElement("span");
      el.className = "truncate";
      el.textContent = "Very long overflowing drone component title text";

      // Mock styles and dimensions
      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "ellipsis",
        overflow: "hidden",
        webkitLineClamp: "none",
      } as any);

      Object.defineProperty(el, "scrollWidth", { value: 200, configurable: true });
      Object.defineProperty(el, "clientWidth", { value: 100, configurable: true });

      expect(isElementEllipsized(el)).toBe(true);
    });

    it("returns false for elements with truncate class when text fits completely", () => {
      const el = document.createElement("span");
      el.className = "truncate";
      el.textContent = "Short";

      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "ellipsis",
        overflow: "hidden",
        webkitLineClamp: "none",
      } as any);

      Object.defineProperty(el, "scrollWidth", { value: 50, configurable: true });
      Object.defineProperty(el, "clientWidth", { value: 100, configurable: true });

      expect(isElementEllipsized(el)).toBe(false);
    });

    it("returns true for line-clamp elements when scrollHeight > clientHeight", () => {
      const el = document.createElement("p");
      el.className = "line-clamp-2";
      el.textContent = "Multi-line clamped paragraph describing hardware features in detail.";

      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "clip",
        overflow: "hidden",
        webkitLineClamp: "2",
      } as any);

      Object.defineProperty(el, "scrollHeight", { value: 80, configurable: true });
      Object.defineProperty(el, "clientHeight", { value: 40, configurable: true });

      expect(isElementEllipsized(el)).toBe(true);
    });
  });

  describe("getElementFullText", () => {
    it("returns normalized text content", () => {
      const el = document.createElement("div");
      el.textContent = "  Quadsmith   Flight   Controller   ";
      expect(getElementFullText(el)).toBe("Quadsmith Flight Controller");
    });

    it("prefers data-full-text attribute when available", () => {
      const el = document.createElement("div");
      el.setAttribute("data-full-text", "Unabridged Full Text Here");
      el.textContent = "Unabridged...";
      expect(getElementFullText(el)).toBe("Unabridged Full Text Here");
    });
  });

  describe("findEllipsizedTarget", () => {
    it("finds ancestor if ancestor has ellipsis", () => {
      const parent = document.createElement("div");
      parent.className = "truncate";

      const child = document.createElement("span");
      child.textContent = "Long title text";
      parent.appendChild(child);
      document.body.appendChild(parent);

      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "ellipsis",
        overflow: "hidden",
        webkitLineClamp: "none",
      } as any);

      Object.defineProperty(parent, "scrollWidth", { value: 300, configurable: true });
      Object.defineProperty(parent, "clientWidth", { value: 150, configurable: true });

      const found = findEllipsizedTarget(child);
      expect(found).not.toBeNull();
      expect(found?.element).toBe(parent);
      expect(found?.text).toBe("Long title text");
    });
  });

  describe("EllipsisHint live behavior", () => {
    it("displays hint upon hover of an overflowing element and hides on pointerout", () => {
      const { unmount } = renderWithProviders(
        <div>
          <div id="test-target" className="truncate" style={{ width: "100px", overflow: "hidden" }}>
            A very long truncated build name
          </div>
          <EllipsisHint />
        </div>,
      );

      const target = document.getElementById("test-target")!;

      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "ellipsis",
        overflow: "hidden",
        webkitLineClamp: "none",
      } as any);

      Object.defineProperty(target, "scrollWidth", { value: 300, configurable: true });
      Object.defineProperty(target, "clientWidth", { value: 100, configurable: true });
      vi.spyOn(target, "getBoundingClientRect").mockReturnValue({
        top: 100,
        bottom: 120,
        left: 50,
        right: 150,
        width: 100,
        height: 20,
      } as any);

      // Initially no hint
      expect(screen.queryByTestId("ellipsis-hint")).not.toBeInTheDocument();

      // Fire pointerover
      act(() => {
        fireEvent.pointerOver(target);
        vi.advanceTimersByTime(200);
      });

      // Hint should now appear with full text
      const hint = screen.getByTestId("ellipsis-hint");
      expect(hint).toBeInTheDocument();
      expect(hint).toHaveTextContent("A very long truncated build name");

      // Fire pointerout
      act(() => {
        fireEvent.pointerOut(target);
      });

      // Hint should disappear
      expect(screen.queryByTestId("ellipsis-hint")).not.toBeInTheDocument();

      unmount();
    });

    it("dismisses hint immediately on Escape key", () => {
      const { unmount } = renderWithProviders(
        <div>
          <div id="test-esc" className="truncate">
            Long text for escape test
          </div>
          <EllipsisHint />
        </div>,
      );

      const target = document.getElementById("test-esc")!;

      vi.spyOn(window, "getComputedStyle").mockReturnValue({
        textOverflow: "ellipsis",
        overflow: "hidden",
        webkitLineClamp: "none",
      } as any);

      Object.defineProperty(target, "scrollWidth", { value: 250, configurable: true });
      Object.defineProperty(target, "clientWidth", { value: 100, configurable: true });
      vi.spyOn(target, "getBoundingClientRect").mockReturnValue({
        top: 100,
        bottom: 120,
        left: 50,
        right: 150,
        width: 100,
        height: 20,
      } as any);

      act(() => {
        fireEvent.pointerOver(target);
        vi.advanceTimersByTime(200);
      });

      expect(screen.getByTestId("ellipsis-hint")).toBeInTheDocument();

      // Press Escape
      act(() => {
        fireEvent.keyDown(window, { key: "Escape" });
      });

      expect(screen.queryByTestId("ellipsis-hint")).not.toBeInTheDocument();

      unmount();
    });
  });
});
