import { Link } from "react-router-dom";

export function HomePage() {
  return (
    <div className="max-w-4xl">
      <h1 className="text-4xl font-extrabold tracking-tight mb-6">Welcome to Quadsmith</h1>
      <p className="text-lg text-zinc-600 dark:text-zinc-400 mb-8 leading-relaxed">
        Quadsmith is the ultimate hardware data sourcing and component browser for FPV drone
        builders. Whether you are planning a new freestyle rig, a cinematic cruiser, or a micro
        whoop, you can explore our comprehensive database of motors, frames, flight controllers, and
        more to find the perfect parts for your next build.
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-8">
        <Link
          to="/components/hardware"
          className="block p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:border-blue-500 transition-colors"
        >
          <h2 className="text-xl font-bold mb-2">Browse Hardware</h2>
          <p className="text-zinc-500 dark:text-zinc-400">
            Explore our extensive catalog of FPV drone components, complete with detailed
            specifications and metrics.
          </p>
        </Link>
        <div className="block p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 opacity-60">
          <h2 className="text-xl font-bold mb-2">Build Planner (Coming Soon)</h2>
          <p className="text-zinc-500 dark:text-zinc-400">
            Design your dream drone and let Quadsmith automatically check for component
            compatibility and estimated performance.
          </p>
        </div>
      </div>
    </div>
  );
}
