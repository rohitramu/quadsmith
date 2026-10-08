import { useParams, Link } from "react-router-dom";

export function CategoryPage() {
  const { categoryId } = useParams();

  // Placeholder data mapping category to collections
  const collections = categoryId === "hardware" ? [
    { id: "motors", name: "Motors" },
    { id: "frames", name: "Frames" },
    { id: "electronic-speed-controllers", name: "Electronic Speed Controllers" },
    { id: "flight_controllers", name: "Flight Controllers" },
    { id: "gps-receivers", name: "GPS Receivers" },
  ] : [];

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6 capitalize">{categoryId}</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {collections.map(c => (
          <Link key={c.id} to={`/components/${categoryId}/${c.id}`} className="block p-6 border border-zinc-200 dark:border-zinc-800 rounded-lg hover:border-blue-500 transition-colors bg-zinc-50 dark:bg-zinc-900">
            <h2 className="text-xl font-semibold">{c.name}</h2>
            <p className="text-zinc-500 dark:text-zinc-400 mt-2">Browse all {c.name.toLowerCase()}</p>
          </Link>
        ))}
        {collections.length === 0 && <p>No collections found in this category.</p>}
      </div>
    </div>
  );
}
