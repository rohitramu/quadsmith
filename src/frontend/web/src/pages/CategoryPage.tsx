import { useParams, Link } from "react-router-dom";
import { Cpu } from "lucide-react";
import { HARDWARE_COLLECTIONS, getCollectionPath } from "../lib/hardwareCollections";
import { getCollectionColor } from "../lib/collectionColors";
import { CollectionIcon } from "../components/CollectionIcon";
import { useDocumentMeta } from "../hooks/useDocumentMeta";

export function CategoryPage() {
  const { categoryId } = useParams();

  const formattedCat = categoryId
    ? categoryId.charAt(0).toUpperCase() + categoryId.slice(1)
    : "Hardware";

  useDocumentMeta({
    title: `${formattedCat} Components — Quadsmith`,
    description: `Explore and compare ${formattedCat.toLowerCase()} drone components and hardware on Quadsmith.`,
  });

  const collections = categoryId === "hardware" ? HARDWARE_COLLECTIONS : [];

  return (
    <div>
      <div className="flex items-center gap-3 mb-6">
        {categoryId === "hardware" && (
          <Cpu size={32} className="text-zinc-600 dark:text-zinc-400 shrink-0" aria-hidden="true" />
        )}
        <h1 className="text-3xl font-bold capitalize">{categoryId}</h1>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {collections.map((c) => {
          const colColor = c.color || getCollectionColor(c.id);
          return (
            <Link
              key={c.id}
              to={`/${getCollectionPath(c)}`}
              className={`relative flex flex-col justify-between p-6 border border-zinc-200 dark:border-zinc-800 rounded-xl transition-all bg-zinc-50 dark:bg-zinc-900 group overflow-hidden shadow-xs hover:shadow-md ${colColor.hoverBorderClass}`}
            >
              {/* Card top trim line */}
              <div className={`absolute top-0 left-0 right-0 h-1.5 ${colColor.trimClass}`} />

              <div>
                <div className="flex items-center gap-2.5 mb-2">
                  <CollectionIcon
                    collection={c.id}
                    size={22}
                    className={`${colColor.textClass} shrink-0`}
                  />
                  <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100 group-hover:text-zinc-700 dark:group-hover:text-zinc-300 transition-colors">
                    {c.name}
                  </h2>
                </div>
                <p className="text-zinc-500 dark:text-zinc-400 text-sm leading-relaxed">
                  {c.description || `Browse all ${c.name.toLowerCase()}`}
                </p>
              </div>
            </Link>
          );
        })}
        {collections.length === 0 && (
          <p className="text-zinc-500">No collections found in this category.</p>
        )}
      </div>
    </div>
  );
}
