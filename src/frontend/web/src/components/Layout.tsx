import { Outlet, Link, useLocation } from "react-router-dom";
import { Moon, Sun, Search, ChevronDown, ChevronRight } from "lucide-react";
import { useState, useEffect } from "react";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import { HARDWARE_COLLECTIONS } from "../lib/hardwareCollections";

export function Layout() {
  const [theme, setTheme] = useState(localStorage.getItem("theme") || "dark");
  const location = useLocation();
  const [expandedMenu, setExpandedMenu] = useState<string | null>(() => {
    if (location.pathname.includes("/components/hardware")) return "hardware";
    return null;
  });

  const toggleMenu = (menu: string) => {
    setExpandedMenu(expandedMenu === menu ? null : menu);
  };

  useEffect(() => {
    if (theme === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    localStorage.setItem("theme", theme);
  }, [theme]);

  const toggleTheme = () => setTheme(theme === "dark" ? "light" : "dark");

  return (
    <div className="flex flex-col h-screen w-full bg-white dark:bg-zinc-950 text-zinc-900 dark:text-zinc-50 font-sans">
      {/* Top Navbar */}
      <header className="sticky top-0 z-50 flex items-center h-16 px-4 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900">
        {/* Logo */}
        <Link to="/" className="flex items-center hover:opacity-90 transition-opacity w-64">
          <img
            src={theme === "dark" ? logoDark : logoLight}
            alt="Quadsmith"
            className="h-11 w-auto"
          />
        </Link>

        {/* Search Bar */}
        <div className="flex-1 max-w-2xl mx-auto px-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" size={18} />
            <input
              type="text"
              placeholder="Search components..."
              className="w-full bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-full py-2 pl-10 pr-4 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end w-64">
          <button
            onClick={toggleTheme}
            aria-label="Toggle theme"
            className="p-2 rounded-full hover:bg-zinc-200 dark:hover:bg-zinc-800"
          >
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </button>
        </div>
      </header>

      {/* Main Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <aside className="w-64 border-r border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900 flex flex-col overflow-y-auto">
          <nav className="p-4">
            <div className="mb-4">
              <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                Discover
              </h2>
              <ul className="space-y-1">
                <li>
                  <Link
                    to="/"
                    className={`block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 text-sm font-medium ${
                      location.pathname === "/"
                        ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400"
                        : "text-zinc-700 dark:text-zinc-300"
                    }`}
                  >
                    Builds Feed
                  </Link>
                </li>
              </ul>
            </div>

            <div className="mb-6">
              <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                Components
              </h2>
              <ul className="space-y-1">
                <li>
                  <div className="flex items-center justify-between rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800">
                    <Link to="/components/hardware" className="flex-1 px-3 py-2">
                      Hardware
                    </Link>
                    <button
                      onClick={() => toggleMenu("hardware")}
                      aria-label={
                        expandedMenu === "hardware"
                          ? "Collapse Hardware menu"
                          : "Expand Hardware menu"
                      }
                      className="p-2 mr-1 rounded-md hover:bg-zinc-300 dark:hover:bg-zinc-700 text-zinc-500"
                    >
                      {expandedMenu === "hardware" ? (
                        <ChevronDown size={16} />
                      ) : (
                        <ChevronRight size={16} />
                      )}
                    </button>
                  </div>
                  {expandedMenu === "hardware" && (
                    <ul className="pl-6 mt-1 space-y-1">
                      {HARDWARE_COLLECTIONS.map((c) => (
                        <li key={c.id}>
                          <Link
                            to={`/components/hardware/${c.id}`}
                            className="block px-3 py-1.5 text-sm rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-400"
                          >
                            {c.name}
                          </Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
                <li>
                  <Link
                    to="/components/software"
                    className="block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800"
                  >
                    Software
                  </Link>
                </li>
                <li>
                  <Link
                    to="/components/gear"
                    className="block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800"
                  >
                    Gear
                  </Link>
                </li>
              </ul>
            </div>
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto">
          <div className="p-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
