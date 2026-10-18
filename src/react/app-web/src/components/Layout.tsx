import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import {
  Search,
  X,
  Loader2,
  Hammer,
  GalleryVertical,
  Settings,
  LogOut,
  Menu,
  User,
} from "lucide-react";
import { useState, useEffect, useRef } from "react";
import { useQuery } from "@connectrpc/connect-query";
import { search } from "@quadsmith/shared/gen/quadsmith/api/search-SearchService_connectquery";
import logo from "../assets/quadsmith-logo.svg";
import {
  HARDWARE_COLLECTIONS,
  SOFTWARE_COLLECTIONS,
  GEAR_COLLECTIONS,
  getCollectionPath,
} from "../lib/hardwareCollections";
import { AccountRestrictedPage } from "../pages/AccountRestrictedPage";
import { UserBadge } from "./UserBadge";

import { getCollectionColor } from "@quadsmith/shared";
import { CollectionBadge } from "@quadsmith/shared";
import { CollectionIcon } from "@quadsmith/shared";
import { EllipsisHint } from "@quadsmith/shared";
import { ErrorBoundary } from "@quadsmith/shared";
import { useAuth } from "../context/AuthContext";
import { getAvatarPalette, getInitial } from "@quadsmith/shared";

export function Layout() {
  const { user, profile, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  // Mobile drawer state
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  // Close drawer when location changes
  useEffect(() => {
    setIsDrawerOpen(false);
  }, [location.pathname]);

  // Close drawer on Escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isDrawerOpen) {
        setIsDrawerOpen(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isDrawerOpen]);

  // Account menu state
  const [isAccountMenuOpen, setIsAccountMenuOpen] = useState(false);
  const accountMenuRef = useRef<HTMLDivElement>(null);

  const handleAccountClick = () => {
    setIsAccountMenuOpen((prev) => !prev);
  };

  // Close account menu when location changes
  useEffect(() => {
    setIsAccountMenuOpen(false);
  }, [location.pathname]);

  // Close account menu on Escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isAccountMenuOpen) {
        setIsAccountMenuOpen(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isAccountMenuOpen]);

  // Account menu outside click dismissal
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (accountMenuRef.current && !accountMenuRef.current.contains(e.target as Node)) {
        setIsAccountMenuOpen(false);
      }
    }
    if (isAccountMenuOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isAccountMenuOpen]);

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [isSearchFocused, setIsSearchFocused] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const searchContainerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Viewport tracking for search placeholder & focus expansion
  const [isMobileViewport, setIsMobileViewport] = useState(() => {
    if (typeof window !== "undefined") {
      return window.innerWidth < 640;
    }
    return false;
  });

  useEffect(() => {
    const handleResize = () => {
      setIsMobileViewport(window.innerWidth < 640);
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  // Global "/" keyboard shortcut to focus search
  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      if (e.key === "/" && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const target = e.target as HTMLElement | null;
        if (
          target &&
          (target.tagName === "INPUT" ||
            target.tagName === "TEXTAREA" ||
            target.tagName === "SELECT" ||
            target.isContentEditable)
        ) {
          return;
        }
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", handleGlobalKeyDown);
    return () => window.removeEventListener("keydown", handleGlobalKeyDown);
  }, []);

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

  // Close search dropdown on click outside
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
      searchInputRef.current?.blur();
    }
  };

  const navigationContent = (
    <nav className="p-4" aria-label="Sidebar">
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
                  className={`flex items-center justify-between px-3 py-2 rounded-md hover:bg-zinc-800 ${
                    location.pathname === "/builds"
                      ? "bg-zinc-800 text-blue-400 font-medium"
                      : "text-zinc-300"
                  }`}
                >
                  <div className="flex items-center gap-2.5">
                    <GalleryVertical size={16} className={`${buildColor.textClass} shrink-0`} />
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
                  className={`flex items-center justify-between px-3 py-2 rounded-md hover:bg-zinc-800 ${
                    location.pathname === "/builds/new"
                      ? "bg-zinc-800 text-blue-400 font-medium"
                      : "text-zinc-300"
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
              className={`block px-3 py-2 rounded-md hover:bg-zinc-800 font-medium ${
                location.pathname === "/components/hardware"
                  ? "bg-zinc-800 text-blue-400"
                  : "text-zinc-200"
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
                      className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-800 ${
                        location.pathname.startsWith(`/${getCollectionPath(c)}`)
                          ? "bg-zinc-800 text-blue-400 font-medium"
                          : "text-zinc-400"
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
              className={`block px-3 py-2 rounded-md hover:bg-zinc-800 font-medium ${
                location.pathname === "/components/software"
                  ? "bg-zinc-800 text-blue-400"
                  : "text-zinc-200"
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
                      className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-800 ${
                        location.pathname.startsWith(`/${getCollectionPath(c)}`)
                          ? "bg-zinc-800 text-blue-400 font-medium"
                          : "text-zinc-400"
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
              className={`block px-3 py-2 rounded-md hover:bg-zinc-800 font-medium ${
                location.pathname === "/components/gear"
                  ? "bg-zinc-800 text-blue-400"
                  : "text-zinc-200"
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
                      className={`flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-zinc-800 ${
                        location.pathname.startsWith(`/${getCollectionPath(c)}`)
                          ? "bg-zinc-800 text-blue-400 font-medium"
                          : "text-zinc-400"
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
  );

  return (
    <div className="flex flex-col h-screen w-full min-w-[320px] bg-zinc-950 text-zinc-50 font-sans">
      {/* Top Navbar */}
      <header className="sticky top-0 z-40 flex items-center h-14 sm:h-16 px-3 sm:px-4 border-b border-zinc-800 bg-zinc-900">
        <div className="flex items-center justify-between w-full">
          {/* Logo & Drawer trigger */}
          <div
            className={`flex items-center gap-2 sm:gap-2.5 shrink-0 w-auto md:w-64 ${
              isSearchFocused && isMobileViewport ? "hidden sm:flex" : "flex"
            }`}
          >
            <button
              type="button"
              onClick={() => setIsDrawerOpen(true)}
              aria-label="Open navigation menu"
              className="md:hidden p-1.5 -ml-1 text-zinc-400 hover:text-zinc-100 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer"
            >
              <Menu size={20} />
            </button>
            <Link to="/" className="flex items-center hover:opacity-90 transition-opacity">
              <img src={logo} alt="Quadsmith" className="h-9 sm:h-11 w-auto" />
            </Link>
            {user?.activeBan && (
              <span className="px-2 py-0.5 rounded-full text-[10px] font-bold tracking-wide uppercase bg-rose-500/10 text-rose-400 border border-rose-500/20 shrink-0">
                Suspended
              </span>
            )}
          </div>

          {/* Search Bar with Autocomplete Dropdown */}
          <div
            ref={searchContainerRef}
            className={`flex-1 min-w-0 mx-1 sm:mx-4 relative ${
              isSearchFocused && isMobileViewport ? "w-full max-w-none" : "max-w-md sm:max-w-xl"
            }`}
          >
            <div className="relative flex items-center">
              <Search
                className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400 pointer-events-none"
                size={16}
              />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setIsSearchOpen(true);
                }}
                onFocus={() => {
                  setIsSearchFocused(true);
                  if (searchQuery.trim().length > 0) {
                    setIsSearchOpen(true);
                  }
                }}
                onBlur={() => {
                  setIsSearchFocused(false);
                }}
                onKeyDown={handleKeyDown}
                placeholder={isMobileViewport ? "Search" : "Search builds, components, users..."}
                className="w-full bg-zinc-950 border border-zinc-800 rounded-full py-1.5 sm:py-2 pl-9 sm:pl-10 pr-14 sm:pr-16 text-xs sm:text-sm focus:outline-hidden focus:ring-2 focus:ring-emerald-500 transition-all"
              />
              <div className="absolute right-2.5 top-1/2 -translate-y-1/2 flex items-center gap-1">
                {searchQuery ? (
                  <button
                    type="button"
                    onClick={() => {
                      setSearchQuery("");
                      setIsSearchOpen(false);
                      searchInputRef.current?.focus();
                    }}
                    aria-label="Clear search"
                    className="p-1 text-zinc-400 hover:text-zinc-200 cursor-pointer"
                  >
                    <X size={14} />
                  </button>
                ) : (
                  <kbd className="hidden sm:inline-flex items-center justify-center px-1.5 py-0.5 text-[10px] font-mono font-medium text-zinc-500 bg-zinc-800 border border-zinc-700 rounded shadow-2xs pointer-events-none select-none">
                    /
                  </kbd>
                )}
                {isSearchFocused && isMobileViewport && (
                  <button
                    type="button"
                    onMouseDown={(e) => {
                      e.preventDefault();
                      setIsSearchFocused(false);
                      setIsSearchOpen(false);
                      searchInputRef.current?.blur();
                    }}
                    aria-label="Close search"
                    className="p-1 text-zinc-400 hover:text-zinc-200 sm:hidden cursor-pointer"
                  >
                    <X size={15} />
                  </button>
                )}
              </div>
            </div>

            {/* Autocomplete Dropdown */}
            {isSearchOpen && debouncedQuery.length > 0 && (
              <div className="fixed inset-x-2 top-14 sm:absolute sm:inset-x-0 sm:top-full mt-2 bg-zinc-900 border border-zinc-800 rounded-2xl shadow-2xl overflow-hidden z-50 max-h-96 overflow-y-auto">
                {isSearching ? (
                  <div className="p-4 text-center text-sm text-zinc-400 flex items-center justify-center gap-2">
                    <Loader2 size={16} className="animate-spin text-emerald-500" />
                    <span>Searching catalog...</span>
                  </div>
                ) : searchResults.length === 0 ? (
                  <div className="p-4 text-center text-sm text-zinc-400">
                    No matching components or builds found for &ldquo;
                    <span className="font-semibold">{debouncedQuery}</span>&rdquo;
                  </div>
                ) : (
                  <ul className="py-1 divide-y divide-zinc-800/50">
                    {searchResults.map((item, idx) => {
                      const isSelected = idx === selectedIndex;
                      const colColor = getCollectionColor(item.path || item.collectionName);
                      const details: string[] = [];
                      if (item.metadata["manufacturer"])
                        details.push(item.metadata["manufacturer"]);
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
                                ? "bg-emerald-950/40 text-emerald-100"
                                : "hover:bg-zinc-800/60 text-zinc-100"
                            }`}
                          >
                            {item.primaryDisplayImage ? (
                              <img
                                src={item.primaryDisplayImage}
                                alt={item.name}
                                className="w-9 h-9 object-cover rounded-lg border border-zinc-800 shrink-0"
                              />
                            ) : (
                              <div
                                className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 bg-zinc-800 border border-zinc-700/60 ${colColor.textClass}`}
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
                                <p className="text-[11px] text-zinc-500 mt-0.5 truncate">
                                  {details.join(" • ")}
                                </p>
                              ) : item.description ? (
                                <p className="text-[11px] text-zinc-500 mt-0.5 truncate">
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

          {/* Actions: User Avatar */}
          <div
            className={`flex items-center justify-end gap-2 sm:gap-3 shrink-0 w-auto md:w-64 ${
              isSearchFocused && isMobileViewport ? "hidden sm:flex" : "flex"
            }`}
          >
            {/* Rightmost Account Avatar Button & Dropdown Menu */}
            {user && (
              <div ref={accountMenuRef} className="relative">
                <button
                  type="button"
                  aria-label="Account Settings"
                  aria-haspopup="menu"
                  aria-expanded={isAccountMenuOpen}
                  title={`Account Settings (@${user.id})`}
                  onClick={handleAccountClick}
                  className="flex items-center justify-center transition-transform hover:scale-105 shrink-0 rounded-full focus:outline-hidden focus:ring-2 focus:ring-emerald-500 cursor-pointer"
                >
                  {profile?.avatarUrl ? (
                    <img
                      src={profile.avatarUrl}
                      alt={user.id}
                      className="w-8 h-8 rounded-full object-cover border border-zinc-800 shadow-2xs"
                    />
                  ) : (
                    <div
                      className={`w-8 h-8 rounded-full border flex items-center justify-center font-bold text-xs shadow-2xs ${getAvatarPalette(user.uuid).badgeClass}`}
                    >
                      {getInitial(user.id)}
                    </div>
                  )}
                </button>

                {/* Account Dropdown Menu */}
                {isAccountMenuOpen && (
                  <div className="absolute right-0 top-full pt-1.5 z-50">
                    <div
                      className="w-48 bg-zinc-900 border border-zinc-800 rounded-2xl shadow-xl py-1.5 text-xs font-medium"
                      role="menu"
                      aria-label="Account menu"
                    >
                      <div className="px-3.5 py-2.5 border-b border-zinc-800/80 mb-1">
                        <div className="flex items-center">
                          <UserBadge username={user.id} size="sm" withHoverCard={false} to={null} />
                        </div>
                        {user.email && (
                          <p className="text-[11px] text-zinc-500 truncate mt-1.5">{user.email}</p>
                        )}
                      </div>

                      <Link
                        to={`/users/${user.id}`}
                        role="menuitem"
                        onClick={() => setIsAccountMenuOpen(false)}
                        className="flex items-center gap-2.5 px-3 py-2 text-zinc-200 hover:bg-zinc-800/80 rounded-xl mx-1 transition-colors"
                      >
                        <User size={15} className="text-zinc-400 shrink-0" />
                        <span>Public Profile</span>
                      </Link>

                      <Link
                        to="/account"
                        role="menuitem"
                        onClick={() => setIsAccountMenuOpen(false)}
                        className="flex items-center gap-2.5 px-3 py-2 text-zinc-200 hover:bg-zinc-800/80 rounded-xl mx-1 transition-colors"
                      >
                        <Settings size={15} className="text-zinc-400 shrink-0" />
                        <span>Account Settings</span>
                      </Link>

                      <div className="my-1 border-t border-zinc-800/80" />

                      <button
                        type="button"
                        role="menuitem"
                        onClick={async () => {
                          setIsAccountMenuOpen(false);
                          await logout();
                          navigate("/login", { replace: true });
                        }}
                        className="w-[calc(100%-8px)] flex items-center gap-2.5 px-3 py-2 text-rose-400 hover:bg-rose-950/40 rounded-xl mx-1 transition-colors text-left font-medium cursor-pointer"
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
        </div>
      </header>

      {/* Mobile Slide-Over Drawer & Backdrop Overlay */}
      {isDrawerOpen && (
        <div className="fixed inset-0 z-50 md:hidden flex">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/50 transition-opacity backdrop-blur-xs"
            onClick={() => setIsDrawerOpen(false)}
            aria-hidden="true"
          />

          {/* Drawer content */}
          <div
            className="relative flex-1 flex flex-col max-w-[280px] w-full bg-zinc-900 border-r border-zinc-800 shadow-2xl z-10 overflow-y-auto"
            role="dialog"
            aria-modal="true"
            aria-label="Navigation drawer"
          >
            {/* Drawer Header */}
            <div className="flex items-center justify-between px-4 h-14 border-b border-zinc-800">
              <Link
                to="/"
                onClick={() => setIsDrawerOpen(false)}
                className="flex items-center hover:opacity-90 transition-opacity"
              >
                <img src={logo} alt="Quadsmith" className="h-8 w-auto" />
              </Link>
              <button
                type="button"
                onClick={() => setIsDrawerOpen(false)}
                aria-label="Close navigation menu"
                className="p-1.5 text-zinc-500 hover:text-zinc-300 rounded-lg hover:bg-zinc-800 cursor-pointer"
              >
                <X size={18} />
              </button>
            </div>

            {/* Navigation links inside drawer */}
            {navigationContent}
          </div>
        </div>
      )}

      {/* Main Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Desktop Persistent Sidebar */}
        <aside className="hidden md:flex md:w-64 shrink-0 border-r border-zinc-800 bg-zinc-900 flex-col overflow-y-auto">
          {navigationContent}
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 overflow-y-auto min-w-0">
          <div className="p-4 sm:p-6 lg:p-8">
            {user?.activeBan && location.pathname !== "/account" ? (
              <AccountRestrictedPage />
            ) : (
              <ErrorBoundary
                variant="page"
                title="Page failed to render"
                resetKeys={[location.pathname]}
              >
                <Outlet />
              </ErrorBoundary>
            )}
          </div>
        </main>
      </div>

      {/* Global Ellipsis Hint for all truncated text across UI */}
      <EllipsisHint />
    </div>
  );
}
