import { useState, useRef, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ChevronDown, Package, Zap, Wrench } from "lucide-react";
import { useChangelogParser } from "../hooks/useChangelogParser";
import { Header } from "../components/common/header";
import { Footer } from "@/components/common/footer";

export function ChangelogPage() {
  const { t } = useTranslation();
  const { parseChangelog } = useChangelogParser();
  const [expandedVersions, setExpandedVersions] = useState<string[]>([
    parseChangelog()[0]?.version || "",
  ]);
  const versionRefs = useRef<{ [key: string]: HTMLDivElement | null }>({});

  const changelog = parseChangelog();

  const toggleVersion = (version: string) => {
    setExpandedVersions((prev) =>
      prev.includes(version)
        ? prev.filter((v) => v !== version)
        : [...prev, version]
    );
  };

  const scrollToVersion = (version: string) => {
    const element = versionRefs.current[version];
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
      if (!expandedVersions.includes(version)) {
        toggleVersion(version);
      }
    }
  };

  const getCategoryIcon = (categoryName: string) => {
    if (categoryName.toLowerCase().includes("major")) {
      return <Zap className="w-5 h-5 text-yellow-500" />;
    } else if (categoryName.toLowerCase().includes("minor")) {
      return <Package className="w-5 h-5 text-blue-500" />;
    } else if (categoryName.toLowerCase().includes("fix")) {
      return <Wrench className="w-5 h-5 text-green-500" />;
    }
    return <Package className="w-5 h-5 text-gray-500" />;
  };

  // Evenly spread each version across the bar width, oldest to newest
  // left-to-right. `changelog` is ordered newest-first.
  const datePositions = useMemo(() => {
    if (changelog.length === 0) return [];

    const lastIndex = changelog.length - 1;
    return changelog.map((v, idx) => ({
      version: v.version,
      date: v.date,
      position: lastIndex === 0 ? 50 : ((lastIndex - idx) / lastIndex) * 100,
    }));
  }, [changelog]);

  return (
    <>
      <Header />
      <main className="min-h-screen bg-background dark:bg-background py-12 px-4 sm:px-6 lg:px-8 pt-24">
        <div className="max-w-4xl mx-auto">
          {/* Header Section */}
          <div className="mb-12">
            <h1 className="text-4xl sm:text-5xl font-bold mb-4 text-foreground">
              {t("changelog.title")}
            </h1>
            <p className="text-lg text-foreground/70 max-w-2xl">
              {t("changelog.subtitle")}
            </p>
          </div>

          {/* Date-Proportional Timeline */}
          {changelog.length > 0 && (
            <div className="mb-16">
              <div className="relative h-24">
                {/* Timeline Line */}
                <div className="absolute top-1/2 left-0 right-0 h-1 bg-gradient-to-r from-amber-500/20 via-amber-500/50 to-amber-500/20 rounded-full transform -translate-y-1/2" />

                {/* Timeline Start and End Labels */}
                <div className="absolute top-0 left-0 text-xs text-foreground/60">
                  {datePositions[datePositions.length - 1]?.date || "Start"}
                </div>
                <div className="absolute top-0 right-0 text-xs text-foreground/60">
                  Today
                </div>

                {/* Timeline Dots - Proportionally Spaced */}
                {datePositions.map((item) => (
                  <button
                    key={item.version}
                    onClick={() => scrollToVersion(item.version)}
                    className="group absolute top-1/2 transform -translate-y-1/2 flex flex-col items-center gap-1 -translate-x-1/2"
                    style={{ left: `${item.position}%` }}
                  >
                    {/* Dot */}
                    <div
                      className="relative w-5 h-5 rounded-full border-2 border-amber-500 bg-background/80 backdrop-blur
                      transition-all duration-300 hover:scale-150 hover:shadow-lg hover:shadow-amber-500/50
                      group-hover:bg-amber-500/20 z-10"
                    />

                    {/* Version Tooltip */}
                    <div
                      className="absolute top-full pt-4 px-3 py-2 bg-background/95 dark:bg-background/90 border border-border/50 dark:border-white/10
                      rounded-lg backdrop-blur text-xs font-medium text-foreground whitespace-nowrap
                      opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none
                      shadow-lg dark:shadow-xl/20 z-20"
                    >
                      <div>v{item.version}</div>
                      <div className="text-foreground/60 text-xs">
                        {item.date}
                      </div>
                      <div className="absolute -top-1.5 left-1/2 transform -translate-x-1/2 w-3 h-3 bg-background/95 dark:bg-background/90 border-t border-l border-border/50 dark:border-white/10 rounded-sm rotate-45" />
                    </div>

                    {/* Active indicator */}
                    {expandedVersions.includes(item.version) && (
                      <div className="absolute -inset-1.5 rounded-full border border-amber-500/50 opacity-50 animate-pulse" />
                    )}
                  </button>
                ))}
              </div>

              {/* Timeline Info Text */}
              <div className="text-center mt-8 text-sm text-foreground/60">
                {t("changelog.clickToDots")}
              </div>
            </div>
          )}

          {/* Changelog Versions */}
          <div className="space-y-4">
            {changelog.map((versionData) => (
              <div
                key={versionData.version}
                ref={(el) => {
                  if (el) {
                    versionRefs.current[versionData.version] = el;
                  }
                }}
                className="group relative overflow-hidden rounded-xl border border-border/50 dark:border-white/10 backdrop-blur-sm transition-all duration-300 hover:border-border/80 dark:hover:border-white/20 hover:shadow-lg dark:hover:shadow-xl/20"
              >
                {/* Background gradient on hover */}
                <div className="absolute inset-0 bg-gradient-to-r from-amber-500/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

                {/* Version Header - Always Visible */}
                <button
                  onClick={() => toggleVersion(versionData.version)}
                  className="w-full text-left p-6 flex items-center justify-between relative z-10"
                >
                  <div className="flex items-center gap-4">
                    {/* Version Badge */}
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-3">
                        <span className="inline-block px-3 py-1 bg-amber-500/20 text-amber-600 dark:text-amber-400 rounded-full text-sm font-semibold">
                          v{versionData.version}
                        </span>
                        <time className="text-sm text-foreground/60">
                          {versionData.date}
                        </time>
                      </div>
                    </div>
                  </div>

                  <ChevronDown
                    className={`w-5 h-5 text-foreground/60 transition-transform duration-300 ${
                      expandedVersions.includes(versionData.version)
                        ? "rotate-180"
                        : ""
                    }`}
                  />
                </button>

                {/* Version Content - Expandable */}
                {expandedVersions.includes(versionData.version) && (
                  <div className="relative z-10 border-t border-border/30 dark:border-white/5 px-6 py-6 bg-white/30 dark:bg-black/20 space-y-6">
                    {versionData.categories.length === 0 ? (
                      <p className="text-foreground/60 italic">
                        {t("changelog.noChanges")}
                      </p>
                    ) : (
                      versionData.categories.map((category, catIndex) => (
                        <div
                          key={`${category.name}-${catIndex}`}
                          className="space-y-4"
                        >
                          {/* Category Header */}
                          <div className="flex items-center gap-3 mb-4">
                            {getCategoryIcon(category.name)}
                            <h3 className="text-lg font-semibold text-foreground">
                              {category.name}
                            </h3>
                          </div>

                          {/* Category Entries */}
                          <div className="space-y-3 pl-8">
                            {category.entries.map((entry, entryIndex) => (
                              <div
                                key={`${entry.section}-${entryIndex}`}
                                className="space-y-2"
                              >
                                {/* Subsection Header */}
                                {entry.section && (
                                  <h4 className="text-md font-medium text-foreground/90 text-amber-600 dark:text-amber-400">
                                    {entry.section}
                                  </h4>
                                )}

                                {/* Items List */}
                                <ul className="space-y-2">
                                  {entry.items.map((item, itemIndex) => (
                                    <li
                                      key={itemIndex}
                                      className="flex gap-3 text-foreground/80 group/item"
                                    >
                                      <span className="inline-block w-2 h-2 rounded-full bg-amber-500 mt-2 flex-shrink-0 group-hover/item:scale-150 transition-transform" />
                                      <span className="leading-relaxed">
                                        {/* Format inline code */}
                                        {item
                                          .split(/(`[^`]+`)/g)
                                          .map((part, i) =>
                                            part.startsWith("`") ? (
                                              <code
                                                key={i}
                                                className="bg-black/20 dark:bg-white/10 px-2 py-0.5 rounded text-xs font-mono text-amber-600 dark:text-amber-400"
                                              >
                                                {part.slice(1, -1)}
                                              </code>
                                            ) : (
                                              part
                                            )
                                          )}
                                      </span>
                                    </li>
                                  ))}
                                </ul>
                              </div>
                            ))}
                          </div>

                          {/* Divider between categories */}
                          {catIndex < versionData.categories.length - 1 && (
                            <div className="h-px bg-gradient-to-r from-transparent via-border/50 to-transparent my-6" />
                          )}
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Footer */}
          {changelog.length > 0 && (
            <div className="mt-12 text-center">
              <p className="text-sm text-foreground/60">
                {changelog.length === 1
                  ? t("changelog.showingVersions", { count: 1 })
                  : t("changelog.showingVersionsPlural", {
                      count: changelog.length,
                    })}
              </p>
            </div>
          )}
        </div>
      </main>
      <Footer />
    </>
  );
}
