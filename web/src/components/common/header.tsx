import { useState, useEffect, useRef } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Menu, X, LogOut, ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ModeToggle } from "@/components/common/mode-toggle";
import { LanguageSwitcher } from "@/components/common/language-switcher";
import { useAuth } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import { ROUTES, MENU_GROUPS } from "@/config/routes";
import "./header.css";

type AnimationDirection = "none" | "left" | "right";

export function Header() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [expandedMobileGroup, setExpandedMobileGroup] = useState<string | null>(
    null
  );
  const [activeMenuGroup, setActiveMenuGroup] = useState<string | null>(null);
  const [animationDirection, setAnimationDirection] =
    useState<AnimationDirection>("none");
  const previousMenuGroup = useRef<string | null>(null);
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { logout, isLoading, isAuthenticated } = useAuth();

  const handleLogout = async () => {
    try {
      await logout();
      setTimeout(() => {
        navigate(ROUTES.LOGIN.path, { replace: true });
      }, 100);
    } catch (err) {
      console.error("Logout failed:", err);
    }
  };

  // Close dropdown when clicking elsewhere or leaving header area
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (
        !target.closest(".menu-trigger") &&
        !target.closest(".dropdown-container")
      ) {
        setActiveMenuGroup(null);
      }
    };

    if (activeMenuGroup) {
      document.addEventListener("click", handleClickOutside);
      return () => {
        document.removeEventListener("click", handleClickOutside);
      };
    }
  }, [activeMenuGroup]);

  // Handle menu group hover
  const handleMenuGroupHover = (groupName: string) => {
    if (previousMenuGroup.current === null) {
      setAnimationDirection("none");
    } else if (previousMenuGroup.current === groupName) {
      setAnimationDirection("none");
    } else {
      // Determine animation direction based on menu order
      const menuOrder = MENU_GROUPS.map((g) => g.name);
      const prevIndex = menuOrder.indexOf(previousMenuGroup.current!);
      const currIndex = menuOrder.indexOf(groupName);
      setAnimationDirection(currIndex > prevIndex ? "right" : "left");
    }

    previousMenuGroup.current = groupName;
    setActiveMenuGroup(groupName);
  };

  // Close menu on route change
  useEffect(() => {
    setActiveMenuGroup(null);
  }, [location]);

  // Only redirect if user was previously authenticated but is now logged out
  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      const wasAuthenticated = sessionStorage.getItem("wasAuthenticated");
      if (wasAuthenticated) {
        navigate(ROUTES.LOGIN.path, { replace: true });
        sessionStorage.removeItem("wasAuthenticated");
      }
    }
  }, [isLoading, isAuthenticated, navigate]);

  // Mark that user was authenticated when they log in
  useEffect(() => {
    if (isAuthenticated) {
      sessionStorage.setItem("wasAuthenticated", "true");
    }
  }, [isAuthenticated]);

  const visibleMenuGroups = MENU_GROUPS.filter(
    (group) => !group.requiresAuth || isAuthenticated
  );

  return (
    <>
      <div
        className="header-wrapper"
        onMouseLeave={() => {
          // Close dropdown when mouse leaves the entire header area
          setActiveMenuGroup(null);
        }}
      >
        <header className="sticky top-0 z-50 flex justify-center pt-4 px-4">
          <div className="max-w-7xl w-full py-2 px-6 rounded-2xl backdrop-blur-2xl border shadow-2xl bg-background-secondary/60 dark:bg-[rgba(39,39,37,0.4)] border-border dark:border-white/10">
            <div className="flex items-center justify-between">
              {/* Logo and Brand */}
              <div
                className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity"
                onClick={() => navigate(ROUTES.HOME.path)}
              >
                <div className="w-10 h-10 rounded-full overflow-hidden flex items-center justify-center bg-amber-400/10">
                  <img
                    src="/logo_chronos.png"
                    alt="Chronos Logo"
                    className="w-11 h-11 object-cover"
                  />
                </div>
                <h1 className="text-xl font-bold text-amber-600 dark:text-yellow-400">
                  Chronos
                </h1>
              </div>

              {/* Desktop Navigation with Dropdowns */}
              <nav className="hidden md:flex items-center gap-8">
                {/* Home Link */}
                <a
                  href={ROUTES.HOME.path}
                  className="text-foreground hover:text-amber-600 dark:hover:text-amber-400 transition-colors"
                >
                  {t("header.home")}
                </a>

                {/* Menu Groups */}
                {visibleMenuGroups.map((group) => (
                  <button
                    key={group.name}
                    className="menu-trigger flex items-center gap-2 text-foreground hover:text-amber-600 dark:hover:text-amber-400 transition-colors"
                    onMouseEnter={() => handleMenuGroupHover(group.name)}
                    onMouseLeave={() => {
                      // Don't close dropdown, just keep the menu open
                    }}
                  >
                    {t(`header.${group.label}`)}
                    <ChevronDown
                      className={`w-4 h-4 transition-transform ${
                        activeMenuGroup === group.name ? "rotate-180" : ""
                      }`}
                    />
                  </button>
                ))}
              </nav>

              {/* Right Side - Language Switcher, Theme Toggle, Logout Button & Mobile Menu Button */}
              <div className="flex items-center gap-2">
                <LanguageSwitcher />
                <ModeToggle />
                {isAuthenticated ? (
                  <Button
                    onClick={handleLogout}
                    disabled={isLoading}
                    variant="ghost"
                    className="text-foreground dark:text-white hover:text-red-600 dark:hover:text-red-400 hover:bg-red-400/10 transition-colors hidden sm:flex"
                  >
                    <LogOut className="w-4 h-4" />
                  </Button>
                ) : (
                  <>
                    <Button
                      onClick={() => navigate(ROUTES.LOGIN.path)}
                      variant="ghost"
                      className="text-foreground dark:text-white hover:text-amber-600 dark:hover:text-amber-400 hover:bg-amber-400/10 transition-colors hidden sm:flex border"
                    >
                      {t("header.startNow")}
                    </Button>
                  </>
                )}

                {/* Hamburger Menu Button */}
                <button
                  onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                  className="md:hidden text-foreground dark:text-white hover:text-amber-600 dark:hover:text-amber-400 transition-colors"
                >
                  {mobileMenuOpen ? (
                    <X className="w-6 h-6" />
                  ) : (
                    <Menu className="w-6 h-6" />
                  )}
                </button>
              </div>
            </div>

            {/* Mobile Navigation Menu */}
            {mobileMenuOpen && (
              <nav className="md:hidden mt-4 pt-4 border-t border-black/10 dark:border-white/5 flex flex-col gap-0 bg-white/50 dark:bg-transparent rounded-b-lg">
                <a
                  href={ROUTES.HOME.path}
                  className="px-4 py-3 text-black/70 dark:text-white/80 hover:text-amber-600 dark:hover:text-amber-300 transition-colors flex items-center justify-between border-b border-black/10 dark:border-white/5"
                  onClick={() => setMobileMenuOpen(false)}
                >
                  {t("header.home")}
                  <ChevronDown className="w-4 h-4 rotate-90 opacity-40 group-hover:opacity-60 transition-opacity" />
                </a>

                {visibleMenuGroups.map((group) => (
                  <div key={group.name}>
                    {/* Main menu item button */}
                    <button
                      onClick={() =>
                        setExpandedMobileGroup(
                          expandedMobileGroup === group.name ? null : group.name
                        )
                      }
                      className="px-4 py-3 w-full flex items-center justify-between text-left font-medium text-amber-600 dark:text-amber-300/90 hover:text-amber-700 dark:hover:text-amber-300 transition-colors border-b border-black/10 dark:border-white/5 group"
                    >
                      {t(`header.${group.label}`)}
                      <ChevronDown
                        className={`w-4 h-4 opacity-40 group-hover:opacity-60 transition-all ${
                          expandedMobileGroup === group.name ? "rotate-90" : ""
                        }`}
                      />
                    </button>

                    {/* Animated submenu slide-in */}
                    {expandedMobileGroup === group.name && (
                      <div className="overflow-hidden bg-black/5 dark:bg-white/2">
                        <div className="animate-slideInLeft flex flex-col gap-0">
                          {group.items.map((item) => (
                            <a
                              key={item.path}
                              href={item.path}
                              target={item.external ? "_blank" : undefined}
                              rel={
                                item.external
                                  ? "noopener noreferrer"
                                  : undefined
                              }
                              className="px-6 py-2.5 text-black/60 dark:text-white/70 hover:text-black dark:hover:text-white transition-colors text-sm border-b border-black/5 dark:border-white/5 last:border-b-0"
                              onClick={() => {
                                setMobileMenuOpen(false);
                                setExpandedMobileGroup(null);
                              }}
                            >
                              {t(`header.${item.name}`)}
                            </a>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}

                <div className=" my-2" />
                {isAuthenticated ? (
                  <button
                    onClick={() => {
                      handleLogout();
                      setMobileMenuOpen(false);
                    }}
                    disabled={isLoading}
                    className="px-4 py-3 text-red-500 dark:text-red-400/80 hover:text-red-600 dark:hover:text-red-300 transition-colors flex items-center gap-3 border-b border-black/10 dark:border-white/5 text-sm"
                  >
                    <LogOut className="w-4 h-4" />
                    {t("header.logout") || "Logout"}
                  </button>
                ) : (
                  <>
                    <button
                      onClick={() => {
                        navigate(`${ROUTES.LOGIN.path}?mode=login`);
                        setMobileMenuOpen(false);
                      }}
                      className="px-4 py-3 w-full text-center text-black/70 dark:text-white/70 hover:text-black dark:hover:text-white border border-black/15 dark:border-white/10 hover:border-black/25 dark:hover:border-white/20 transition-colors rounded-lg text-sm mb-2"
                    >
                      {t("header.signIn") || "Log In"}
                    </button>
                    <button
                      onClick={() => {
                        navigate(`${ROUTES.LOGIN.path}?mode=signup`);
                        setMobileMenuOpen(false);
                      }}
                      className="px-4 py-3 w-full text-center bg-white text-black hover:bg-white/90 transition-colors rounded-lg font-semibold text-sm"
                    >
                      {t("header.signUp") || "Get Started"}
                    </button>
                  </>
                )}
              </nav>
            )}
          </div>
        </header>

        {/* Fixed Dropdown Container - Static, always shown when hovering */}
        {activeMenuGroup && (
          <div className="dropdown-container fixed top-[calc(4rem+2px)] left-1/2 transform -translate-x-1/2 z-40 w-screen flex justify-center pointer-events-none">
            {/* Invisible bridge to prevent dropdown from closing when moving mouse from header to dropdown */}
            <div className="absolute -top-2 left-0 right-0 h-4 -z-10" />

            {/* Static container with enhanced glassmorphism */}
            <div
              className="border border-black/10 dark:border-white/5 rounded-xl shadow-2xl
              py-6 px-8 flex gap-12 min-w-fit overflow-hidden h-52 pointer-events-auto bg-white/50 dark:bg-black/0"
            >
              {/* Content wrapper that animates - key added to force re-render */}
              <div
                key={`content-${activeMenuGroup}`}
                className={`flex items-center gap-16 animation-${animationDirection}`}
              >
                {/* Left column - Menu items */}
                <div className="flex flex-col gap-1 min-w-max justify-center">
                  {MENU_GROUPS.find(
                    (g) => g.name === activeMenuGroup
                  )?.items.map((item, index) => (
                    <div key={item.path}>
                      <a
                        href={item.path}
                        target={item.external ? "_blank" : undefined}
                        rel={item.external ? "noopener noreferrer" : undefined}
                        className="px-4 py-2 text-black/70 dark:text-white/70 hover:text-amber-600 dark:hover:text-amber-300 hover:bg-amber-400/5 dark:hover:bg-amber-400/10 transition-all duration-200 rounded whitespace-nowrap text-sm block"
                        onClick={() => setActiveMenuGroup(null)}
                      >
                        {t(`header.${item.name}`)}
                      </a>
                      {index <
                        (MENU_GROUPS.find((g) => g.name === activeMenuGroup)
                          ?.items.length ?? 0) -
                          1 && (
                        <div className="h-px bg-gradient-to-r from-transparent via-black/10 dark:via-white/15 to-transparent my-0.5" />
                      )}
                    </div>
                  ))}

                  {/* Logout button for settings group */}
                  {activeMenuGroup === "settings" && (
                    <>
                      <div className="h-px bg-gradient-to-r from-transparent via-black/15 dark:via-white/20 to-transparent my-2" />
                      <button
                        onClick={() => {
                          handleLogout();
                          setActiveMenuGroup(null);
                        }}
                        disabled={isLoading}
                        className="px-4 py-2 text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 
                        hover:bg-red-400/5 dark:hover:bg-red-400/10 transition-all duration-200 rounded whitespace-nowrap text-sm"
                      >
                        {t("header.logout") || "Logout"}
                      </button>
                    </>
                  )}
                </div>

                {/* Right column - Elegant geometric rectangle with gold pattern */}
                <div className="w-56 h-32 rounded-lg bg-white/40 dark:bg-black relative overflow-hidden flex-shrink-0 border border-black/15 dark:border-white/30">
                  {/* Diagonal stripes pattern */}
                  <svg
                    className="absolute inset-0 w-full h-full"
                    preserveAspectRatio="none"
                  >
                    <defs>
                      <pattern
                        id="diagonal-stripes"
                        x="0"
                        y="0"
                        width="20"
                        height="20"
                        patternUnits="userSpaceOnUse"
                      >
                        <line
                          x1="0"
                          y1="0"
                          x2="20"
                          y2="20"
                          stroke="rgba(215, 160, 41, 0.2)"
                          strokeWidth="1"
                        />
                      </pattern>
                    </defs>
                    <rect
                      width="100%"
                      height="100%"
                      fill="url(#diagonal-stripes)"
                    />
                  </svg>

                  {/* Gold border accents */}
                  <div className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-yellow-600/40 to-transparent" />
                  <div className="absolute bottom-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-yellow-700/30 to-transparent" />
                  <div className="absolute top-0 left-0 w-px h-full bg-gradient-to-b from-transparent via-yellow-600/30 to-transparent" />
                  <div className="absolute top-0 right-0 w-px h-full bg-gradient-to-b from-transparent via-yellow-700/20 to-transparent" />

                  {/* Geometric shapes - gold triangles */}
                  <svg
                    className="absolute top-4 left-5 w-5 h-5"
                    viewBox="0 0 24 24"
                    fill="none"
                  >
                    <polygon
                      points="12,3 20,20 4,20"
                      fill="rgba(215, 160, 41, 0.5)"
                    />
                  </svg>
                  <svg
                    className="absolute bottom-4 right-5 w-4 h-4"
                    viewBox="0 0 24 24"
                    fill="none"
                  >
                    <polygon
                      points="12,3 20,20 4,20"
                      fill="rgba(215, 160, 41, 0.4)"
                    />
                  </svg>
                  <svg
                    className="absolute top-1/2 right-6 w-3 h-3"
                    viewBox="0 0 24 24"
                    fill="none"
                  >
                    <polygon
                      points="12,3 20,20 4,20"
                      fill="rgba(215, 160, 41, 0.3)"
                    />
                  </svg>

                  {/* Center glow */}
                  <div className="absolute inset-0 bg-gradient-to-br from-yellow-600/10 via-transparent to-yellow-700/5" />
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </>
  );
}
