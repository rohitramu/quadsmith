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
  [ReferenceLinkType.OTHER]: "Other",
  [ReferenceLinkType.UNSPECIFIED]: "Reference Link",
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
  [ReferenceLinkType.OTHER]:
    "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 border-zinc-200 dark:border-zinc-700",
  [ReferenceLinkType.UNSPECIFIED]:
    "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300 border-zinc-200 dark:border-zinc-700",
};

export interface SocialLinkPreviewCardProps {
  link: ReferenceLink | { type: number; url: string };
  className?: string;
}

export function SocialLinkPreviewCard({ link, className = "" }: SocialLinkPreviewCardProps) {
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

  const typeLabel = LINK_TYPE_LABELS[link.type] || "Link";
  const badgeStyle = LINK_TYPE_STYLES[link.type] || LINK_TYPE_STYLES[ReferenceLinkType.OTHER];

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
          {/* Header Row: Badge, Site Favicon/Host, External Icon */}
          <div className="flex items-center justify-between gap-2 mb-1.5">
            <div className="flex items-center gap-2 min-w-0">
              <span
                className={`px-2 py-0.5 text-xs font-semibold rounded-md border ${badgeStyle} shrink-0`}
              >
                {typeLabel}
              </span>
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
