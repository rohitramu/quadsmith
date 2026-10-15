import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import {
  Moon,
  Sun,
  Search,
  X,
  Loader2,
  Hammer,
  GalleryVertical,
  Settings,
  LogOut,
} from "lucide-react";
import { useState, useEffect, useRef } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { search } from "@quadsmith/shared/gen/quadsmith/api/search-SearchService_connectquery";
import logoDark from "../assets/quadsmith-logo-dark.svg";
import logoLight from "../assets/quadsmith-logo-light.svg";
import {
  HARDWARE_COLLECTIONS,
  SOFTWARE_COLLECTIONS,
  GEAR_COLLECTIONS,
  getCollectionPath,
} from "../lib/hardwareCollections";
import { AccountRestrictedPage } from "../pages/AccountRestrictedPage";

import { getCollectionColor } from "@quadsmith/shared";
import { CollectionBadge } from "@quadsmith/shared";
import { CollectionIcon } from "@quadsmith/shared";
import { EllipsisHint } from "@quadsmith/shared";
import { useAuth } from "../context/AuthContext";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";

export function Layout() {
  const { user, profile, logout } = useAuth();
  const [theme, setTheme] = useState(localStorage.getItem("theme") || "dark");
  const location = useLocation();
  const navigate = useNavigate();

  const [isAccountMenuOpen, setIsAccountMenuOpen] = useState(false);
  const accountMenuTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleAccountMouseEnter = () => {
    if (accountMenuTimerRef.current) {
      clearTimeout(accountMenuTimerRef.current);
      accountMenuTimerRef.current = null;
    }
    setIsAccountMenuOpen(true);
  };

  const handleAccountMouseLeave = () => {
    accountMenuTimerRef.current = setTimeout(() => {
      setIsAccountMenuOpen(false);
    }, 150);
  };

  useEffect(() => {
    return () => {
      if (accountMenuTimerRef.current) {
        clearTimeout(accountMenuTimerRef.current);
      }
    };
  }, []);

  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery.trim());
    }, 200);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const { data: searchResponse, isLoading: isSearching } = useQuery(
    search,
    { query: debouncedQuery, limit: 8 },
    { enabled: debouncedQuery.length > 0 },
  );

  const searchResults = searchResponse?.results || [];

  // Close dropdown on click outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (searchContainerRef.current && !searchContainerRef.current.contains(e.target as Node)) {
        setIsSearchOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Reset selected index when results change
  useEffect(() => {
    setSelectedIndex(-1);
  }, [searchResults]);

  const handleSelectResult = (item: { path: string; id: string }) => {
    setIsSearchOpen(false);
    setSearchQuery("");
    navigate(`/${item.path}/${item.id}`);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!isSearchOpen || searchResults.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => (prev < searchResults.length - 1 ? prev + 1 : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => (prev > 0 ? prev - 1 : searchResults.length - 1));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (selectedIndex >= 0 && selectedIndex < searchResults.length) {
        handleSelectResult(searchResults[selectedIndex]);
      }
    } else if (e.key === "Escape") {
      setIsSearchOpen(false);
    }
  };

  useEffect(() => {
    if (theme === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
    localStorage.setItem("theme", theme);
  }, [theme]);

  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "theme" && (e.newValue === "light" || e.newValue === "dark")) {
        setTheme(e.newValue);
      }
    };
    const handleCustom = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail === "light" || detail === "dark") {
        setTheme(detail);
      }
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("qs-theme-change", handleCustom);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("qs-theme-change", handleCustom);
    };
  }, []);

  const toggleTheme = () => setTheme(theme === "dark" ? "light" : "dark");

  return (
    <div className="flex flex-col h-screen w-full bg-white dark:bg-zinc-950 text-zinc-900 dark:text-zinc-50 font-sans">
      {/* Top Navbar */}
      <header className="sticky top-0 z-50 flex items-center h-16 px-4 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900">
        {/* Logo */}
        <div className="flex items-center gap-2.5 w-64">
          <Link to="/" className="flex items-center hover:opacity-90 transition-opacity">
            <img
              src={theme === "dark" ? logoDark : logoLight}
              alt="Quadsmith"
              className="h-11 w-auto"
            />
          </Link>
          {user?.activeBan && (
            <span className="px-2 py-0.5 rounded-full text-[10px] font-bold tracking-wide uppercase bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20 shrink-0">
              Suspended
            </span>
          )}
        </div>

        {/* Search Bar with Autocomplete Dropdown */}
        <div ref={searchContainerRef} className="flex-1 max-w-2xl mx-auto px-4 relative">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" size={18} />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setIsSearchOpen(true);
              }}
              onFocus={() => {
                if (searchQuery.trim().length > 0) {
                  setIsSearchOpen(true);
                }
              }}
              onKeyDown={handleKeyDown}
              placeholder="Search components..."
              className="w-full bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 rounded-full py-2 pl-10 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => {
                  setSearchQuery("");
                  setIsSearchOpen(false);
                }}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200"
              >
                <X size={15} />
              </button>
            )}
          </div>

          {/* Autocomplete Dropdown */}
          {isSearchOpen && debouncedQuery.length > 0 && (
            <div className="absolute left-4 right-4 mt-2 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl shadow-2xl overflow-hidden z-50 max-h-96 overflow-y-auto">
              {isSearching ? (
                <div className="p-4 text-center text-sm text-zinc-400 flex items-center justify-center gap-2">
                  <Loader2 size={16} className="animate-spin text-blue-500" />
                  <span>Searching catalog...</span>
                </div>
              ) : searchResults.length === 0 ? (
                <div className="p-4 text-center text-sm text-zinc-500 dark:text-zinc-400">
                  No matching components or builds found for &ldquo;
                  <span className="font-semibold">{debouncedQuery}</span>&rdquo;
                </div>
              ) : (
                <ul className="py-1 divide-y divide-zinc-100 dark:divide-zinc-800/50">
                  {searchResults.map((item, idx) => {
                    const isSelected = idx === selectedIndex;
                    const colColor = getCollectionColor(item.path || item.collectionName);
                    const details: string[] = [];
                    if (item.metadata["manufacturer"]) details.push(item.metadata["manufacturer"]);
                    if (item.metadata["cell_count_s"])
                      details.push(`${item.metadata["cell_count_s"]}S`);
                    if (item.metadata["weight_g"]) {
                      const wt = parseFloat(item.metadata["weight_g"]);
                      details.push(
                        isNaN(wt) ? `${item.metadata["weight_g"]}g` : `${wt.toFixed(2)}g`,
                      );
                    }

                    return (
                      <li key={`${item.path}-${item.id}`}>
                        <button
                          type="button"
                          onClick={() => handleSelectResult(item)}
                          className={`w-full text-left px-3.5 py-2.5 flex items-center gap-3 transition-colors ${
                            isSelected
                              ? "bg-blue-50 dark:bg-blue-950/40 text-blue-900 dark:text-blue-100"
                              : "hover:bg-zinc-50 dark:hover:bg-zinc-800/60 text-zinc-900 dark:text-zinc-100"
                          }`}
                        >
                          {item.primaryDisplayImage ? (
                            <img
                              src={item.primaryDisplayImage}
                              alt={item.name}
                              className="w-9 h-9 object-cover rounded-lg border border-zinc-200 dark:border-zinc-800 shrink-0"
                            />
                          ) : (
                            <div
                              className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200/80 dark:border-zinc-700/60 ${colColor.textClass}`}
                            >
                              <CollectionIcon
                                collection={item.path || item.collectionName}
                                size={16}
                              />
                            </div>
                          )}

                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-xs sm:text-sm truncate">
                                {item.name}
                              </span>
                              <CollectionBadge
                                collection={item.path || item.collectionName}
                                label={item.collectionName}
                                size="xs"
                              />
                            </div>

                            {details.length > 0 ? (
                              <p className="text-[11px] text-zinc-400 dark:text-zinc-500 mt-0.5 truncate">
                                {details.join(" • ")}
                              </p>
                            ) : item.description ? (
                              <p className="text-[11px] text-zinc-400 dark:text-zinc-500 mt-0.5 truncate">
                                {item.description}
                              </p>
                            ) : null}
                          </div>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 w-64">
          <button
            onClick={toggleTheme}
            aria-label="Toggle theme"
            className="p-2 rounded-full hover:bg-zinc-200 dark:hover:bg-zinc-800"
          >
            {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
          </button>

          {/* Rightmost Account Button & Hover Dropdown Menu */}
          {user && (
            <div
              className="relative"
              onMouseEnter={handleAccountMouseEnter}
              onMouseLeave={handleAccountMouseLeave}
            >
              <Link
                to="/account"
                aria-label="Account Settings"
                title={`Account Settings (@${user.id})`}
                className="flex items-center justify-center transition-transform hover:scale-105 shrink-0"
              >
                {profile?.avatarUrl ? (
                  <img
                    src={profile.avatarUrl}
                    alt={user.id}
                    className="w-8 h-8 rounded-full object-cover border border-zinc-200 dark:border-zinc-800 shadow-2xs"
                  />
                ) : (
                  <div
                    className={`w-8 h-8 rounded-full border flex items-center justify-center font-bold text-xs shadow-2xs ${getAvatarPalette(user.uuid).badgeClass}`}
                  >
                    {getInitial(user.id)}
                  </div>
                )}
              </Link>

              {/* Hover Dropdown Menu */}
              {isAccountMenuOpen && (
                <div
                  className="absolute right-0 top-full pt-1.5 z-50"
                  onMouseEnter={handleAccountMouseEnter}
                  onMouseLeave={handleAccountMouseLeave}
                >
                  <div
                    className="w-48 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-2xl shadow-xl py-1.5 text-xs font-medium"
                    role="menu"
                    aria-label="Account menu"
                  >
                    <div className="px-3.5 py-2 border-b border-zinc-100 dark:border-zinc-800/80 mb-1">
                      <Link
                        to={`/users/${user.id}`}
                        onClick={() => setIsAccountMenuOpen(false)}
                        className="font-semibold text-zinc-900 dark:text-zinc-100 hover:text-blue-600 dark:hover:text-blue-400 hover:underline transition-colors block truncate"
                      >
                        @{user.id}
                      </Link>
                      {user.email && (
                        <p className="text-[11px] text-zinc-400 dark:text-zinc-500 truncate">
                          {user.email}
                        </p>
                      )}
                    </div>

                    <Link
                      to="/account"
                      role="menuitem"
                      onClick={() => setIsAccountMenuOpen(false)}
                      className="flex items-center gap-2.5 px-3 py-2 text-zinc-700 dark:text-zinc-200 hover:bg-zinc-100 dark:hover:bg-zinc-800/80 rounded-xl mx-1 transition-colors"
                    >
                      <Settings size={15} className="text-zinc-500 dark:text-zinc-400 shrink-0" />
                      <span>Account settings</span>
                    </Link>

                    <div className="my-1 border-t border-zinc-100 dark:border-zinc-800/80" />

                    <button
                      type="button"
                      role="menuitem"
                      onClick={async () => {
                        setIsAccountMenuOpen(false);
                        await logout();
                        navigate("/login", { replace: true });
                      }}
                      className="w-[calc(100%-8px)] flex items-center gap-2.5 px-3 py-2 text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/40 rounded-xl mx-1 transition-colors text-left font-medium"
                    >
                      <LogOut size={15} className="shrink-0" />
                      <span>Log out</span>
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
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
                  {(() => {
                    const buildColor = getCollectionColor("builds");
                    return (
                      <Link
                        to="/builds"
                        className={`flex items-center justify-between px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 ${
                          location.pathname === "/builds"
                            ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400 font-medium"
                            : "text-zinc-700 dark:text-zinc-300"
                        }`}
                      >
                        <div className="flex items-center gap-2.5">
                          <GalleryVertical
                            size={16}
                            className={`${buildColor.textClass} shrink-0`}
                          />
                          <span>Build Feed</span>
                        </div>
                      </Link>
                    );
                  })()}
                </li>
                <li>
                  {(() => {
                    const forgeColor = getCollectionColor("the-forge");
                    return (
                      <Link
                        to="/builds/new"
                        className={`flex items-center justify-between px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 ${
                          location.pathname === "/builds/new"
                            ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400 font-medium"
                            : "text-zinc-700 dark:text-zinc-300"
                        }`}
                      >
                        <div className="flex items-center gap-2.5">
                          <Hammer size={16} className={`${forgeColor.textClass} shrink-0`} />
                          <span>The Forge</span>
                        </div>
                      </Link>
                    );
                  })()}
                </li>
              </ul>
            </div>

            <div className="mb-6">
              <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                Components
              </h2>
              <ul className="space-y-1">
                <li>
                  <Link
                    to="/components/hardware"
                    className={`block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 font-medium ${
                      location.pathname === "/components/hardware"
                        ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400"
                        : "text-zinc-800 dark:text-zinc-200"
                    }`}
                  >
                    Hardware
                  </Link>
                  <ul className="pl-6 mt-1 space-y-1">
                    {HARDWARE_COLLECTIONS.map((c) => {
                      const colColor = getCollectionColor(c.id);
                      return (
                        <li key={c.id}>
                          <Link
                            to={`/${getCollectionPath(c)}`}
                            className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 ${
                              location.pathname.startsWith(`/${getCollectionPath(c)}`)
                                ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400 font-medium"
                                : "text-zinc-600 dark:text-zinc-400"
                            }`}
                          >
                            <CollectionIcon
                              collection={c.id}
                              size={15}
                              className={`${colColor.textClass} shrink-0`}
                            />
                            <span>{c.name}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </li>
                <li>
                  <Link
                    to="/components/software"
                    className={`block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 font-medium ${
                      location.pathname === "/components/software"
                        ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400"
                        : "text-zinc-800 dark:text-zinc-200"
                    }`}
                  >
                    Software
                  </Link>
                  <ul className="pl-6 mt-1 space-y-1">
                    {SOFTWARE_COLLECTIONS.map((c) => {
                      const colColor = getCollectionColor(c.id);
                      return (
                        <li key={c.id}>
                          <Link
                            to={`/${getCollectionPath(c)}`}
                            className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 ${
                              location.pathname.startsWith(`/${getCollectionPath(c)}`)
                                ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400 font-medium"
                                : "text-zinc-600 dark:text-zinc-400"
                            }`}
                          >
                            <CollectionIcon
                              collection={c.id}
                              size={15}
                              className={`${colColor.textClass} shrink-0`}
                            />
                            <span>{c.name}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </li>
                <li>
                  <Link
                    to="/components/gear"
                    className={`block px-3 py-2 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 font-medium ${
                      location.pathname === "/components/gear"
                        ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400"
                        : "text-zinc-800 dark:text-zinc-200"
                    }`}
                  >
                    Gear
                  </Link>
                  <ul className="pl-6 mt-1 space-y-1">
                    {GEAR_COLLECTIONS.map((c) => {
                      const colColor = getCollectionColor(c.id);
                      return (
                        <li key={c.id}>
                          <Link
                            to={`/${getCollectionPath(c)}`}
                            className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 ${
                              location.pathname.startsWith(`/${getCollectionPath(c)}`)
                                ? "bg-zinc-200/70 dark:bg-zinc-800 text-blue-600 dark:text-blue-400 font-medium"
                                : "text-zinc-600 dark:text-zinc-400"
                            }`}
                          >
                            <CollectionIcon
                              collection={c.id}
                              size={15}
                              className={`${colColor.textClass} shrink-0`}
                            />
                            <span>{c.name}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </li>
              </ul>
            </div>
          </nav>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto">
          <div className="p-8">
            {user?.activeBan && location.pathname !== "/account" ? (
              <AccountRestrictedPage />
            ) : (
              <Outlet />
            )}
          </div>
        </main>
      </div>

      {/* Global Ellipsis Hint for all truncated text across UI */}
      <EllipsisHint />
    </div>
  );
}
