import React from "react";
import { MediaType, type Media } from "../gen/quadsmith/media_pb";
import { getYouTubeEmbedUrl } from "../lib/format";
import { ExternalLink, Film, Image as ImageIcon } from "lucide-react";

export interface MediaGalleryProps {
  primaryDisplayImage?: string;
  media?: Media[];
  title?: string;
}

export function MediaGallery({ primaryDisplayImage, media = [], title }: MediaGalleryProps) {
  const hasPrimaryImage = Boolean(primaryDisplayImage);
  const hasMedia = media && media.length > 0;

  if (!hasPrimaryImage && !hasMedia) {
    return null;
  }

  return (
    <section aria-label="Media Gallery" className="mt-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-bold text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
          <span>{title || "Media Gallery"}</span>
        </h2>
      </div>

      {/* Primary Display Image */}
      {hasPrimaryImage && (
        <div className="mb-6 rounded-2xl overflow-hidden border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-2 sm:p-4 flex items-center justify-center shadow-xs">
          <img
            src={primaryDisplayImage}
            alt="Primary Display"
            className="max-h-96 w-full object-contain rounded-xl"
            loading="eager"
          />
        </div>
      )}

      {/* Media Array (Images and YouTube Links) */}
      {hasMedia && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {media.map((item, index) => {
            const isYoutube =
              item.type === MediaType.YOUTUBE ||
              (item.url && (item.url.includes("youtube.com") || item.url.includes("youtu.be")));
            const embedUrl = isYoutube ? getYouTubeEmbedUrl(item.url) : null;

            return (
              <div
                key={index}
                className="flex flex-col rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 overflow-hidden shadow-xs group"
              >
                {/* Media Container */}
                {isYoutube && embedUrl ? (
                  <div className="relative aspect-video w-full bg-black">
                    <iframe
                      src={embedUrl}
                      title={item.title || `Media Video ${index + 1}`}
                      allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                      allowFullScreen
                      className="w-full h-full border-0"
                    />
                  </div>
                ) : isYoutube ? (
                  <div className="relative aspect-video w-full bg-zinc-900 flex flex-col items-center justify-center p-4 text-center">
                    <Film className="w-12 h-12 text-red-500 mb-2" />
                    <span className="text-sm font-medium text-white">
                      {item.title || "YouTube Video"}
                    </span>
                    <a
                      href={item.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors"
                    >
                      <span>Watch on YouTube</span>
                      <ExternalLink size={12} />
                    </a>
                  </div>
                ) : (
                  <div className="relative aspect-video w-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center overflow-hidden">
                    <img
                      src={item.url}
                      alt={item.title || `Media Item ${index + 1}`}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                      loading="lazy"
                    />
                  </div>
                )}

                {/* Metadata & Actions */}
                <div className="p-3.5 flex flex-col justify-between flex-1 gap-2">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium rounded-full bg-zinc-200 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300">
                        {isYoutube ? (
                          <Film size={11} className="text-red-500" />
                        ) : (
                          <ImageIcon size={11} />
                        )}
                        <span>{isYoutube ? "YouTube" : "Image"}</span>
                      </span>
                      {item.title && (
                        <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100 truncate">
                          {item.title}
                        </h3>
                      )}
                    </div>
                    {item.description && (
                      <p className="text-xs text-zinc-600 dark:text-zinc-400 line-clamp-2">
                        {item.description}
                      </p>
                    )}
                  </div>

                  <a
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1 self-start font-medium"
                  >
                    <span>{isYoutube ? "Open in YouTube" : "View Full Image"}</span>
                    <ExternalLink size={12} />
                  </a>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
