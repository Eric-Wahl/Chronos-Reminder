import { useTranslation } from "react-i18next";
import { Header } from "@/components/common/header";
import { Footer } from "@/components/common/footer";
import {
  Server,
  Database,
  Container,
  Code,
  Settings,
  Layers,
  HardDrive,
  Link,
  Copy,
  Check,
  Menu,
  X,
} from "lucide-react";
import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function SelfHostPage() {
  const { t } = useTranslation();
  const [copiedCommand, setCopiedCommand] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<string>("overview");
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [navbarTop, setNavbarTop] = useState(320); // Initial position (top-80 = 320px)

  // Track scroll position to highlight active section and adjust navbar
  useEffect(() => {
    const handleScroll = () => {
      const sections = [
        "overview",
        "architecture",
        "backend",
        "frontend",
        "environment",
        "database",
      ];

      // Find the section that is currently in view
      for (const sectionId of sections) {
        const element = document.getElementById(sectionId);
        if (element) {
          const rect = element.getBoundingClientRect();
          // Check if section is in the upper half of viewport
          if (rect.top <= 150 && rect.bottom >= 150) {
            setActiveSection(sectionId);
            break;
          }
        }
      }

      // Adjust navbar position based on scroll
      // Start moving it up after scrolling past 320px
      const scrollY = window.scrollY;
      const minTop = 128; // Minimum top position (top-32 equivalent)
      const maxTop = 320; // Maximum top position (top-80 equivalent)

      // Move navbar up as we scroll, but keep it at least at minTop
      const newTop = Math.max(minTop, maxTop - scrollY);
      setNavbarTop(newTop);
    };

    window.addEventListener("scroll", handleScroll);
    handleScroll(); // Initial check

    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedCommand(id);
    setTimeout(() => setCopiedCommand(null), 2000);
  };

  const CodeBlock = ({
    code,
    id,
    language = "bash",
  }: {
    code: string;
    id: string;
    language?: string;
  }) => (
    <div className="relative group w-full">
      <pre className="bg-muted/50 dark:bg-muted/30 p-4 rounded-lg overflow-x-auto border border-border/50 max-w-full">
        <code
          className={`language-${language} text-sm break-all whitespace-pre-wrap`}
        >
          {code}
        </code>
      </pre>
      <button
        onClick={() => copyToClipboard(code, id)}
        className="absolute top-3 right-3 p-2 rounded-md bg-background/80 hover:bg-background border border-border/50 opacity-0 group-hover:opacity-100 transition-opacity"
        aria-label="Copy to clipboard"
      >
        {copiedCommand === id ? (
          <Check className="w-4 h-4 text-green-500" />
        ) : (
          <Copy className="w-4 h-4" />
        )}
      </button>
    </div>
  );

  const SectionCard = ({
    icon: Icon,
    title,
    children,
    id,
  }: {
    icon: React.ComponentType<{ className?: string }>;
    title: string;
    children: React.ReactNode;
    id: string;
  }) => (
    <Card id={id} className="scroll-mt-24">
      <CardHeader>
        <CardTitle className="flex items-center gap-3 text-2xl">
          <div className="p-2 rounded-lg bg-primary/10">
            <Icon className="w-6 h-6 text-primary" />
          </div>
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">{children}</CardContent>
    </Card>
  );

  const navigationSections = [
    {
      id: "overview",
      label: t("selfHost.nav.overview"),
      icon: Layers,
    },
    {
      id: "architecture",
      label: t("selfHost.nav.architecture"),
      icon: Server,
    },
    {
      id: "backend",
      label: t("selfHost.nav.backend"),
      icon: Container,
    },
    {
      id: "frontend",
      label: t("selfHost.nav.frontend"),
      icon: Code,
    },
    {
      id: "environment",
      label: t("selfHost.nav.environment"),
      icon: Settings,
    },
    {
      id: "database",
      label: t("selfHost.nav.database"),
      icon: Database,
    },
  ];

  const scrollToSection = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
      setMobileMenuOpen(false); // Close mobile menu after navigation
    }
  };

  return (
    <>
      <Header />
      <main className="min-h-screen bg-background dark:bg-background pt-24">
        <div className="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-12">
          {/* Header Section */}
          <div className="mb-12">
            <h1 className="text-4xl sm:text-5xl font-bold mb-4 text-foreground">
              {t("selfHost.title")}
            </h1>
            <p className="text-lg text-foreground/70 max-w-3xl">
              {t("selfHost.subtitle")}
            </p>
          </div>

          {/* Mobile Navigation Button */}
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="lg:hidden fixed bottom-6 right-6 z-50 p-4 rounded-full bg-primary text-primary-foreground shadow-lg hover:bg-primary/90 transition-colors"
            aria-label="Toggle navigation menu"
          >
            {mobileMenuOpen ? (
              <X className="w-6 h-6" />
            ) : (
              <Menu className="w-6 h-6" />
            )}
          </button>

          {/* Mobile Navigation Menu */}
          {mobileMenuOpen && (
            <div className="lg:hidden fixed inset-0 z-40 bg-background/95 backdrop-blur-sm">
              <nav className="h-full overflow-y-auto p-6 pt-24">
                <div className="text-sm font-semibold text-foreground/60 mb-3">
                  {t("selfHost.nav.title")}
                </div>
                <div className="space-y-2">
                  {navigationSections.map((section) => (
                    <button
                      key={section.id}
                      onClick={() => scrollToSection(section.id)}
                      className={`w-full flex items-center gap-3 px-4 py-3 text-base rounded-lg transition-all ${
                        activeSection === section.id
                          ? "bg-primary/10 text-primary font-medium border-l-4 border-primary"
                          : "text-foreground/70 hover:text-foreground hover:bg-muted/50"
                      }`}
                    >
                      <section.icon className="w-5 h-5" />
                      {section.label}
                    </button>
                  ))}
                </div>
              </nav>
            </div>
          )}
        </div>

        {/* Content with Sidebar */}
        <div className="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col lg:flex-row gap-8">
            {/* Fixed Navigation - takes space in layout */}
            <div className="hidden lg:block lg:w-64 flex-shrink-0">
              <nav
                className="fixed w-64 space-y-1 transition-all duration-100"
                style={{ top: `${navbarTop}px` }}
              >
                <div className="text-sm font-semibold text-foreground/60 mb-3">
                  {t("selfHost.nav.title")}
                </div>
                {navigationSections.map((section) => (
                  <button
                    key={section.id}
                    onClick={() => scrollToSection(section.id)}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 text-sm rounded-lg transition-all ${
                      activeSection === section.id
                        ? "bg-primary/10 text-primary font-medium border-l-2 border-primary"
                        : "text-foreground/70 hover:text-foreground hover:bg-muted/50"
                    }`}
                  >
                    <section.icon className="w-4 h-4" />
                    {section.label}
                  </button>
                ))}
              </nav>
            </div>

            {/* Main Content */}
            <div className="flex-1 space-y-8 min-w-0 w-full pb-12">
              {/* Overview */}
              <SectionCard
                id="overview"
                icon={Layers}
                title={t("selfHost.overview.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.overview.description")}
                </p>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-6">
                  <div className="p-4 rounded-lg bg-muted/30 border border-border/50">
                    <Server className="w-8 h-8 text-primary mb-2" />
                    <h3 className="font-semibold mb-1">
                      {t("selfHost.overview.backend")}
                    </h3>
                    <p className="text-sm text-foreground/70">
                      {t("selfHost.overview.backendDesc")}
                    </p>
                  </div>
                  <div className="p-4 rounded-lg bg-muted/30 border border-border/50">
                    <Code className="w-8 h-8 text-primary mb-2" />
                    <h3 className="font-semibold mb-1">
                      {t("selfHost.overview.frontend")}
                    </h3>
                    <p className="text-sm text-foreground/70">
                      {t("selfHost.overview.frontendDesc")}
                    </p>
                  </div>
                  <div className="p-4 rounded-lg bg-muted/30 border border-border/50">
                    <Database className="w-8 h-8 text-primary mb-2" />
                    <h3 className="font-semibold mb-1">
                      {t("selfHost.overview.database")}
                    </h3>
                    <p className="text-sm text-foreground/70">
                      {t("selfHost.overview.databaseDesc")}
                    </p>
                  </div>
                </div>
              </SectionCard>

              {/* Architecture */}
              <SectionCard
                id="architecture"
                icon={Server}
                title={t("selfHost.architecture.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.architecture.description")}
                </p>
                <div className="space-y-3 mt-4">
                  <div className="flex items-start gap-3">
                    <div className="p-1.5 rounded bg-primary/10 mt-1">
                      <Server className="w-4 h-4 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium">
                        {t("selfHost.architecture.backendTitle")}
                      </p>
                      <p className="text-sm text-foreground/70">
                        {t("selfHost.architecture.backendDesc")}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="p-1.5 rounded bg-primary/10 mt-1">
                      <Database className="w-4 h-4 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium">
                        {t("selfHost.architecture.postgresTitle")}
                      </p>
                      <p className="text-sm text-foreground/70">
                        {t("selfHost.architecture.postgresDesc")}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="p-1.5 rounded bg-primary/10 mt-1">
                      <HardDrive className="w-4 h-4 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium">
                        {t("selfHost.architecture.redisTitle")}
                      </p>
                      <p className="text-sm text-foreground/70">
                        {t("selfHost.architecture.redisDesc")}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="p-1.5 rounded bg-primary/10 mt-1">
                      <Code className="w-4 h-4 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium">
                        {t("selfHost.architecture.frontendTitle")}
                      </p>
                      <p className="text-sm text-foreground/70">
                        {t("selfHost.architecture.frontendDesc")}
                      </p>
                    </div>
                  </div>
                </div>
              </SectionCard>

              {/* Backend Deployment */}
              <SectionCard
                id="backend"
                icon={Container}
                title={t("selfHost.backend.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.backend.description")}
                </p>

                <div className="space-y-4 mt-6">
                  <div>
                    <h4 className="font-semibold mb-2 flex items-center gap-2">
                      <Container className="w-4 h-4" />
                      {t("selfHost.backend.dockerImage")}
                    </h4>
                    <CodeBlock
                      id="backend-image"
                      code="ghcr.io/eric-wahl/chronos-reminder:1.0.3"
                    />
                  </div>

                  <div>
                    <h4 className="font-semibold mb-2">
                      {t("selfHost.backend.dockerRun")}
                    </h4>
                    <CodeBlock
                      id="backend-run"
                      code={`docker run -d \\
  --name chronos-backend \\
  -p 8080:8080 \\
  --env-file .env \\
  ghcr.io/eric-wahl/chronos-reminder:1.0.3`}
                    />
                  </div>

                  <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/30">
                    <p className="text-sm flex items-start gap-2">
                      <Database className="w-4 h-4 mt-0.5 text-blue-500" />
                      <span>{t("selfHost.backend.migrationNote")}</span>
                    </p>
                  </div>
                </div>
              </SectionCard>

              {/* Frontend Deployment */}
              <SectionCard
                id="frontend"
                icon={Code}
                title={t("selfHost.frontend.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.frontend.description")}
                </p>

                <div className="space-y-4 mt-6">
                  <div>
                    <h4 className="font-semibold mb-2 flex items-center gap-2">
                      <Container className="w-4 h-4" />
                      {t("selfHost.frontend.dockerImage")}
                    </h4>
                    <CodeBlock
                      id="frontend-image"
                      code="ghcr.io/eric-wahl/chronos-reminder/web:1.0.3"
                    />
                  </div>

                  <div>
                    <h4 className="font-semibold mb-2">
                      {t("selfHost.frontend.dockerRun")}
                    </h4>
                    <CodeBlock
                      id="frontend-run"
                      code={`docker run -d \\
  --name chronos-web \\
  -p 3000:3000 \\
  -e VITE_API_URL=http://your-backend-url:8080 \\
  -e VITE_DISCORD_CLIENT_ID=your_discord_client_id \\
  -e VITE_DISCORD_REDIRECT_URI=http://your-domain.com/auth/callback/discord \\
  ghcr.io/eric-wahl/chronos-reminder/web:1.0.3`}
                    />
                  </div>

                  <div>
                    <h4 className="font-semibold mb-2">
                      {t("selfHost.frontend.envVars")}
                    </h4>
                    <div className="rounded-lg border border-border/50 overflow-x-auto">
                      <div className="min-w-full inline-block align-middle">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.variable")}
                              </TableHead>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.description")}
                              </TableHead>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.example")}
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                VITE_API_URL
                              </TableCell>
                              <TableCell>
                                {t("selfHost.frontend.vars.apiUrl")}
                              </TableCell>
                              <TableCell className="font-mono text-xs text-foreground/70">
                                http://localhost:8080
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                VITE_DISCORD_CLIENT_ID
                              </TableCell>
                              <TableCell>
                                {t("selfHost.frontend.vars.discordClientId")}
                              </TableCell>
                              <TableCell className="font-mono text-xs text-foreground/70">
                                1234567890123456789
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                VITE_DISCORD_REDIRECT_URI
                              </TableCell>
                              <TableCell>
                                {t("selfHost.frontend.vars.discordRedirect")}
                              </TableCell>
                              <TableCell className="font-mono text-xs text-foreground/70">
                                http://domain.com/auth/callback/discord
                              </TableCell>
                            </TableRow>
                          </TableBody>
                        </Table>
                      </div>
                    </div>
                  </div>
                </div>
              </SectionCard>

              {/* Environment Variables */}
              <SectionCard
                id="environment"
                icon={Settings}
                title={t("selfHost.environment.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.environment.description")}
                </p>

                <div className="space-y-4 mt-6">
                  <div>
                    <h4 className="font-semibold mb-3">
                      {t("selfHost.environment.allVars")}
                    </h4>
                    <div className="rounded-lg border border-border/50 overflow-x-auto">
                      <div className="min-w-full inline-block align-middle">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.variable")}
                              </TableHead>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.required")}
                              </TableHead>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.description")}
                              </TableHead>
                              <TableHead className="font-semibold">
                                {t("selfHost.table.default")}
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {/* Core Settings */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.core")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                ENVIRONMENT
                              </TableCell>
                              <TableCell>
                                <span className="text-yellow-600 dark:text-yellow-500">
                                  {t("selfHost.table.recommended")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.environment")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                DEV
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                API_PORT
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.apiPort")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                8080
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                API_CORS
                              </TableCell>
                              <TableCell>
                                <span className="text-yellow-600 dark:text-yellow-500">
                                  {t("selfHost.table.recommended")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.apiCors")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                *
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DEFAULT_TZ
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.defaultTz")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                16
                              </TableCell>
                            </TableRow>

                            {/* Discord Configuration */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.discord")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DISCORD_BOT_TOKEN
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.discordBotToken")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DISCORD_CLIENT_ID
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.discordClientId")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DISCORD_CLIENT_SECRET
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t(
                                  "selfHost.environment.vars.discordClientSecret",
                                )}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DISCORD_REDIRECT_URI
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t(
                                  "selfHost.environment.vars.discordRedirectUri",
                                )}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>

                            {/* Database Configuration */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.database")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DB_HOST
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.dbHost")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                localhost
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DB_PORT
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.dbPort")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                5432
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DB_USER
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.dbUser")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                postgres
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DB_PASSWORD
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.dbPassword")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                DB_NAME
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.dbName")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                chronosreminder
                              </TableCell>
                            </TableRow>

                            {/* Redis Configuration */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.redis")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                REDIS_HOST
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.redisHost")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                localhost
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                REDIS_PORT
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.redisPort")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                6379
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                REDIS_PASSWORD
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.redisPassword")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                REDIS_DB
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.redisDb")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                0
                              </TableCell>
                            </TableRow>

                            {/* Security */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.security")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                JWT_SECRET
                              </TableCell>
                              <TableCell>
                                <span className="text-red-600 dark:text-red-500">
                                  {t("selfHost.table.required")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.jwtSecret")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>

                            {/* Rate Limiting */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.rateLimit")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                RATE_LIMIT_ENABLED
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t(
                                  "selfHost.environment.vars.rateLimitEnabled",
                                )}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                true
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                RATE_LIMIT_REQUESTS_PER_WINDOW
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t(
                                  "selfHost.environment.vars.rateLimitRequests",
                                )}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                100
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                RATE_LIMIT_WINDOW_SECONDS
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.rateLimitWindow")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                60
                              </TableCell>
                            </TableRow>

                            {/* Email (Optional) */}
                            <TableRow>
                              <TableCell
                                colSpan={4}
                                className="font-semibold bg-muted/30"
                              >
                                {t("selfHost.environment.categories.email")}
                              </TableCell>
                            </TableRow>
                            <TableRow>
                              <TableCell className="font-mono text-sm">
                                RESEND_API_KEY
                              </TableCell>
                              <TableCell>
                                <span className="text-green-600 dark:text-green-500">
                                  {t("selfHost.table.optional")}
                                </span>
                              </TableCell>
                              <TableCell>
                                {t("selfHost.environment.vars.resendApiKey")}
                              </TableCell>
                              <TableCell className="font-mono text-xs">
                                -
                              </TableCell>
                            </TableRow>
                          </TableBody>
                        </Table>
                      </div>
                    </div>
                  </div>

                  <div className="p-4 rounded-lg bg-amber-500/10 border border-amber-500/30">
                    <p className="text-sm font-semibold mb-2 flex items-center gap-2">
                      <Settings className="w-4 h-4 text-amber-500" />
                      {t("selfHost.environment.exampleTitle")}
                    </p>
                    <CodeBlock
                      id="env-example"
                      language="bash"
                      code={`# Core Settings
ENVIRONMENT=PROD
API_PORT=8080
API_CORS=*
DEFAULT_TZ=16

# Discord Configuration
DISCORD_BOT_TOKEN=your_bot_token_here
DISCORD_CLIENT_ID=your_client_id
DISCORD_CLIENT_SECRET=your_client_secret
DISCORD_REDIRECT_URI=http://your-domain.com/auth/callback/discord

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=chronosuser
DB_PASSWORD=your_secure_password
DB_NAME=chronosreminder

# Redis Configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Security
JWT_SECRET=your-super-secret-jwt-key-change-this

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_WINDOW=100
RATE_LIMIT_WINDOW_SECONDS=60

# Email (Optional - for password reset)
RESEND_API_KEY=your_resend_api_key`}
                    />
                  </div>
                </div>
              </SectionCard>

              {/* Database Setup */}
              <SectionCard
                id="database"
                icon={Database}
                title={t("selfHost.database.title")}
              >
                <p className="text-foreground/80">
                  {t("selfHost.database.description")}
                </p>

                <div className="space-y-6 mt-6">
                  {/* PostgreSQL */}
                  <div>
                    <h4 className="font-semibold mb-3 flex items-center gap-2">
                      <Database className="w-5 h-5 text-primary" />
                      {t("selfHost.database.postgresTitle")}
                    </h4>
                    <p className="text-sm text-foreground/70 mb-4">
                      {t("selfHost.database.postgresDesc")}
                    </p>
                    <CodeBlock
                      id="postgres-run"
                      code={`docker run -d \\
  --name chronos-postgres \\
  -e POSTGRES_USER=chronosuser \\
  -e POSTGRES_PASSWORD=your_secure_password \\
  -e POSTGRES_DB=chronosreminder \\
  -p 5432:5432 \\
  -v postgres_data:/var/lib/postgresql/data \\
  postgres:16`}
                    />
                  </div>

                  {/* Redis */}
                  <div>
                    <h4 className="font-semibold mb-3 flex items-center gap-2">
                      <HardDrive className="w-5 h-5 text-primary" />
                      {t("selfHost.database.redisTitle")}
                    </h4>
                    <p className="text-sm text-foreground/70 mb-4">
                      {t("selfHost.database.redisDesc")}
                    </p>
                    <CodeBlock
                      id="redis-run"
                      code={`docker run -d \\
  --name chronos-redis \\
  -p 6379:6379 \\
  -v redis_data:/data \\
  redis:8-alpine redis-server --appendonly yes`}
                    />
                  </div>

                  {/* Migration Note */}
                  <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/30">
                    <p className="text-sm font-semibold mb-2 flex items-center gap-2">
                      <Database className="w-4 h-4 text-green-500" />
                      {t("selfHost.database.migrationTitle")}
                    </p>
                    <p className="text-sm text-foreground/80">
                      {t("selfHost.database.migrationDesc")}
                    </p>
                  </div>

                  {/* Docker Compose */}
                  <div>
                    <h4 className="font-semibold mb-3 flex items-center gap-2">
                      <Layers className="w-5 h-5 text-primary" />
                      {t("selfHost.database.dockerComposeTitle")}
                    </h4>
                    <p className="text-sm text-foreground/70 mb-4">
                      {t("selfHost.database.dockerComposeDesc")}
                    </p>
                    <CodeBlock
                      id="docker-compose"
                      language="yaml"
                      code={`version: "3.8"

services:
  postgres:
    image: postgres:16
    container_name: chronos-postgres
    environment:
      - POSTGRES_USER=\${DB_USER}
      - POSTGRES_PASSWORD=\${DB_PASSWORD}
      - POSTGRES_DB=\${DB_NAME}
    ports:
      - "\${DB_PORT}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:8-alpine
    container_name: chronos-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes
    restart: unless-stopped

  backend:
    image: ghcr.io/eric-wahl/chronos-reminder:1.0.3
    container_name: chronos-backend
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  web:
    image: ghcr.io/eric-wahl/chronos-reminder/web:1.0.3
    container_name: chronos-web
    environment:
      - VITE_API_URL=http://localhost:8080
      - VITE_DISCORD_CLIENT_ID=\${DISCORD_CLIENT_ID}
      - VITE_DISCORD_REDIRECT_URI=\${DISCORD_REDIRECT_URI}
    ports:
      - "3000:3000"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:`}
                    />
                  </div>

                  {/* Quick Start */}
                  <div className="p-4 rounded-lg bg-primary/10 border border-primary/30">
                    <p className="text-sm font-semibold mb-3 flex items-center gap-2">
                      <Link className="w-4 h-4 text-primary" />
                      {t("selfHost.database.quickStart")}
                    </p>
                    <ol className="space-y-2 text-sm text-foreground/80">
                      <li className="flex gap-2">
                        <span className="font-semibold">1.</span>
                        <span>{t("selfHost.database.quickStartStep1")}</span>
                      </li>
                      <li className="flex gap-2">
                        <span className="font-semibold">2.</span>
                        <span>{t("selfHost.database.quickStartStep2")}</span>
                      </li>
                      <li className="flex gap-2">
                        <span className="font-semibold">3.</span>
                        <span>{t("selfHost.database.quickStartStep3")}</span>
                      </li>
                      <li className="flex gap-2">
                        <span className="font-semibold">4.</span>
                        <span>{t("selfHost.database.quickStartStep4")}</span>
                      </li>
                    </ol>
                  </div>
                </div>
              </SectionCard>

              {/* Support Section */}
              <Card className="bg-gradient-to-br from-primary/5 to-primary/10 border-primary/20">
                <CardContent className="p-6">
                  <h3 className="text-xl font-semibold mb-3 flex items-center gap-2">
                    <Link className="w-5 h-5" />
                    {t("selfHost.support.title")}
                  </h3>
                  <p className="text-foreground/80 mb-4">
                    {t("selfHost.support.description")}
                  </p>
                  <div className="flex flex-wrap gap-3">
                    <a
                      href="https://github.com/Eric-Wahl/Chronos-Reminder"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-background hover:bg-muted border border-border transition-colors"
                    >
                      <Code className="w-4 h-4" />
                      {t("selfHost.support.github")}
                    </a>
                    <a
                      href="https://discord.gg/m3MsM922QD"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-background hover:bg-muted border border-border transition-colors"
                    >
                      <Link className="w-4 h-4" />
                      {t("selfHost.support.discord")}
                    </a>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </>
  );
}
