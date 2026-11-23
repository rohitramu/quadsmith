import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { ChevronRight, AlertTriangle } from "lucide-react";
import {
  CollectionBadge,
  formatProductTitle,
  getCatalogCollectionMeta,
  useDocumentMeta,
} from "@quadsmith/shared";
import {
  getHardwareCollection,
  getCollectionPath,
  type HardwareCollectionDef,
} from "../lib/hardwareCollections";
import { MediaGallery } from "../components/MediaGallery";
import { ReferenceLinksList } from "../components/SocialLinkPreviewCard";
import { SocialToolbar } from "../components/SocialToolbar";
import { SocialTabs, type SocialTabType } from "../components/SocialTabs";

function ProductDetailView({
  categoryId,
  collection,
  productId,
}: {
  categoryId: string;
  collection: HardwareCollectionDef;
  productId: string;
}) {
  const catalogMeta = getCatalogCollectionMeta(collection.id);

  const { data, isLoading, error } = useQuery(
    collection.getQuery,
    { id: productId },
    { enabled: !!productId },
  );

  const [activeSocialTab, setActiveSocialTab] = useState<SocialTabType>("reviews");
  const item = data as any;

  useDocumentMeta({
    title: item
      ? `${item.manufacturer ? item.manufacturer + " " : ""}${item.name || item.id} — Quadsmith`
      : `${collection.name} — Quadsmith`,
    description:
      item?.description ||
      `Explore ${collection.name} hardware specifications and compatibility on Quadsmith.`,
    image: item?.primaryDisplayImage || "/og-default.png",
    type: "article",
  });

  return (
    <div className="max-w-3xl mx-auto">
      <nav
        aria-label="Breadcrumb"
        className="mb-4 text-sm text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link
          to={`/components/${categoryId}`}
          className="capitalize hover:text-zinc-100 hover:underline transition-colors"
        >
          {categoryId}
        </Link>
        <ChevronRight size={14} className="text-zinc-500 shrink-0" aria-hidden="true" />
        <CollectionBadge
          collection={collection.id}
          label={collection.name}
          size="md"
          to={`/${getCollectionPath(collection)}`}
        />
        <ChevronRight size={14} className="text-zinc-500 shrink-0" aria-hidden="true" />
        <span className="text-zinc-100 font-medium truncate max-w-md" aria-current="page">
          {formatProductTitle(item?.manufacturer, item?.name || item?.id, productId)}
        </span>
      </nav>

      {isLoading ? (
        <p className="text-zinc-500">Loading details...</p>
      ) : error ? (
        <p className="text-red-500">Error: {error.message}</p>
      ) : !item ? (
        <p className="text-zinc-500">Product not found.</p>
      ) : (
        <>
          {item.isArchived && (
            <div className="mb-6 p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-300 flex items-start gap-3 text-sm">
              <AlertTriangle size={18} className="text-amber-500 shrink-0 mt-0.5" />
              <div>
                <span className="font-bold">Archived Component</span>
                <p className="mt-0.5 text-xs opacity-90 leading-relaxed">
                  This component has been archived and delisted from the public catalog. It remains
                  visible here for historical builds and external references.
                </p>
              </div>
            </div>
          )}

          <div className="mb-8">
            <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <CollectionBadge
                    collection={collection.id}
                    label={catalogMeta?.singularLabel || collection.name}
                    size="sm"
                  />
                </div>
                <h1 className="text-3xl font-bold">{item.name || item.id}</h1>
                <p className="text-xl text-zinc-400">
                  {item.manufacturer || "Unknown Manufacturer"}
                </p>
              </div>
            </div>

            <div className="mt-4">
              <SocialToolbar
                target={{
                  type: "component",
                  id: item.id || item.uuid || productId,
                  collection: collection.id,
                }}
                targetName={item.name || item.id}
                onJumpToReviews={() => {
                  setActiveSocialTab("reviews");
                  const el = document.getElementById("social-hub");
                  el?.scrollIntoView({ behavior: "smooth" });
                }}
                onJumpToQa={() => {
                  setActiveSocialTab("qa");
                  const el = document.getElementById("social-hub");
                  el?.scrollIntoView({ behavior: "smooth" });
                }}
              />
            </div>
          </div>

          {/* Key Spec Highlight Cards */}
          {collection.highlights && collection.highlights.length > 0 && (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              {collection.highlights.map((h) => (
                <div key={h.label} className="p-4 bg-zinc-900 rounded-lg border border-zinc-800">
                  <h3 className="text-sm font-semibold text-zinc-500 uppercase">{h.label}</h3>
                  <div className="text-2xl mt-1 font-medium">{h.value(item)}</div>
                </div>
              ))}
            </div>
          )}

          {/* Technical Specifications Table */}
          {collection.technicalSpecs && collection.technicalSpecs.length > 0 && (
            <div className="mt-6 p-4 bg-zinc-900 rounded-lg border border-zinc-800">
              <h3 className="text-sm font-semibold text-zinc-500 uppercase mb-3">Specifications</h3>
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm">
                {collection.technicalSpecs.map((spec) => (
                  <div
                    key={spec.label}
                    className="flex justify-between py-1.5 border-b border-zinc-800/60"
                  >
                    <dt className="text-zinc-500">{spec.label}</dt>
                    <dd className="font-medium text-zinc-200">{spec.value(item)}</dd>
                  </div>
                ))}
              </dl>
            </div>
          )}

          {item.description && (
            <div className="mt-6 p-4 bg-zinc-900 rounded-lg border border-zinc-800">
              <h3 className="text-sm font-semibold text-zinc-500 uppercase mb-1">Description</h3>
              <p className="text-zinc-300 whitespace-pre-line">{item.description}</p>
            </div>
          )}

          {/* Media & Display Image Gallery */}
          <MediaGallery
            primaryDisplayImage={item.primaryDisplayImage}
            media={item.media}
            title="Media Gallery"
          />

          {item.referenceLinks && item.referenceLinks.length > 0 && (
            <div className="mt-6">
              <h2 className="text-lg font-semibold text-zinc-100 mb-3">Reference Links</h2>
              <ReferenceLinksList links={item.referenceLinks} />
            </div>
          )}

          {/* Social Interactions: Reviews, Q&A, Collections */}
          <SocialTabs
            target={{
              type: "component",
              id: item.id || item.uuid || productId,
              collection: collection.id,
            }}
            entityName={item.name || item.id}
            activeTab={activeSocialTab}
            onTabChange={setActiveSocialTab}
          />
        </>
      )}
    </div>
  );
}

export function ProductPage() {
  const { categoryId, collectionId, productId } = useParams();
  const collection = getHardwareCollection(collectionId);

  if (!collection) {
    return (
      <div className="max-w-3xl mx-auto">
        <p className="text-zinc-500">
          Product page for {collectionId?.replace(/[-_]/g, " ")} coming soon.
        </p>
      </div>
    );
  }

  return (
    <ProductDetailView
      key={`${collection.id}-${productId}`}
      categoryId={categoryId || "hardware"}
      collection={collection}
      productId={productId || ""}
    />
  );
}
