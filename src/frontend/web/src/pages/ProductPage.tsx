import { useParams, Link } from "react-router-dom";
import { useQuery } from "@connectrpc/connect-query";
import { getMotor } from "../gen/quadsmith/motor-MotorService_connectquery";
import { ReferenceLinkType } from "../gen/quadsmith/reference_link_pb";
import { formatStatorSize } from "../lib/format";
import { ExternalLink, ChevronRight } from "lucide-react";

const LINK_TYPE_LABELS: Record<number, string> = {
  [ReferenceLinkType.PURCHASE]: "Purchase",
  [ReferenceLinkType.PRODUCT_PAGE]: "Official Product Page",
  [ReferenceLinkType.DOCUMENTATION]: "Documentation",
  [ReferenceLinkType.FORUM_POST]: "Forum Discussion",
  [ReferenceLinkType.REVIEW]: "Review",
  [ReferenceLinkType.OTHER]: "Other",
  [ReferenceLinkType.UNSPECIFIED]: "Reference Link",
};

export function ProductPage() {
  const { categoryId, collectionId, productId } = useParams();

  const isMotors = collectionId === "motors";
  const { data, isLoading, error } = useQuery(getMotor, { id: productId }, { enabled: isMotors && !!productId });

  const m = data;

  return (
    <div className="max-w-3xl">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-zinc-500 dark:text-zinc-400 flex items-center gap-1.5 flex-wrap">
        <Link
          to={`/components/${categoryId}`}
          className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          {categoryId}
        </Link>
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" aria-hidden="true" />
        <Link
          to={`/components/${categoryId}/${collectionId}`}
          className="capitalize hover:text-zinc-900 dark:hover:text-zinc-100 hover:underline transition-colors"
        >
          {collectionId?.replace(/[-_]/g, ' ')}
        </Link>
        <ChevronRight size={14} className="text-zinc-400 dark:text-zinc-500 shrink-0" aria-hidden="true" />
        <span className="text-zinc-900 dark:text-zinc-100 font-medium truncate max-w-md" aria-current="page">
          {m?.name || m?.id || productId}
        </span>
      </nav>

      {!isMotors ? (
        <p>Product page for {collectionId?.replace(/[-_]/g, ' ')} coming soon.</p>
      ) : isLoading ? (
        <p>Loading details...</p>
      ) : error ? (
        <p className="text-red-500">Error: {error.message}</p>
      ) : !m ? (
        <p>Product not found.</p>
      ) : (
        <>
          <div className="mb-8">
        <h1 className="text-3xl font-bold">{m.name || m.id}</h1>
        <p className="text-xl text-zinc-500 dark:text-zinc-400">{m.manufacturer || "Unknown Manufacturer"}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">KV</h3>
          <p className="text-2xl mt-1">{m.kv || "N/A"}</p>
        </div>
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">Weight (g)</h3>
          <p className="text-2xl mt-1">{m.weightG ? `${m.weightG}` : "N/A"}</p>
        </div>
        <div className="p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase">Stator Size</h3>
          <p className="text-2xl mt-1 font-mono">{formatStatorSize(m.statorDiameterMm, m.statorHeightMm, "N/A")}</p>
        </div>
      </div>

      {m.description && (
        <div className="mt-6 p-4 bg-zinc-50 dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <h3 className="text-sm font-semibold text-zinc-500 uppercase mb-1">Description</h3>
          <p className="text-zinc-700 dark:text-zinc-300 whitespace-pre-line">{m.description}</p>
        </div>
      )}

      {m.referenceLinks && m.referenceLinks.length > 0 && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-zinc-100 mb-3">Reference Links</h2>
          <div className="flex flex-col gap-2">
            {m.referenceLinks.map((link, idx) => (
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
