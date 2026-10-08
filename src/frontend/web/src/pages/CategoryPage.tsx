import { useParams, Link } from "react-router-dom";
import { HARDWARE_COLLECTIONS } from "../lib/hardwareCollections";

export function CategoryPage() {
  const { categoryId } = useParams();

  const collections = categoryId === "hardware" ? HARDWARE_COLLECTIONS : [];

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6 capitalize">{categoryId}</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {collections.map((c) => (
          <Link
            key={c.id}
            to={`/components/${categoryId}/${c.id}`}
            className="block p-6 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-blue-500 transition-colors bg-zinc-50 dark:bg-zinc-900 group"
          >
            <h2 className="text-xl font-semibold text-zinc-900 dark:text-zinc-100 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
              {c.name}
            </h2>
            <p className="text-zinc-500 dark:text-zinc-400 mt-2">
              Browse all {c.name.toLowerCase()}
            </p>
          </Link>
        ))}
        {collections.length === 0 && (
          <p className="text-zinc-500">
            No collections found in this category.
          </p>
        )}
      </div>
    </div>
  );
}
