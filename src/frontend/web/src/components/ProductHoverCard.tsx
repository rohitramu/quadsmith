import { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import {
  getHardwareCollection,
  getCollectionPath,
  getCollectionColor,
} from "../lib/hardwareCollections";
import { CollectionBadge } from "./CollectionBadge";
import { CollectionIcon } from "./CollectionIcon";
import { ExternalLink } from "lucide-react";

export interface ProductHoverCardProps {
  collectionId: string;
  item?: any;
  productId?: string;
  children: React.ReactNode;
  className?: string;
  as?: React.ElementType;
}

export function ProductHoverCard({
  collectionId,
  item: propItem,
  productId,
  children,
  className = "",
  as,
}: ProductHoverCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [coords, setCoords] = useState<{ top: number; left: number; placeAbove: boolean }>({
    top: 0,
    left: 0,
    placeAbove: false,
  });

  const triggerRef = useRef<HTMLElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const lastMousePosRef = useRef<{ clientX: number; clientY: number } | null>(null);
  const openTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const collection = getHardwareCollection(collectionId);
  const colColor = collection?.color || getCollectionColor(collectionId);

  // If item wasn't passed directly, fetch it using collection.getQuery
  const shouldFetch = !propItem && !!productId && !!collection?.getQuery;
  const { data: fetchedItem } = useQuery(
    collection?.getQuery,
    { id: productId },
    { enabled: shouldFetch, staleTime: 60_000 },
  );

  const product = propItem || fetchedItem;

  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const cardWidth = 320;
    const estimatedHeight = 240;

    // Viewport bounds checking
    const spaceBelow = window.innerHeight - rect.bottom;
    const placeAbove = spaceBelow < estimatedHeight && rect.top > estimatedHeight;

    let left = rect.left;
    if (as === "tr" && lastMousePosRef.current) {
      left = lastMousePosRef.current.clientX - 40;
    }

    if (left + cardWidth > window.innerWidth - 16) {
      left = window.innerWidth - cardWidth - 16;
    }
    if (left < 16) left = 16;

    const top = placeAbove ? rect.top - 8 : rect.bottom + 8;

    setCoords({
      top: top + window.scrollY,
      left: left + window.scrollX,
      placeAbove,
    });
  }, [as]);

  const handleMouseEnter = (e: React.MouseEvent) => {
    lastMousePosRef.current = { clientX: e.clientX, clientY: e.clientY };
    if (closeTimerRef.current) {
      clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
    openTimerRef.current = setTimeout(() => {
      updatePosition();
      setIsOpen(true);
    }, 200); // 200ms hover intent delay
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    lastMousePosRef.current = { clientX: e.clientX, clientY: e.clientY };
  };

  const handleMouseLeave = () => {
    if (openTimerRef.current) {
      clearTimeout(openTimerRef.current);
      openTimerRef.current = null;
    }
    closeTimerRef.current = setTimeout(() => {
      setIsOpen(false);
    }, 150); // 150ms grace period to allow cursor entering card
  };

  useEffect(() => {
    return () => {
      if (openTimerRef.current) clearTimeout(openTimerRef.current);
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  const targetPath = collection
    ? getCollectionPath(collection)
    : `components/hardware/${collectionId}`;
  const productUrl = `/${targetPath}/${product?.id || product?.uuid || productId}`;

  const Component = (as || "div") as React.ElementType;
  const combinedClassName = as ? className : `inline-block ${className}`.trim();

  return (
    <Component
      ref={triggerRef}
      onMouseEnter={handleMouseEnter}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      className={combinedClassName}
      data-testid="product-hover-trigger"
    >
      {children}

      {isOpen &&
        product &&
        createPortal(
          <div
            ref={cardRef}
            onMouseEnter={() => {
              if (closeTimerRef.current) {
                clearTimeout(closeTimerRef.current);
                closeTimerRef.current = null;
              }
            }}
            onMouseLeave={handleMouseLeave}
            style={{
              position: "absolute",
              top: `${coords.top}px`,
              left: `${coords.left}px`,
              transform: coords.placeAbove ? "translateY(-100%)" : "none",
              zIndex: 9999,
            }}
            className="w-80 p-3.5 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-xl shadow-xl transition-opacity animate-in fade-in zoom-in-95 duration-150 pointer-events-auto"
            data-testid="product-hover-card"
          >
            {/* Header: Thumbnail + Title + Badge */}
            <div className="flex items-start gap-3">
              <div
                className={`w-12 h-12 rounded-lg shrink-0 flex items-center justify-center overflow-hidden border border-zinc-200/80 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/80 ${colColor.textClass}`}
              >
                {product.primaryDisplayImage ? (
                  <img
                    src={product.primaryDisplayImage}
                    alt=""
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <CollectionIcon collection={collectionId} size={22} />
                )}
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5 mb-0.5">
                  <CollectionBadge collection={collectionId} size="xs" />
                  {product.manufacturer && (
                    <span className="text-[11px] font-semibold text-zinc-400 dark:text-zinc-500 uppercase truncate">
                      {product.manufacturer}
                    </span>
                  )}
                </div>
                <h4 className="text-sm font-bold text-zinc-900 dark:text-zinc-100 truncate">
                  {product.name || product.id}
                </h4>
              </div>
            </div>

            {/* Spec Highlights Grid */}
            {collection?.highlights && collection.highlights.length > 0 && (
              <div className="grid grid-cols-2 gap-1.5 mt-3 pt-2.5 border-t border-zinc-100 dark:border-zinc-800/80">
                {collection.highlights.slice(0, 4).map((h) => {
                  let val = h.value(product);
                  if (val == null || val === "" || val === "-") val = "N/A";
                  return (
                    <div
                      key={h.label}
                      className="bg-zinc-50 dark:bg-zinc-800/50 p-1.5 rounded-md border border-zinc-100 dark:border-zinc-800/60"
                    >
                      <span className="text-[10px] uppercase font-semibold text-zinc-400 dark:text-zinc-500 block truncate">
                        {h.label}
                      </span>
                      <span className="text-xs font-semibold text-zinc-800 dark:text-zinc-200 truncate block">
                        {val}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Description Clamped */}
            {product.description && (
              <p className="text-xs text-zinc-500 dark:text-zinc-400 line-clamp-2 mt-2 leading-relaxed">
                {product.description}
              </p>
            )}

            {/* Footer Direct Link */}
            <div className="mt-2.5 pt-2 border-t border-zinc-100 dark:border-zinc-800 flex items-center justify-end">
              <Link
                to={productUrl}
                className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 dark:text-blue-400 hover:underline"
              >
                <span>Details</span>
                <ExternalLink size={12} />
              </Link>
            </div>
          </div>,
          document.body,
        )}
    </Component>
  );
}
