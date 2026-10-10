import { useState } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { ExternalLink, Globe } from "lucide-react";
import { getLinkPreview } from "../gen/quadsmith/link_preview-LinkPreviewService_connectquery";
import { ReferenceLinkType } from "../gen/quadsmith/reference_link_pb";
import type { ReferenceLink } from "../gen/quadsmith/reference_link_pb";

const LINK_TYPE_LABELS: Record<number, string> = {
  [ReferenceLinkType.PURCHASE]: "Purchase",
  [ReferenceLinkType.PRODUCT_PAGE]: "Official Product Page",
  [ReferenceLinkType.DOCUMENTATION]: "Documentation",
  [ReferenceLinkType.FORUM_POST]: "Forum Discussion",
  [ReferenceLinkType.REVIEW]: "Review",
};

const LINK_TYPE_STYLES: Record<number, string> = {
  [ReferenceLinkType.PURCHASE]:
    "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800",
  [ReferenceLinkType.PRODUCT_PAGE]:
    "bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300 border-blue-200 dark:border-blue-800",
  [ReferenceLinkType.DOCUMENTATION]:
    "bg-purple-50 text-purple-700 dark:bg-purple-950/40 dark:text-purple-300 border-purple-200 dark:border-purple-800",
  [ReferenceLinkType.REVIEW]:
    "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300 border-amber-200 dark:border-amber-800",
  [ReferenceLinkType.FORUM_POST]:
    "bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300 border-sky-200 dark:border-sky-800",
};

export const OTHER_LABEL = "Other";
export const OTHER_STYLE =
  "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 border-zinc-200 dark:border-zinc-700";

export function validateReferenceLink(link: {
  types?: (ReferenceLinkType | number)[];
  type?: ReferenceLinkType | number;
  url?: string;
}) {
  const rawTypes = Array.isArray(link.types)
    ? link.types
    : link.type !== undefined && link.type !== null
      ? [link.type]
      : [];

  for (const t of rawTypes) {
    if (t === ReferenceLinkType.UNSPECIFIED || t === 0) {
      throw new Error(
        "ReferenceLinkType.UNSPECIFIED is not permitted. Omit categories or use an empty array to infer 'Other'.",
      );
    }
  }
}

export function validateReferenceLinks(
  links?: Array<{
    types?: (ReferenceLinkType | number)[];
    type?: ReferenceLinkType | number;
    url?: string;
  } | null>,
) {
  if (!links) return;
  const seen = new Set<string>();
  for (const link of links) {
    if (!link) continue;
    validateReferenceLink(link);
    if (link.url) {
      const norm = link.url.trim().replace(/\/+$/, "");
      if (norm) {
        if (seen.has(norm)) {
          throw new Error(
            `Duplicate reference link URL "${link.url}" is not permitted. Combine categories into the types array of a single link.`,
          );
        }
        seen.add(norm);
      }
    }
  }
}

export function consolidateReferenceLinks<
  T extends {
    url?: string;
    types?: (ReferenceLinkType | number)[];
    type?: ReferenceLinkType | number;
  },
>(links?: T[] | null): T[] {
  if (!links || links.length === 0) return [];

  const map = new Map<string, T>();

  for (const link of links) {
    if (!link || !link.url) continue;
    const normUrl = link.url.trim().replace(/\/+$/, "").toLowerCase();

    const rawTypes: number[] = [];
    if (Array.isArray(link.types)) {
      rawTypes.push(...link.types);
    }
    if (link.type !== undefined && link.type !== null) {
      rawTypes.push(link.type);
    }
    const validTypes = rawTypes.filter((t) => t !== ReferenceLinkType.UNSPECIFIED && t !== 0);

    const existing = map.get(normUrl);
    if (!existing) {
      map.set(normUrl, {
        ...link,
        types: validTypes,
      });
    } else {
      const existingTypes = Array.isArray(existing.types) ? existing.types : [];
      const merged = Array.from(new Set([...existingTypes, ...validTypes]));
      existing.types = merged;
    }
  }

  return Array.from(map.values());
}

export function ReferenceLinksList({
  links,
  className = "",
}: {
  links?: any[] | null;
  className?: string;
}) {
  const consolidated = consolidateReferenceLinks(links);
  if (!consolidated || consolidated.length === 0) return null;

  return (
    <div className={`flex flex-col gap-2.5 ${className}`}>
      {consolidated.map((link, idx) => (
        <SocialLinkPreviewCard key={link.url || idx} link={link} />
      ))}
    </div>
  );
}

export interface SocialLinkPreviewCardProps {
  link:
    | ReferenceLink
    | {
        types?: ReferenceLinkType[] | number[];
        type?: ReferenceLinkType | number;
        url: string;
      };
  className?: string;
}

export function SocialLinkPreviewCard({ link, className = "" }: SocialLinkPreviewCardProps) {
  validateReferenceLink(link);

  const [imageError, setImageError] = useState(false);
  const [faviconError, setFaviconError] = useState(false);

  const { data: preview, isLoading } = useQuery(
    getLinkPreview,
    { url: link.url },
    {
      staleTime: 1000 * 60 * 60, // Cache client-side for 1 hour
      retry: 1,
      enabled: !!link.url,
    },
  );

  let hostname = "";
  try {
    hostname = new URL(link.url).hostname.replace(/^www\./, "");
  } catch {
    hostname = link.url;
  }

  const rawLink = link as { types?: number[]; type?: number };
  const rawList: number[] = [];
  if (Array.isArray(rawLink.types)) {
    rawList.push(...rawLink.types);
  }
  if (rawLink.type !== undefined && rawLink.type !== null) {
    rawList.push(rawLink.type);
  }

  const linkTypes: number[] = Array.from(
    new Set(rawList.filter((t) => t !== ReferenceLinkType.UNSPECIFIED && t !== 0)),
  );
  const isInferredOther = linkTypes.length === 0;

  const title = preview?.title || hostname;
  const description = preview?.description;
  const siteName = preview?.siteName || hostname;
  const hasImage = preview?.image && !imageError;

  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      className={`group flex flex-col sm:flex-row items-stretch bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800/80 border border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 rounded-xl overflow-hidden transition-all shadow-xs hover:shadow-sm ${className}`}
    >
      {/* Thumbnail Image (if available) */}
      {hasImage && (
        <div className="sm:w-36 h-28 sm:h-auto shrink-0 bg-zinc-100 dark:bg-zinc-800 relative overflow-hidden">
          <img
            src={preview.image}
            alt=""
            onError={() => setImageError(true)}
            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
            loading="lazy"
          />
        </div>
      )}

      {/* Main Content Info */}
      <div className="flex-1 p-3.5 flex flex-col justify-between min-w-0">
        <div>
          {/* Header Row: Badge Chips, Site Favicon/Host, External Icon */}
          <div className="flex items-center justify-between gap-2 mb-1.5">
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex flex-wrap items-center gap-1.5 shrink-0">
                {isInferredOther ? (
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 text-xs font-semibold rounded-full border ${OTHER_STYLE} shrink-0`}
                  >
                    {OTHER_LABEL}
                  </span>
                ) : (
                  linkTypes.map((t, idx) => {
                    const label = LINK_TYPE_LABELS[t] || OTHER_LABEL;
                    const badgeStyle = LINK_TYPE_STYLES[t] || OTHER_STYLE;
                    return (
                      <span
                        key={idx}
                        className={`inline-flex items-center px-2.5 py-0.5 text-xs font-semibold rounded-full border ${badgeStyle} shrink-0`}
                      >
                        {label}
                      </span>
                    );
                  })
                )}
              </div>
              <div className="flex items-center gap-1.5 text-xs text-zinc-500 dark:text-zinc-400 truncate">
                {preview?.favicon && !faviconError ? (
                  <img
                    src={preview.favicon}
                    alt=""
                    onError={() => setFaviconError(true)}
                    className="w-3.5 h-3.5 rounded-xs shrink-0"
                  />
                ) : (
                  <Globe className="w-3.5 h-3.5 shrink-0 text-zinc-400" />
                )}
                <span className="truncate font-medium">{siteName}</span>
              </div>
            </div>
            <ExternalLink className="w-4 h-4 text-zinc-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 shrink-0 transition-colors" />
          </div>

          {/* Title */}
          {isLoading ? (
            <div
              className="h-4 bg-zinc-200 dark:bg-zinc-800 rounded w-2/3 animate-pulse my-1"
              data-testid="link-preview-skeleton"
            />
          ) : (
            <h4 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100 group-hover:text-blue-600 dark:group-hover:text-blue-400 line-clamp-1 transition-colors">
              {title}
            </h4>
          )}

          {/* Description Snippet (if available) */}
          {isLoading ? (
            <div className="h-3 bg-zinc-100 dark:bg-zinc-800/60 rounded w-5/6 animate-pulse mt-1" />
          ) : description ? (
            <p className="text-xs text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-1 leading-relaxed">
              {description}
            </p>
          ) : null}
        </div>

        {/* URL footer */}
        <div className="mt-2 text-[11px] font-mono text-zinc-400 dark:text-zinc-500 truncate">
          {link.url}
        </div>
      </div>
    </a>
  );
}
