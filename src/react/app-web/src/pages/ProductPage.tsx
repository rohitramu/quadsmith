import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { ChevronRight, Edit, Trash2, AlertTriangle, Loader2, Copy } from "lucide-react";
import {
  getHardwareCollection,
  getCollectionPath,
  type HardwareCollectionDef,
} from "../lib/hardwareCollections";
import { getComponentServiceDef } from "../lib/componentServices";
import { useAuth } from "../context/AuthContext";
import { ComponentFormModal } from "../components/ComponentFormModal";
import { CollectionBadge } from "@quadsmith/shared";
import { MediaGallery } from "../components/MediaGallery";
import { UserBadge } from "../components/UserBadge";
import { ReferenceLinksList } from "../components/SocialLinkPreviewCard";
import { SocialToolbar } from "../components/SocialToolbar";
import { SocialTabs, type SocialTabType } from "../components/SocialTabs";
import { useDocumentMeta } from "@quadsmith/shared";
import { formatProductTitle } from "@quadsmith/shared";

function ProductDetailView({
  categoryId,
  collection,
  productId,
}: {
  categoryId: string;
  collection: HardwareCollectionDef;
  productId: string;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user, isAuthenticated } = useAuth();
  const srvDef = getComponentServiceDef(collection.id);

  const { data, isLoading, error, refetch } = useQuery(
    collection.getQuery,
    { id: productId },
    { enabled: !!productId },
  );

  const [activeSocialTab, setActiveSocialTab] = useState<SocialTabType>("reviews");
  const [isCreateSimilarOpen, setIsCreateSimilarOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const { mutateAsync: mutateDelete } = useMutation(srvDef?.deleteMethod);

  const item = data as any;
  const isCreator =
    isAuthenticated &&
    user &&
    (user.uuid === item?.userUuid || user.id === item?.authorUsername || user.role === "ADMIN");

  const handleDelete = async () => {
    if (!srvDef?.deleteMethod) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      const res: any = await mutateDelete({ id: item.id || item.uuid || productId });
      await queryClient.invalidateQueries();
      setIsDeleteConfirmOpen(false);
      if (res?.isArchived) {
        await refetch();
      } else {
        navigate(`/${getCollectionPath(collection)}`);
      }
    } catch (err: unknown) {
      setDeleteError(err instanceof Error ? err.message : "Failed to delete component");
    } finally {
      setIsDeleting(false);
    }
  };

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
        <CollectionBadge
          collection={collection.id}
          label={collection.name}
          size="md"
          to={`/${getCollectionPath(collection)}`}
        />
        <ChevronRight
          size={14}
          className="text-zinc-400 dark:text-zinc-500 shrink-0"
          aria-hidden="true"
        />
        <span
          className="text-zinc-900 dark:text-zinc-100 font-medium truncate max-w-md"
          aria-current="page"
        >
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
            <div className="mb-6 p-4 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300 flex items-start gap-3 text-sm">
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
                    label={srvDef?.singularLabel || collection.name}
                    size="sm"
                  />
                  {item.authorUsername && (
                    <UserBadge
                      username={item.authorUsername}
                      prefix="Added by "
                      size="sm"
                      isAuthor={true}
                    />
                  )}
                </div>
                <h1 className="text-3xl font-bold">{item.name || item.id}</h1>
                <p className="text-xl text-zinc-500 dark:text-zinc-400">
                  {item.manufacturer || "Unknown Manufacturer"}
                </p>
              </div>

              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button"
                  onClick={() => setIsCreateSimilarOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-xs font-medium transition-colors cursor-pointer"
                >
                  <Copy size={14} className="text-zinc-500" />
                  <span>Create Similar</span>
                </button>
                {isCreator && (
                  <>
                    <button
                      type="button"
                      onClick={() => setIsEditOpen(true)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-xs font-medium transition-colors cursor-pointer"
                    >
                      <Edit size={14} className="text-zinc-500" />
                      <span>Edit Component</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setIsDeleteConfirmOpen(true)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-red-200 dark:border-red-900/50 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 text-xs font-medium transition-colors cursor-pointer"
                    >
                      <Trash2 size={14} />
                      <span>Delete</span>
                    </button>
                  </>
                )}
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

          {/* Create Similar Component Modal */}
          <ComponentFormModal
            isOpen={isCreateSimilarOpen}
            onClose={() => setIsCreateSimilarOpen(false)}
            collectionId={collection.id}
            initialData={item}
            isClone={true}
            onSuccess={(newItem) => {
              if (newItem?.id || newItem?.uuid) {
                navigate(
                  `/components/${categoryId}/${collection.id}/${newItem.id || newItem.uuid}`,
                );
              }
            }}
          />

          {/* Edit Component Modal */}
          <ComponentFormModal
            isOpen={isEditOpen}
            onClose={() => setIsEditOpen(false)}
            collectionId={collection.id}
            initialData={item}
            onSuccess={() => {
              refetch();
            }}
          />

          {/* Delete Confirmation Modal */}
          {isDeleteConfirmOpen && (
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="delete-confirm-title"
              className="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4 animate-in fade-in duration-150"
            >
              <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl w-full max-w-md p-6 shadow-2xl space-y-4">
                <div className="w-10 h-10 rounded-full bg-amber-100 dark:bg-amber-950 text-amber-600 flex items-center justify-center">
                  <AlertTriangle size={20} />
                </div>

                <div>
                  <h3 id="delete-confirm-title" className="text-lg font-bold">
                    Delete or Archive Component
                  </h3>
                  <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 leading-relaxed">
                    If this component is referenced by existing builds, reviews, or collections, it
                    will be <strong>archived and delisted</strong> from catalog searches to prevent
                    breaking other pilots' builds. If unreferenced, it will be permanently deleted.
                  </p>
                </div>

                {deleteError && (
                  <div className="p-2.5 rounded-lg bg-red-500/10 border border-red-500/20 text-red-600 text-xs">
                    {deleteError}
                  </div>
                )}

                <div className="flex items-center justify-end gap-2 pt-2">
                  <button
                    type="button"
                    disabled={isDeleting}
                    onClick={() => setIsDeleteConfirmOpen(false)}
                    className="px-4 py-2 rounded-lg border border-zinc-300 dark:border-zinc-700 text-xs font-medium hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={isDeleting}
                    onClick={handleDelete}
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-red-600 hover:bg-red-700 text-white text-xs font-semibold shadow-xs transition-colors cursor-pointer disabled:opacity-50"
                  >
                    {isDeleting && <Loader2 size={12} className="animate-spin" />}
                    Confirm Delete
                  </button>
                </div>
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
