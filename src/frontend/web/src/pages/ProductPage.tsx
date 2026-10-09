import { useParams, Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { ReferenceLinkType } from "../gen/quadsmith/reference_link_pb";
import { ExternalLink, ChevronRight } from "lucide-react";
import {
  getHardwareCollection,
  getCollectionPath,
  type HardwareCollectionDef,
} from "../lib/hardwareCollections";
import { MediaGallery } from "../components/MediaGallery";

const LINK_TYPE_LABELS: Record<number, string> = {
  [ReferenceLinkType.PURCHASE]: "Purchase",
  [ReferenceLinkType.PRODUCT_PAGE]: "Official Product Page",
  [ReferenceLinkType.DOCUMENTATION]: "Documentation",
  [ReferenceLinkType.FORUM_POST]: "Forum Discussion",
  [ReferenceLinkType.REVIEW]: "Review",
  [ReferenceLinkType.OTHER]: "Other",
  [ReferenceLinkType.UNSPECIFIED]: "Reference Link",
};

function ProductDetailView({
  categoryId,
  collection,
  productId,
}: {
  categoryId: string;
  collection: HardwareCollectionDef;
  productId: string;
}) {
  const { data, isLoading, error } = useQuery(
    collection.getQuery,
    { id: productId },
    { enabled: !!productId },
  );

  const item = data as any;

  return (
    <div className="max-w-3xl">
      <nav
        aria-label="Breadcrumb"
        className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap"
      >
        <Link
          to={`/components/${categoryId}`}
          className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          {categoryId}
        </Link>
        <ChevronRight
          size={14}
          className="text-zinc-400 dark:text-zinc-500 shrink-0"
          aria-hidden="true"
        />
        <Link
          to={`/${getCollectionPath(collection)}`}
          className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          {collection.name}
        </Link>
        <ChevronRight
          size={14}
          className="text-zinc-400 dark:text-zinc-500 shrink-0"
          aria-hidden="true"
        />
        <span
          className="text-zinc-900 dark:text-zinc-100 font-medium truncate max-w-md"
          aria-current="page"
        >
          {item?.name || item?.id || productId}
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
          <div className="mb-8">
            <h1 className="text-3xl font-bold">{item.name || item.id}</h1>
            <p className="text-xl text-zinc-500 dark:text-zinc-400">
              {item.manufacturer || "Unknown Manufacturer"}
            </p>
          </div>

          {/* Key Spec Highlight Cards */}
          {collection.highlights && collection.highlights.length > 0 && (
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              {collection.highlights.map((h) => (
                <div
                  key={h.label}
                  className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800"
                >
                  <h3 className="text-sm font-semibold text-zinc-500 uppercase">{h.label}</h3>
                  <div className="text-2xl mt-1 font-medium">{h.value(item)}</div>
                </div>
              ))}
            </div>
          )}

          {/* Technical Specifications Table */}
          {collection.technicalSpecs && collection.technicalSpecs.length > 0 && (
            <div className="mt-6 p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
              <h3 className="text-sm font-semibold text-zinc-500 uppercase mb-3">Specifications</h3>
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-2 text-sm">
                {collection.technicalSpecs.map((spec) => (
                  <div
                    key={spec.label}
                    className="flex justify-between py-1.5 border-b border-zinc-100 dark:border-zinc-800/60"
                  >
                    <dt className="text-zinc-500">{spec.label}</dt>
                    <dd className="font-medium text-zinc-800 dark:text-zinc-200">
                      {spec.value(item)}
                    </dd>
                  </div>
                ))}
              </dl>
            </div>
          )}

          {item.description && (
            <div className="mt-6 p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
              <h3 className="text-sm font-semibold text-zinc-500 uppercase mb-1">Description</h3>
              <p className="text-zinc-700 dark:text-zinc-300 whitespace-pre-line">
                {item.description}
              </p>
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
              <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-3">
                Reference Links
              </h2>
              <div className="flex flex-col gap-2">
                {item.referenceLinks.map((link: any, idx: number) => (
                  <a
                    key={idx}
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center justify-between p-3 bg-zinc-50 dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 border border-zinc-200 dark:border-zinc-800 rounded-lg transition-colors group"
                  >
                    <div className="flex items-center gap-3">
                      <span className="px-2 py-0.5 text-xs font-medium rounded bg-zinc-200 dark:bg-zinc-800 text-zinc-800 dark:text-zinc-200 shrink-0">
                        {LINK_TYPE_LABELS[link.type] || "Link"}
                      </span>
                      <span className="text-sm text-zinc-700 dark:text-zinc-300 font-mono truncate max-w-lg">
                        {link.url}
                      </span>
                    </div>
                    <ExternalLink className="w-4 h-4 text-zinc-400 group-hover:text-zinc-600 dark:group-hover:text-zinc-200 shrink-0 ml-2" />
                  </a>
                ))}
              </div>
            </div>
          )}
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
      <div className="max-w-3xl">
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
