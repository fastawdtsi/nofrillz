import AIStudio from "./AIStudio";
import { FormEvent, ReactNode, startTransition, useDeferredValue, useEffect, useState } from "react";
import {
  ApiError,
  blockUser,
  deletePost,
  getStats,
  listPosts,
  listUsers,
} from "./api";
import type {
  AccountType,
  AdminPost,
  AdminUser,
  ConnectionSettings,
  Stats,
} from "./types";

const defaultSettings: ConnectionSettings = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:3000",
  adminApiKey: import.meta.env.VITE_ADMIN_API_KEY ?? "",
};

const pageIds = ["dashboard", "users", "posts", "ai-studio", "settings"] as const;
const primaryPageIds = ["dashboard", "users", "posts", "ai-studio"] as const;

type PageId = (typeof pageIds)[number];

type Flash = {
  tone: "success" | "error";
  text: string;
} | null;

type ToneName = "blue" | "red" | "green" | "yellow" | "orange" | "purple" | "teal";

const pageMeta: Record<
  PageId,
  {
    nav: string;
    caption: string;
    eyebrow: string;
    title: string;
    description: string;
  }
> = {
  dashboard: {
    nav: "Main Dashboard",
    caption: "Landing page",
    eyebrow: "Landing page",
    title: "Main dashboard",
    description: "A live internal snapshot for moderation, account volume, and AI operations.",
  },
  users: {
    nav: "Users",
    caption: "Accounts and moderation",
    eyebrow: "People and profiles",
    title: "Users",
    description: "Search, inspect, and block accounts without leaving the portal.",
  },
  posts: {
    nav: "Posts",
    caption: "Moderation queue",
    eyebrow: "Content operations",
    title: "Posts",
    description: "Review recent content, inspect the author, and delete posts when needed.",
  },
  "ai-studio": {
    nav: "AI Studio",
    caption: "AI creation tools",
    eyebrow: "Automation workspace",
    title: "AI Studio",
    description: "Configure content missions, research sources, and model variants.",
  },
  settings: {
    nav: "Settings",
    caption: "Connection and portal setup",
    eyebrow: "Portal configuration",
    title: "Settings",
    description: "Manage the API URL, admin key, and connection behavior for this internal tool.",
  },
};

const toneClasses: Record<
  ToneName,
  {
    active: string;
    avatar: string;
    badge: string;
    card: string;
    dot: string;
    filled: string;
    hover: string;
    outline: string;
    value: string;
  }
> = {
  blue: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--blue)/0.18)] bg-[rgb(var(--blue-soft)/0.92)] text-[rgb(var(--blue))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--blue))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
  red: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--red)/0.18)] bg-[rgb(var(--red-soft)/0.92)] text-[rgb(var(--red))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--red))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--red)/0.28)] text-[rgb(var(--red))] hover:bg-[rgb(var(--red-soft)/0.55)]",
    value: "text-[rgb(var(--ink))]",
  },
  green: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--green)/0.18)] bg-[rgb(var(--green-soft)/0.92)] text-[rgb(var(--green))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--green))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
  yellow: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--yellow)/0.2)] bg-[rgb(var(--yellow-soft)/0.92)] text-[rgb(var(--yellow))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--yellow))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
  orange: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--orange)/0.18)] bg-[rgb(var(--orange-soft)/0.92)] text-[rgb(var(--orange))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--orange))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
  purple: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--purple)/0.18)] bg-[rgb(var(--purple-soft)/0.92)] text-[rgb(var(--purple))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--purple))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
  teal: {
    active:
      "border-[rgb(var(--red)/0.24)] bg-[rgb(var(--red-soft)/0.34)] shadow-[0_18px_36px_rgb(var(--shadow)/0.08)]",
    avatar: "bg-[rgb(var(--surface-muted))] text-[rgb(var(--ink))]",
    badge:
      "border-[rgb(var(--teal)/0.18)] bg-[rgb(var(--teal-soft)/0.92)] text-[rgb(var(--teal))]",
    card:
      "border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.58)]",
    dot: "bg-[rgb(var(--teal))]",
    filled:
      "bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))] hover:opacity-92",
    hover: "hover:border-[rgb(var(--line)/0.62)]",
    outline:
      "border-[rgb(var(--line)/0.72)] text-[rgb(var(--ink))] hover:border-[rgb(var(--line)/0.9)] hover:bg-[rgb(var(--surface-muted)/0.65)]",
    value: "text-[rgb(var(--ink))]",
  },
};

function toneForPage(page: PageId): ToneName {
  switch (page) {
    case "dashboard":
      return "blue";
    case "users":
      return "teal";
    case "posts":
      return "orange";
    case "ai-studio":
      return "purple";
    case "settings":
      return "yellow";
    default:
      return "blue";
  }
}

function toneForAccountType(accountType?: AccountType): ToneName {
  switch (accountType) {
    case "ai":
      return "purple";
    case "system":
      return "orange";
    case "human":
    default:
      return "teal";
  }
}

function toneForSource(source?: AdminPost["source"]): ToneName {
  switch (source) {
    case "ai":
      return "purple";
    case "system":
      return "orange";
    case "human":
    default:
      return "blue";
  }
}

function readLocalStorage<T>(key: string, fallback: T) {
  if (typeof window === "undefined") {
    return fallback;
  }

  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) {
      return fallback;
    }
    return { ...fallback, ...JSON.parse(raw) };
  } catch {
    return fallback;
  }
}

function readStoredTheme() {
  if (typeof window === "undefined") {
    return "dark";
  }

  const stored = window.localStorage.getItem("nofrillz-admin-theme");
  return stored === "light" || stored === "dark" ? stored : "dark";
}

function isPageId(value: string): value is PageId {
  return pageIds.some((page) => page === value);
}

function readPageFromHash() {
  if (typeof window === "undefined") {
    return "dashboard" as PageId;
  }

  const value = window.location.hash.replace(/^#\/?/, "").trim().toLowerCase();
  return isPageId(value) ? value : "dashboard";
}

function writePageHash(page: PageId) {
  if (typeof window === "undefined") {
    return;
  }

  const nextHash = `#${page}`;
  if (window.location.hash !== nextHash) {
    window.location.hash = nextHash;
  }
}

function formatDate(value?: string) {
  if (!value) {
    return "Just now";
  }

  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatCount(value?: number) {
  return new Intl.NumberFormat().format(value ?? 0);
}

function getErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Something went wrong.";
}

function initialsForUser(user: {
  first_name?: string;
  last_name?: string;
  username?: string;
}) {
  const first = user.first_name?.trim().charAt(0) ?? "";
  const last = user.last_name?.trim().charAt(0) ?? "";
  if (first || last) {
    return `${first}${last}`.toUpperCase();
  }
  return user.username?.slice(0, 2).toUpperCase() || "NF";
}

function Avatar({
  user,
}: {
  user: {
    account_type?: AccountType;
    first_name?: string;
    last_name?: string;
    username?: string;
    avatar_url?: string;
  };
}) {
  const [imageFailed, setImageFailed] = useState(false);

  useEffect(() => {
    setImageFailed(false);
  }, [user.avatar_url]);

  if (user.avatar_url && !imageFailed) {
    return (
      <img
        alt={user.username ?? "avatar"}
        className="h-10 w-10 rounded-full border border-[rgb(var(--line)/0.55)] object-cover shadow-sm"
        onError={() => setImageFailed(true)}
        src={user.avatar_url}
      />
    );
  }

  return (
    <div
      className="flex h-10 w-10 items-center justify-center rounded-full border border-[rgb(var(--line)/0.55)] bg-[rgb(var(--surface-muted))] text-sm text-[rgb(var(--ink))]"
      style={{
        fontFamily:
          '"Snell Roundhand", "Apple Chancery", "Brush Script MT", "Segoe Script", cursive',
      }}
    >
      {initialsForUser(user)}
    </div>
  );
}

function StatCard({
  label,
  value,
  note,
  tone,
}: {
  label: string;
  value: string;
  note: string;
  tone: ToneName;
}) {
  const toneStyle = toneClasses[tone];

  return (
    <div className="rounded-[14px] border border-[rgb(var(--line)/0.48)] bg-[rgb(var(--surface)/0.72)] p-5 shadow-[0_18px_48px_rgb(var(--shadow)/0.08)] backdrop-blur">
      <div className="inline-flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.32em] text-[rgb(var(--muted))]">
        <span className={`h-2.5 w-2.5 rounded-full ${toneStyle.dot}`} />
        {label}
      </div>
      <div className={`mt-3 text-3xl font-semibold tracking-[-0.03em] ${toneStyle.value}`}>
        {value}
      </div>
      <div className="mt-2 text-sm text-[rgb(var(--muted))]">{note}</div>
    </div>
  );
}

function Section({
  title,
  subtitle,
  children,
  actions,
}: {
  title: string;
  subtitle: string;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <section className="rounded-[16px] border border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface)/0.76)] p-6 shadow-[0_24px_70px_rgb(var(--shadow)/0.12)] backdrop-blur md:p-7">
      <div className="flex flex-col gap-4 border-b border-[rgb(var(--line)/0.7)] pb-5 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-[-0.03em] text-[rgb(var(--ink))]">
            {title}
          </h2>
          <p className="mt-1 text-sm leading-6 text-[rgb(var(--muted))]">{subtitle}</p>
        </div>
        {actions}
      </div>
      <div className="pt-5">{children}</div>
    </section>
  );
}

function SidebarButton({
  active,
  label,
  caption,
  onClick,
  tone,
}: {
  active: boolean;
  label: string;
  caption: string;
  onClick: () => void;
  tone: ToneName;
}) {
  const toneStyle = toneClasses[tone];

  return (
    <button
      className={`w-full rounded-[16px] border px-4 py-3 text-left transition ${
        active
          ? "border-[rgb(var(--line)/0.72)] bg-[rgb(var(--surface-muted))] shadow-[0_18px_36px_rgb(var(--shadow)/0.06)]"
          : `border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.54)] ${toneStyle.hover} hover:bg-[rgb(var(--surface)/0.9)]`
      }`}
      onClick={onClick}
      type="button"
    >
      <div className={`text-sm font-semibold ${active ? "text-[rgb(var(--ink))]" : "text-[rgb(var(--ink))]"}`}>
        {label}
      </div>
      <div
        className={`mt-1 text-xs uppercase tracking-[0.22em] ${
          active ? "text-[rgb(var(--muted))]" : "text-[rgb(var(--muted))]"
        }`}
      >
        {caption}
      </div>
    </button>
  );
}

function PageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow: string;
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <header className="rounded-[16px] border border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface)/0.72)] px-5 py-5 shadow-[0_24px_70px_rgb(var(--shadow)/0.12)] backdrop-blur md:px-7">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-3xl">
          <div className="text-[11px] font-semibold uppercase tracking-[0.34em] text-[rgb(var(--muted))]">
            {eyebrow}
          </div>
          <h1 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[rgb(var(--ink))] sm:text-4xl">
            {title}
          </h1>
          <p className="mt-3 max-w-2xl text-sm leading-7 text-[rgb(var(--muted))] sm:text-base">
            {description}
          </p>
        </div>
        {actions}
      </div>
    </header>
  );
}

function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body: string;
  action?: ReactNode;
}) {
  return (
    <div className="rounded-[14px] border border-dashed border-[rgb(var(--line)/0.9)] bg-[rgb(var(--surface)/0.64)] px-6 py-10 text-center">
      <div className="text-lg font-semibold tracking-[-0.02em] text-[rgb(var(--ink))]">{title}</div>
      <p className="mx-auto mt-3 max-w-xl text-sm leading-7 text-[rgb(var(--muted))]">{body}</p>
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}

function ConnectionBadge({
  hasConnection,
  loading,
  hasError,
}: {
  hasConnection: boolean;
  loading: boolean;
  hasError: boolean;
}) {
  let label = "Connected";
  let className = toneClasses.green.badge;

  if (!hasConnection) {
    label = "Not configured";
    className = toneClasses.yellow.badge;
  } else if (hasError) {
    label = "Needs attention";
    className = toneClasses.red.badge;
  } else if (loading) {
    label = "Syncing";
    className = toneClasses.blue.badge;
  }

  return (
    <div className={`rounded-full border px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] ${className}`}>
      {label}
    </div>
  );
}

function ToneBadge({
  label,
  tone,
}: {
  label: string;
  tone: ToneName;
}) {
  return (
    <span
      className={`inline-flex rounded-full border px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.22em] ${toneClasses[tone].badge}`}
    >
      {label}
    </span>
  );
}

function App() {
  const [activePage, setActivePage] = useState<PageId>(() => readPageFromHash());
  const [settings, setSettings] = useState<ConnectionSettings>(() =>
    readLocalStorage("nofrillz-admin-settings", defaultSettings),
  );
  const [draftSettings, setDraftSettings] = useState<ConnectionSettings>(() =>
    readLocalStorage("nofrillz-admin-settings", defaultSettings),
  );
  const [theme, setTheme] = useState<"light" | "dark">(() => readStoredTheme());
  const [flash, setFlash] = useState<Flash>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [posts, setPosts] = useState<AdminPost[]>([]);
  const [recentUserPosts, setRecentUserPosts] = useState<AdminPost[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedPostId, setSelectedPostId] = useState<string | null>(null);
  const [aiUsers, setAiUsers] = useState<AdminUser[]>([]);
  const [userSearch, setUserSearch] = useState("");
  const [userFilter, setUserFilter] = useState<"" | "human" | "ai" | "system">("");
  const [postFilter, setPostFilter] = useState<"" | "human" | "ai" | "system">("");
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [loadingPosts, setLoadingPosts] = useState(false);
  const [loadingSelectedPosts, setLoadingSelectedPosts] = useState(false);
  const [refreshTick, setRefreshTick] = useState(0);

  const deferredUserSearch = useDeferredValue(userSearch.trim());
  const hasConnection = settings.baseUrl.trim() !== "" && settings.adminApiKey.trim() !== "";
  const isBusy = loadingOverview || loadingUsers || loadingPosts || loadingSelectedPosts;
  const currentPage = pageMeta[activePage];
  const selectedUser = users.find((user) => user.id === selectedUserId) ?? null;
  const filteredPosts =
    postFilter === "" ? posts : posts.filter((post) => post.source === postFilter);
  const selectedPost = filteredPosts.find((post) => post.id === selectedPostId) ?? filteredPosts[0] ?? null;
  const settingsDirty =
    draftSettings.baseUrl !== settings.baseUrl ||
    draftSettings.adminApiKey !== settings.adminApiKey;
  const keyConfigured = settings.adminApiKey.trim() !== "";
  const hasLoadError = flash?.tone === "error";
  const dashboardPosts = posts.slice(0, 5);
  const featuredAIUsers = aiUsers.slice(0, 4);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const syncPage = () => setActivePage(readPageFromHash());
    if (!window.location.hash) {
      writePageHash("dashboard");
    }
    syncPage();

    window.addEventListener("hashchange", syncPage);
    return () => window.removeEventListener("hashchange", syncPage);
  }, []);

  useEffect(() => {
    window.localStorage.setItem("nofrillz-admin-settings", JSON.stringify(settings));
  }, [settings]);

  useEffect(() => {
    window.localStorage.setItem("nofrillz-admin-theme", theme);
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  useEffect(() => {
    if (!hasConnection) {
      setStats(null);
      setAiUsers([]);
      return;
    }

    let cancelled = false;
    setLoadingOverview(true);

    void (async () => {
      try {
        const [statsResponse, aiUsersResponse] = await Promise.all([
          getStats(settings),
          listUsers(settings, { accountType: "ai", limit: 50 }),
        ]);

        if (cancelled) {
          return;
        }

        const availableAIUsers = aiUsersResponse.users.filter((user) => user.ai_account_id);
        setStats(statsResponse);
        setAiUsers(availableAIUsers);
        setFlash((current) => (current?.tone === "error" ? null : current));

      } catch (error) {
        if (!cancelled) {
          setFlash({ tone: "error", text: getErrorMessage(error) });
        }
      } finally {
        if (!cancelled) {
          setLoadingOverview(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [hasConnection, refreshTick, settings]);

  useEffect(() => {
    if (!hasConnection) {
      setUsers([]);
      setSelectedUserId(null);
      return;
    }

    let cancelled = false;
    setLoadingUsers(true);

    void (async () => {
      try {
        const response = await listUsers(settings, {
          q: deferredUserSearch || undefined,
          accountType: userFilter || undefined,
          limit: 25,
        });

        if (cancelled) {
          return;
        }

        setUsers(response.users);
        setFlash((current) => (current?.tone === "error" ? null : current));
        startTransition(() => {
          setSelectedUserId((current) =>
            response.users.some((user) => user.id === current)
              ? current
              : (response.users[0]?.id ?? null),
          );
        });
      } catch (error) {
        if (!cancelled) {
          setFlash({ tone: "error", text: getErrorMessage(error) });
        }
      } finally {
        if (!cancelled) {
          setLoadingUsers(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [deferredUserSearch, hasConnection, refreshTick, settings, userFilter]);

  useEffect(() => {
    if (!hasConnection) {
      setPosts([]);
      setSelectedPostId(null);
      return;
    }

    let cancelled = false;
    setLoadingPosts(true);

    void (async () => {
      try {
        const response = await listPosts(settings, { limit: 30 });

        if (cancelled) {
          return;
        }

        setPosts(response.posts);
        setFlash((current) => (current?.tone === "error" ? null : current));
        startTransition(() => {
          setSelectedPostId((current) =>
            response.posts.some((post) => post.id === current)
              ? current
              : (response.posts[0]?.id ?? null),
          );
        });
      } catch (error) {
        if (!cancelled) {
          setFlash({ tone: "error", text: getErrorMessage(error) });
        }
      } finally {
        if (!cancelled) {
          setLoadingPosts(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [hasConnection, refreshTick, settings]);

  useEffect(() => {
    if (!hasConnection || selectedUserId === null) {
      setRecentUserPosts([]);
      return;
    }

    let cancelled = false;
    setLoadingSelectedPosts(true);

    void (async () => {
      try {
        const response = await listPosts(settings, { userId: selectedUserId, limit: 8 });
        if (!cancelled) {
          setRecentUserPosts(response.posts);
          setFlash((current) => (current?.tone === "error" ? null : current));
        }
      } catch (error) {
        if (!cancelled) {
          setFlash({ tone: "error", text: getErrorMessage(error) });
        }
      } finally {
        if (!cancelled) {
          setLoadingSelectedPosts(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [hasConnection, refreshTick, selectedUserId, settings]);

  function navigate(page: PageId) {
    setActivePage(page);
    writePageHash(page);
  }

  function triggerRefresh(message?: string) {
    if (message) {
      setFlash({ tone: "success", text: message });
    }
    startTransition(() => {
      setRefreshTick((value) => value + 1);
    });
  }

  function openPost(postId: string) {
    startTransition(() => {
      setSelectedPostId(postId);
      navigate("posts");
    });
  }

  function openUser(userId: string, username?: string, accountType?: AdminUser["account_type"]) {
    if (username) {
      setUserSearch(username);
    }
    if (accountType) {
      setUserFilter(accountType);
    }

    startTransition(() => {
      setSelectedUserId(userId);
      navigate("users");
    });
  }

  async function handleSaveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSettings(draftSettings);
    setFlash({
      tone: "success",
      text:
        draftSettings.baseUrl.trim() && draftSettings.adminApiKey.trim()
          ? "Connection settings saved. Loading admin data."
          : "Saved connection settings updated.",
    });
  }

  function handleResetDraft() {
    setDraftSettings(settings);
    setFlash({ tone: "success", text: "Reverted unsaved settings." });
  }

  function handleClearSavedSettings() {
    const clearedSettings = {
      ...defaultSettings,
      adminApiKey: "",
    };

    setSettings(clearedSettings);
    setDraftSettings(clearedSettings);
    setFlash({ tone: "success", text: "Cleared saved connection settings." });
  }

  async function handleBlockUser(user: AdminUser) {
    const confirmed = window.confirm(`Block @${user.username}?`);
    if (!confirmed) {
      return;
    }

    try {
      await blockUser(settings, user.id);
      triggerRefresh(`Blocked @${user.username}.`);
    } catch (error) {
      setFlash({ tone: "error", text: getErrorMessage(error) });
    }
  }

  async function handleDeletePost(post: AdminPost) {
    const confirmed = window.confirm(`Delete post #${post.id} by @${post.user.username}?`);
    if (!confirmed) {
      return;
    }

    try {
      await deletePost(settings, post.id);
      triggerRefresh(`Deleted post #${post.id}.`);
    } catch (error) {
      setFlash({ tone: "error", text: getErrorMessage(error) });
    }
  }

  function renderConnectionGate() {
    return (
      <EmptyState
        action={
          <button
            className={`rounded-full px-5 py-3 text-sm font-medium transition ${toneClasses.yellow.filled}`}
            onClick={() => navigate("settings")}
            type="button"
          >
            Open settings
          </button>
        }
        body="Save the admin API URL and key in Settings before loading data on this page. Requests only start after you save, so the portal will not interrupt you while you type."
        title="Connect the portal first"
      />
    );
  }

  function renderDashboardPage() {
    if (!hasConnection) {
      return renderConnectionGate();
    }

    return (
      <div className="space-y-5">
        <Section
          actions={
            <div className="text-xs font-medium uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
              {loadingOverview ? "Refreshing..." : "Live admin snapshot"}
            </div>
          }
          subtitle="Core volume and AI activity at a glance."
          title="Overview"
        >
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            <StatCard
              label="Users"
              note="Active, visible accounts"
              tone="blue"
              value={formatCount(stats?.user_count)}
            />
            <StatCard
              label="Human Accounts"
              note="Standard users"
              tone="green"
              value={formatCount(stats?.human_user_count)}
            />
            <StatCard
              label="AI Accounts"
              note="AI-owned users"
              tone="purple"
              value={formatCount(stats?.ai_user_count)}
            />
            <StatCard
              label="Blocked Users"
              note="Hidden from public flows"
              tone="red"
              value={formatCount(stats?.blocked_user_count)}
            />
            <StatCard
              label="Posts"
              note="Visible post volume"
              tone="orange"
              value={formatCount(stats?.post_count)}
            />
            <StatCard
              label="AI Output"
              note="AI posts plus enabled AI accounts"
              tone="teal"
              value={`${formatCount(stats?.ai_post_count)} / ${formatCount(
                stats?.enabled_ai_account_count,
              )}`}
            />
          </div>
        </Section>

        <div className="grid gap-5 xl:grid-cols-[1.1fr,0.9fr]">
          <Section
            actions={
              <button
                className={`rounded-full border px-4 py-2 text-sm font-medium transition ${toneClasses.orange.outline}`}
                onClick={() => navigate("posts")}
                type="button"
              >
                Open posts
              </button>
            }
            subtitle="A quick moderation view into the latest live content."
            title="Recent moderation queue"
          >
            <div className="space-y-3">
              {dashboardPosts.length === 0 && !loadingPosts ? (
                <div className="rounded-[14px] border border-dashed px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                  No recent posts available.
                </div>
              ) : null}

              {dashboardPosts.map((post) => (
                <button
                  className={`w-full rounded-[16px] border bg-[rgb(var(--surface-muted)/0.82)] p-4 text-left transition hover:bg-[rgb(var(--surface)/0.92)] ${toneClasses.orange.hover}`}
                  key={post.id}
                  onClick={() => openPost(post.id)}
                  type="button"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 items-start gap-3">
                      <Avatar user={post.user} />
                      <div className="min-w-0">
                        <div className="text-sm font-semibold text-[rgb(var(--ink))]">
                          @{post.user.username}
                        </div>
                        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs uppercase tracking-[0.24em] text-[rgb(var(--muted))]">
                          <ToneBadge label={post.source} tone={toneForSource(post.source)} />
                          <span>{formatDate(post.created_at)}</span>
                        </div>
                        <p className="mt-3 text-sm leading-6 text-[rgb(var(--ink))]">{post.body}</p>
                      </div>
                    </div>
                    <div className="shrink-0 text-xs font-semibold uppercase tracking-[0.18em] text-[rgb(var(--red))]">
                      View
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </Section>

          <div className="space-y-5">
            <Section
              subtitle="Jump directly into the part of the portal you need."
              title="Admin lanes"
            >
              <div className="grid gap-3 sm:grid-cols-2">
                <button
                  className={`rounded-[16px] border p-4 text-left transition ${toneClasses.teal.card} hover:bg-[rgb(var(--surface)/0.9)]`}
                  onClick={() => navigate("users")}
                  type="button"
                >
                  <div className={`text-sm font-semibold ${toneClasses.teal.value}`}>Users</div>
                  <p className="mt-2 text-sm leading-6 text-[rgb(var(--muted))]">
                    Search accounts, inspect profile details, and block users.
                  </p>
                </button>
                <button
                  className={`rounded-[16px] border p-4 text-left transition ${toneClasses.orange.card} hover:bg-[rgb(var(--surface)/0.9)]`}
                  onClick={() => navigate("posts")}
                  type="button"
                >
                  <div className={`text-sm font-semibold ${toneClasses.orange.value}`}>Posts</div>
                  <p className="mt-2 text-sm leading-6 text-[rgb(var(--muted))]">
                    Review the latest content and remove posts from the live surface.
                  </p>
                </button>
                <button
                  className={`rounded-[16px] border p-4 text-left transition ${toneClasses.purple.card} hover:bg-[rgb(var(--surface)/0.9)]`}
                  onClick={() => navigate("ai-studio")}
                  type="button"
                >
                  <div className={`text-sm font-semibold ${toneClasses.purple.value}`}>AI Studio</div>
                  <p className="mt-2 text-sm leading-6 text-[rgb(var(--muted))]">
                    Create AI content accounts, draft output, and publish generated posts.
                  </p>
                </button>
                <button
                  className={`rounded-[16px] border p-4 text-left transition ${toneClasses.yellow.card} hover:bg-[rgb(var(--surface)/0.9)]`}
                  onClick={() => navigate("settings")}
                  type="button"
                >
                  <div className={`text-sm font-semibold ${toneClasses.yellow.value}`}>Settings</div>
                  <p className="mt-2 text-sm leading-6 text-[rgb(var(--muted))]">
                    Update the API URL, admin key, and portal connection behavior.
                  </p>
                </button>
              </div>
            </Section>

            <Section
              subtitle="Quick context on AI account coverage across the system."
              title="AI footprint"
            >
              {featuredAIUsers.length === 0 ? (
                <div className="rounded-[14px] border border-dashed px-4 py-8 text-center text-sm text-[rgb(var(--muted))]">
                  No AI accounts are configured yet. Create one in AI Studio when you are ready.
                </div>
              ) : (
                <div className="space-y-3">
                  {featuredAIUsers.map((user) => (
                    <button
                      className={`flex w-full items-center justify-between rounded-[14px] border bg-[rgb(var(--surface-muted)/0.8)] px-4 py-3 text-left transition hover:bg-[rgb(var(--surface)/0.92)] ${toneClasses.purple.hover}`}
                      key={user.id}
                      onClick={() => openUser(user.id, user.username, user.account_type)}
                      type="button"
                    >
                      <div className="flex items-center gap-3">
                        <Avatar user={user} />
                        <div>
                          <div className="text-sm font-semibold text-[rgb(var(--ink))]">
                            @{user.username}
                          </div>
                          <div className="mt-1">
                            <ToneBadge label={`AI #${user.ai_account_id}`} tone="purple" />
                          </div>
                        </div>
                      </div>
                      <div className="text-xs text-[rgb(var(--muted))]">{formatCount(user.post_count)} posts</div>
                    </button>
                  ))}
                </div>
              )}
            </Section>
          </div>
        </div>
      </div>
    );
  }

  function renderUsersPage() {
    if (!hasConnection) {
      return renderConnectionGate();
    }

    return (
      <Section
        actions={
          <div className="flex flex-col gap-3 sm:flex-row">
            <input
              className="w-full rounded-[14px] border bg-[rgb(var(--surface-muted)/0.85)] px-4 py-3 text-sm text-[rgb(var(--ink))] outline-none placeholder:text-[rgb(var(--muted))] sm:w-64"
              onChange={(event) => setUserSearch(event.target.value)}
              placeholder="Search email or username"
              value={userSearch}
            />
            <select
              className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.85)] px-4 py-3 text-sm text-[rgb(var(--ink))] outline-none"
              onChange={(event) =>
                setUserFilter(event.target.value as "" | "human" | "ai" | "system")
              }
              value={userFilter}
            >
              <option value="">All accounts</option>
              <option value="human">Human</option>
              <option value="ai">AI</option>
              <option value="system">System</option>
            </select>
          </div>
        }
        subtitle="Search, inspect, and block accounts from one focused workspace."
        title="Accounts"
      >
        <div className="grid gap-5 lg:grid-cols-[1fr,0.98fr]">
          <div className="space-y-3">
            {loadingUsers ? (
              <div className="rounded-[16px] border border-dashed border-[rgb(var(--line)/0.85)] px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                Loading users...
              </div>
            ) : null}

            {!loadingUsers && users.length === 0 ? (
              <div className="rounded-[16px] border border-dashed border-[rgb(var(--line)/0.85)] px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                No users match the current filter.
              </div>
            ) : null}

            {users.map((user) => (
              <button
                className={`w-full rounded-[12px] border p-4 text-left transition ${
                  selectedUserId === user.id
                    ? toneClasses.teal.active
                    : `bg-[rgb(var(--surface-muted)/0.78)] ${toneClasses.teal.hover} hover:bg-[rgb(var(--surface)/0.88)]`
                }`}
                key={user.id}
                onClick={() => startTransition(() => setSelectedUserId(user.id))}
                type="button"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex min-w-0 items-start gap-3">
                    <Avatar user={user} />
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="truncate text-base font-semibold text-[rgb(var(--ink))]">
                          @{user.username}
                        </span>
                        <ToneBadge
                          label={user.account_type}
                          tone={toneForAccountType(user.account_type)}
                        />
                        {user.blocked_at ? (
                          <ToneBadge label="blocked" tone="red" />
                        ) : null}
                      </div>
                      <div className="mt-1 text-sm text-[rgb(var(--muted))]">
                        {[user.first_name, user.last_name].filter(Boolean).join(" ")}{" "}
                        <span className="mx-1">•</span> {user.email}
                      </div>
                      {user.about ? (
                        <p className="mt-3 text-sm leading-6 text-[rgb(var(--muted))]">{user.about}</p>
                      ) : null}
                    </div>
                  </div>
                  <div className="text-right text-xs text-[rgb(var(--muted))]">
                    <div>{formatCount(user.post_count)} posts</div>
                    <div className="mt-1">{formatDate(user.created_at)}</div>
                  </div>
                </div>
              </button>
            ))}
          </div>

          <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.78)] p-5">
            {selectedUser ? (
              <>
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex items-start gap-3">
                    <Avatar user={selectedUser} />
                    <div>
                      <div className="text-lg font-semibold tracking-[-0.03em] text-[rgb(var(--ink))]">
                        @{selectedUser.username}
                      </div>
                      <div className="mt-1 text-sm text-[rgb(var(--muted))]">
                        {selectedUser.email}
                      </div>
                      <div className="mt-3 flex flex-wrap items-center gap-2">
                        <ToneBadge
                          label={selectedUser.account_type}
                          tone={toneForAccountType(selectedUser.account_type)}
                        />
                        {selectedUser.ai_account_id ? (
                          <ToneBadge label={`AI #${selectedUser.ai_account_id}`} tone="purple" />
                        ) : null}
                        {selectedUser.blocked_at ? <ToneBadge label="blocked" tone="red" /> : null}
                      </div>
                    </div>
                  </div>

                  <button
                    className="rounded-full border border-[rgb(var(--danger)/0.25)] px-4 py-2 text-sm font-medium text-[rgb(var(--danger))] transition hover:bg-[rgb(var(--danger)/0.08)] disabled:cursor-not-allowed disabled:opacity-45"
                    disabled={Boolean(selectedUser.blocked_at)}
                    onClick={() => void handleBlockUser(selectedUser)}
                    type="button"
                  >
                    {selectedUser.blocked_at ? "Blocked" : "Block user"}
                  </button>
                </div>

                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-[14px] border bg-[rgb(var(--surface)/0.92)] px-4 py-4">
                    <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                      User ID
                    </div>
                    <div className="mt-2 text-sm text-[rgb(var(--ink))]">{selectedUser.id}</div>
                  </div>
                  <div className="rounded-[14px] border bg-[rgb(var(--surface)/0.92)] px-4 py-4">
                    <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                      Joined
                    </div>
                    <div className="mt-2 text-sm text-[rgb(var(--ink))]">
                      {formatDate(selectedUser.created_at)}
                    </div>
                  </div>
                </div>

                <div className="mt-5 rounded-[16px] border border-[rgb(var(--line)/0.7)] bg-[rgb(var(--surface)/0.88)] px-4 py-4 text-sm leading-7 text-[rgb(var(--muted))]">
                  {selectedUser.about || "No bio on file for this account."}
                </div>

                <div className="mt-6">
                  <div className="flex items-center justify-between">
                    <div className="text-sm font-semibold uppercase tracking-[0.22em] text-[rgb(var(--muted))]">
                      Recent posts
                    </div>
                    {loadingSelectedPosts ? (
                      <div className="text-xs text-[rgb(var(--muted))]">Loading...</div>
                    ) : null}
                  </div>

                  <div className="mt-4 space-y-3">
                    {recentUserPosts.length === 0 && !loadingSelectedPosts ? (
                      <div className="rounded-[14px] border border-dashed px-4 py-8 text-center text-sm text-[rgb(var(--muted))]">
                        No recent posts for this user.
                      </div>
                    ) : null}

                    {recentUserPosts.map((post) => (
                      <div
                        className="rounded-[14px] border bg-[rgb(var(--surface)/0.94)] p-4"
                        key={post.id}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2 text-xs font-semibold uppercase tracking-[0.22em] text-[rgb(var(--muted))]">
                              <ToneBadge label={post.source} tone={toneForSource(post.source)} />
                              <span>{formatDate(post.created_at)}</span>
                            </div>
                            <p className="mt-2 text-sm leading-6 text-[rgb(var(--ink))]">{post.body}</p>
                          </div>
                          <button
                            className="shrink-0 rounded-full border border-[rgb(var(--danger)/0.25)] px-3 py-1.5 text-xs font-medium text-[rgb(var(--danger))] transition hover:bg-[rgb(var(--danger)/0.08)]"
                            onClick={() => void handleDeletePost(post)}
                            type="button"
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            ) : (
              <div className="rounded-[14px] border border-dashed px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                Select a user to inspect their posts and moderation actions.
              </div>
            )}
          </div>
        </div>
      </Section>
    );
  }

  function renderPostsPage() {
    if (!hasConnection) {
      return renderConnectionGate();
    }

    return (
      <Section
        actions={
          <div className="flex flex-col gap-3 sm:flex-row">
            <select
              className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.85)] px-4 py-3 text-sm text-[rgb(var(--ink))] outline-none"
              onChange={(event) =>
                setPostFilter(event.target.value as "" | "human" | "ai" | "system")
              }
              value={postFilter}
            >
              <option value="">All sources</option>
              <option value="human">Human</option>
              <option value="ai">AI</option>
              <option value="system">System</option>
            </select>
            <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-3 text-xs uppercase tracking-[0.22em] text-[rgb(var(--muted))]">
              {loadingPosts ? "Refreshing feed" : `${formatCount(filteredPosts.length)} loaded`}
            </div>
          </div>
        }
        subtitle="Inspect recent live content and remove posts when moderation needs a quick response."
        title="Moderation queue"
      >
        <div className="grid gap-5 lg:grid-cols-[1fr,0.96fr]">
          <div className="space-y-3">
            {loadingPosts ? (
              <div className="rounded-[16px] border border-dashed border-[rgb(var(--line)/0.85)] px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                Loading posts...
              </div>
            ) : null}

            {!loadingPosts && filteredPosts.length === 0 ? (
              <div className="rounded-[16px] border border-dashed border-[rgb(var(--line)/0.85)] px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                No posts match the current filter.
              </div>
            ) : null}

            {filteredPosts.map((post) => (
              <button
                className={`w-full rounded-[12px] border p-4 text-left transition ${
                  selectedPost?.id === post.id
                    ? toneClasses.orange.active
                    : `bg-[rgb(var(--surface-muted)/0.78)] ${toneClasses.orange.hover} hover:bg-[rgb(var(--surface)/0.88)]`
                }`}
                key={post.id}
                onClick={() => startTransition(() => setSelectedPostId(post.id))}
                type="button"
              >
                <div className="flex items-start gap-3">
                  <Avatar user={post.user} />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-base font-semibold text-[rgb(var(--ink))]">
                        @{post.user.username}
                      </span>
                      <ToneBadge label={post.source} tone={toneForSource(post.source)} />
                    </div>
                    <div className="mt-1 text-xs uppercase tracking-[0.22em] text-[rgb(var(--muted))]">
                      {formatDate(post.created_at)} • post #{post.id}
                    </div>
                    <p className="mt-3 text-sm leading-6 text-[rgb(var(--ink))]">{post.body}</p>
                  </div>
                </div>
              </button>
            ))}
          </div>

          <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.78)] p-5">
            {selectedPost ? (
              <>
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="flex items-start gap-3">
                    <Avatar user={selectedPost.user} />
                    <div>
                      <div className="text-lg font-semibold tracking-[-0.03em] text-[rgb(var(--ink))]">
                        @{selectedPost.user.username}
                      </div>
                      <div className="mt-1 text-sm text-[rgb(var(--muted))]">
                        {selectedPost.user.first_name} {selectedPost.user.last_name}
                      </div>
                      <div className="mt-3 flex flex-wrap items-center gap-2">
                        <ToneBadge
                          label={selectedPost.source}
                          tone={toneForSource(selectedPost.source)}
                        />
                        <ToneBadge
                          label={selectedPost.user.account_type}
                          tone={toneForAccountType(selectedPost.user.account_type)}
                        />
                      </div>
                    </div>
                  </div>

                  <button
                    className="rounded-full border border-[rgb(var(--danger)/0.25)] px-4 py-2 text-sm font-medium text-[rgb(var(--danger))] transition hover:bg-[rgb(var(--danger)/0.08)]"
                    onClick={() => void handleDeletePost(selectedPost)}
                    type="button"
                  >
                    Delete post
                  </button>
                </div>

                <div className="mt-5 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-[14px] border bg-[rgb(var(--surface)/0.92)] px-4 py-4">
                    <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                      Post ID
                    </div>
                    <div className="mt-2 text-sm text-[rgb(var(--ink))]">{selectedPost.id}</div>
                  </div>
                  <div className="rounded-[14px] border bg-[rgb(var(--surface)/0.92)] px-4 py-4">
                    <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                      Created
                    </div>
                    <div className="mt-2 text-sm text-[rgb(var(--ink))]">
                      {formatDate(selectedPost.created_at)}
                    </div>
                  </div>
                </div>

                <div className="mt-5 rounded-[16px] border bg-[rgb(var(--surface)/0.94)] px-4 py-5 text-sm leading-7 text-[rgb(var(--ink))]">
                  {selectedPost.body}
                </div>

                <div className="mt-5 flex flex-col gap-3 sm:flex-row">
                  <button
                    className={`rounded-full border px-4 py-2 text-sm font-medium transition ${toneClasses.teal.outline}`}
                    onClick={() =>
                      openUser(
                        selectedPost.user.user_id,
                        selectedPost.user.username,
                        selectedPost.user.account_type,
                      )
                    }
                    type="button"
                  >
                    View author
                  </button>
                  <div className="flex items-center">
                    <ToneBadge
                      label={`Source: ${selectedPost.source}`}
                      tone={toneForSource(selectedPost.source)}
                    />
                  </div>
                </div>
              </>
            ) : (
              <div className="rounded-[14px] border border-dashed px-4 py-10 text-center text-sm text-[rgb(var(--muted))]">
                Select a post to review its details and moderation actions.
              </div>
            )}
          </div>
        </div>
      </Section>
    );
  }

  function renderAIStudioPage() {
    if (!hasConnection) return renderConnectionGate();
    return <AIStudio settings={settings} refreshTick={refreshTick} onChanged={() => setRefreshTick((tick) => tick + 1)} />;
  }

  function renderSettingsPage() {
    return (
      <div className="space-y-5">
        <div className="grid gap-5 xl:grid-cols-[1.08fr,0.92fr]">
          <Section
            actions={
              settingsDirty ? (
                <div className={`rounded-full border px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] ${toneClasses.yellow.badge}`}>
                  Unsaved changes
                </div>
              ) : null
            }
            subtitle="Edit connection details here, then save once to apply them across the portal."
            title="Connection settings"
          >
            <form className="space-y-5" onSubmit={handleSaveSettings}>
              <label className="block rounded-[16px] border bg-[rgb(var(--surface-muted)/0.84)] px-4 py-4">
                <span className="text-[11px] font-semibold uppercase tracking-[0.28em] text-[rgb(var(--muted))]">
                  API URL
                </span>
                <input
                  className="mt-3 w-full bg-transparent text-sm text-[rgb(var(--ink))] outline-none placeholder:text-[rgb(var(--muted))]"
                  onChange={(event) =>
                    setDraftSettings((current) => ({ ...current, baseUrl: event.target.value }))
                  }
                  placeholder="http://localhost:3000"
                  value={draftSettings.baseUrl}
                />
              </label>

              <label className="block rounded-[16px] border bg-[rgb(var(--surface-muted)/0.84)] px-4 py-4">
                <span className="text-[11px] font-semibold uppercase tracking-[0.28em] text-[rgb(var(--muted))]">
                  Admin key
                </span>
                <input
                  className="mt-3 w-full bg-transparent text-sm text-[rgb(var(--ink))] outline-none placeholder:text-[rgb(var(--muted))]"
                  onChange={(event) =>
                    setDraftSettings((current) => ({
                      ...current,
                      adminApiKey: event.target.value,
                    }))
                  }
                  placeholder="Internal API key"
                  type="password"
                  value={draftSettings.adminApiKey}
                />
              </label>

              <div className="rounded-[16px] border bg-[rgb(var(--surface-muted)/0.68)] px-4 py-4 text-sm leading-7 text-[rgb(var(--muted))]">
                Requests only start after you save. That means you can paste or edit the URL and key
                here without the portal trying to reconnect on every keystroke.
              </div>

              <div className="flex flex-col gap-3 sm:flex-row">
                <button
                  className={`rounded-full px-5 py-3 text-sm font-medium transition ${toneClasses.blue.filled}`}
                  type="submit"
                >
                  Save and apply
                </button>
                <button
                  className={`rounded-full border px-5 py-3 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-45 ${toneClasses.blue.outline}`}
                  disabled={!settingsDirty}
                  onClick={handleResetDraft}
                  type="button"
                >
                  Revert draft
                </button>
                <button
                  className="rounded-full border border-[rgb(var(--danger)/0.25)] px-5 py-3 text-sm font-medium text-[rgb(var(--danger))] transition hover:bg-[rgb(var(--danger)/0.08)]"
                  onClick={handleClearSavedSettings}
                  type="button"
                >
                  Clear saved settings
                </button>
              </div>
            </form>
          </Section>

          <div className="space-y-5">
            <Section
              subtitle="The currently applied configuration that the portal is using right now."
              title="Active connection"
            >
              <div className="space-y-3">
                <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-4">
                  <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                    API URL
                  </div>
                  <div className="mt-2 text-sm leading-6 text-[rgb(var(--ink))]">
                    {settings.baseUrl || "Not set"}
                  </div>
                </div>
                <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-4">
                  <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                    Admin key
                  </div>
                  <div className="mt-2 text-sm leading-6 text-[rgb(var(--ink))]">
                    {keyConfigured ? "Saved locally for this portal." : "No key saved yet."}
                  </div>
                </div>
                <div className="rounded-[14px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-4">
                  <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                    Status
                  </div>
                  <div className="mt-3">
                    <ConnectionBadge
                      hasConnection={hasConnection}
                      hasError={Boolean(hasLoadError)}
                      loading={isBusy}
                    />
                  </div>
                </div>
              </div>
            </Section>

            <Section
              subtitle="A few quality-of-life notes for the internal workflow."
              title="Portal behavior"
            >
              <div className="space-y-4 text-sm leading-7 text-[rgb(var(--muted))]">
                <p>
                  Connection values are saved locally in this browser so the portal can reopen where
                  you left off.
                </p>
                <p>
                  Theme mode is separate from the API settings. You can switch light and dark mode at
                  the bottom of the left navigation.
                </p>
                <p>
                  Moderation pages stay focused on the data they manage, while Settings is the only
                  place that applies URL and key changes.
                </p>
              </div>
            </Section>
          </div>
        </div>
      </div>
    );
  }

  function renderPage() {
    switch (activePage) {
      case "dashboard":
        return renderDashboardPage();
      case "users":
        return renderUsersPage();
      case "posts":
        return renderPostsPage();
      case "ai-studio":
        return renderAIStudioPage();
      case "settings":
        return renderSettingsPage();
      default:
        return renderDashboardPage();
    }
  }

  return (
    <div className="mx-auto w-full max-w-[1600px] px-4 py-4 sm:px-6 lg:px-8">
      <div className="grid gap-5 lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="flex flex-col rounded-[16px] border border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface)/0.76)] p-4 shadow-[0_24px_70px_rgb(var(--shadow)/0.12)] backdrop-blur lg:sticky lg:top-4 lg:h-[calc(100vh-2rem)]">
          <div className="rounded-[14px] border border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.72)] px-4 py-5">
            <div className="text-[11px] font-semibold uppercase tracking-[0.36em] text-[rgb(var(--muted))]">
              NoFrillz Internal
            </div>
            <div className="mt-3 text-2xl font-semibold tracking-[-0.04em] text-[rgb(var(--ink))]">
              Admin portal
            </div>
            <p className="mt-3 text-sm leading-6 text-[rgb(var(--muted))]">
              Moderation, account tools, and AI operations in one clean control surface.
            </p>
            <div className="mt-4">
              <ConnectionBadge
                hasConnection={hasConnection}
                hasError={Boolean(hasLoadError)}
                loading={isBusy}
              />
            </div>
          </div>

          <nav className="mt-5 grid gap-2">
            {primaryPageIds.map((page) => (
              <SidebarButton
                active={activePage === page}
                caption={pageMeta[page].caption}
                key={page}
                label={pageMeta[page].nav}
                onClick={() => navigate(page)}
                tone={toneForPage(page)}
              />
            ))}
          </nav>

          <div className="mt-5 grid gap-2 lg:mt-auto">
            <SidebarButton
              active={activePage === "settings"}
              caption={pageMeta.settings.caption}
              label={pageMeta.settings.nav}
              onClick={() => navigate("settings")}
              tone={toneForPage("settings")}
            />
            <button
              className={`w-full rounded-[16px] border border-[rgb(var(--line)/0.45)] bg-[rgb(var(--surface-muted)/0.54)] px-4 py-3 text-left text-sm font-medium text-[rgb(var(--ink))] transition ${toneClasses.purple.hover} hover:bg-[rgb(var(--surface)/0.9)]`}
              onClick={() => setTheme((current) => (current === "dark" ? "light" : "dark"))}
              type="button"
            >
              {theme === "dark" ? "Light mode" : "Dark mode"}
            </button>
          </div>
        </aside>

        <main className="min-w-0 space-y-5">
          <PageHeader
            actions={
              activePage === "settings" ? (
                <div className="text-xs font-medium uppercase tracking-[0.26em] text-[rgb(var(--muted))]">
                  {settingsDirty ? "Draft changes pending" : "Applied configuration"}
                </div>
              ) : (
                <button
                  className={`rounded-full border px-4 py-2 text-sm font-medium transition ${toneClasses.yellow.outline}`}
                  onClick={() => navigate("settings")}
                  type="button"
                >
                  Open settings
                </button>
              )
            }
            description={currentPage.description}
            eyebrow={currentPage.eyebrow}
            title={currentPage.title}
          />

          {flash ? (
            <div
              className={`rounded-[16px] border px-4 py-3 text-sm ${
                flash.tone === "error"
                  ? "border-[rgb(var(--danger)/0.35)] bg-[rgb(var(--danger)/0.1)] text-[rgb(var(--danger))]"
                  : toneClasses.green.badge
              }`}
            >
              {flash.text}
            </div>
          ) : null}

          {renderPage()}
        </main>
      </div>
    </div>
  );
}

export default App;
