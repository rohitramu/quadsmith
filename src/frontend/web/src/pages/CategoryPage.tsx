import { useParams, Link } from "react-router-dom";
import { HARDWARE_COLLECTIONS, getCollectionPath } from "../lib/hardwareCollections";
import { getCollectionColor } from "../lib/collectionColors";
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
      <h1 className="text-3xl font-bold mb-6 capitalize">{categoryId}</h1>
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
                <div className="mb-2">
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
