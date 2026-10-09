import { useEffect } from "react";

export interface DocumentMetaOptions {
  title?: string;
  description?: string;
  image?: string;
  type?: "website" | "article";
}

const DEFAULT_TITLE = "Quadsmith — FPV Drone Component Platform";
const DEFAULT_DESCRIPTION =
  "Design, optimize, and evaluate custom FPV drone builds. Calculate thrust-to-weight, simulate flight times, and verify hardware compatibility.";
const DEFAULT_IMAGE = "/og-default.png";

/**
 * useDocumentMeta synchronizes the document title and Open Graph / Twitter Card
 * meta tags with the current route during client-side SPA navigation.
 */
export function useDocumentMeta(options: DocumentMetaOptions = {}) {
  const {
    title = DEFAULT_TITLE,
    description = DEFAULT_DESCRIPTION,
    image = DEFAULT_IMAGE,
    type = "website",
  } = options;

  useEffect(() => {
    // 1. Update Document Title
    document.title = title;

    // 2. Helper to set or create a meta tag
    const setMeta = (attribute: "name" | "property", key: string, content: string) => {
      let element = document.querySelector<HTMLMetaElement>(`meta[${attribute}="${key}"]`);
      if (!element) {
        element = document.createElement("meta");
        element.setAttribute(attribute, key);
        document.head.appendChild(element);
      }
      element.setAttribute("content", content);
    };

    // Resolve absolute image URL if relative
    let resolvedImage = image;
    if (image.startsWith("/")) {
      resolvedImage = `${window.location.origin}${image}`;
    }

    setMeta("name", "description", description);
    setMeta("property", "og:site_name", "Quadsmith");
    setMeta("property", "og:type", type);
    setMeta("property", "og:title", title);
    setMeta("property", "og:description", description);
    setMeta("property", "og:image", resolvedImage);
    setMeta("property", "og:url", window.location.href);

    setMeta("name", "twitter:card", "summary_large_image");
    setMeta("name", "twitter:title", title);
    setMeta("name", "twitter:description", description);
    setMeta("name", "twitter:image", resolvedImage);
  }, [title, description, image, type]);
}
