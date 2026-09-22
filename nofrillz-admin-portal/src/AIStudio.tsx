import { FormEvent, useEffect, useRef, useState } from "react";
import { createAIAccount, checkAIAccount, listAIContent, getAIAccount, getAIStatus, listUsers, updateAIAccount } from "./api";
import type { AIAccountForm, AIStatus, ContentItem, AdminUser, ConnectionSettings } from "./types";

const emptyForm: AIAccountForm = {
  content_mode: "generative", check_interval_seconds: 86400, exclusions: "", source_urls: [], source_max_age_hours: 168, model_options: ["openai"], default_model_option: "openai",
  email: "", username: "", first_name: "", last_name: "", about: "", enabled: true,
  topic: "", description: "", system_prompt: "", style_prompt: "", min_posts_per_day: 1, max_posts_per_day: 3,
};
const panel = "rounded-[18px] border bg-[rgb(var(--surface))] p-5 space-y-4";
const control = "w-full rounded-[12px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-3 text-sm disabled:opacity-50";
const button = "rounded-full border px-4 py-2 text-sm font-medium disabled:opacity-50 hover:bg-[rgb(var(--surface-muted))]";
const when = (value?: string | null) => value ? new Date(value).toLocaleString() : "—";
const message = (error: unknown) => error instanceof Error ? error.message : "Request failed";
type TextField = "email" | "username" | "first_name" | "about" | "topic" | "description" | "style_prompt" | "system_prompt" | "exclusions";
const fields: { key: TextField; label: string; multiline?: boolean; max: number; hint?: string }[] = [
  { key: "email", label: "Email", max: 254 }, { key: "username", label: "Handle", max: 50 },
  { key: "first_name", label: "Account name", max: 100 },
  { key: "about", label: "Public description", multiline: true, max: 1024 },
  { key: "topic", label: "Topic / beat", max: 128, hint: "The subject this account covers." },
  { key: "description", label: "Content mission", multiline: true, max: 4000, hint: "What useful content should this account provide?" },
  { key: "style_prompt", label: "Tone / style", multiline: true, max: 4000, hint: "The voice should serve the content mission." },
  { key: "exclusions", label: "Exclusions", multiline: true, max: 2000, hint: "Topics, claims, or formats to leave out." },
  { key: "system_prompt", label: "Additional instructions", multiline: true, max: 8000 },
];

export default function AIStudio({ settings, refreshTick, onChanged }: {
  settings: ConnectionSettings; refreshTick: number; onChanged: () => void;
}) {
  const [form, setForm] = useState<AIAccountForm>({ ...emptyForm });
  const [editing, setEditing] = useState<string | null>(null);
  const [roster, setRoster] = useState<AdminUser[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [pageCursor, setPageCursor] = useState<string>();
  const [status, setStatus] = useState<AIStatus>();
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [items, setItems] = useState<ContentItem[]>([]);
  const [tick, setTick] = useState(0);
  const loadVersion = useRef(0);
  const formPanel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let active = true;
    let inFlight = false;
    const load = async () => {
      if (inFlight) return;
      inFlight = true;
      const version = ++loadVersion.current;
      try {
        const [response, config] = await Promise.all([
          listUsers(settings, { accountType: "ai", limit: 25, cursor: pageCursor }), getAIStatus(settings),
        ]);
        if (active && version === loadVersion.current) {
          setRoster(response.users); setCursor(response.next_cursor); setStatus(config); setError("");
        }
      } catch (err) { if (active) setError(message(err)); }
      finally { inFlight = false; if (active) setLoading(false); }
    };
    setLoading(true);
    void load();
    const interval = window.setInterval(() => { if (!document.hidden) void load(); }, 10000);
    return () => { active = false; window.clearInterval(interval); };
  }, [settings, refreshTick, tick, pageCursor]);

  useEffect(() => {
    if (!editing) { setItems([]); return; }
    let active = true;
    const refresh = async () => { try { const result = await listAIContent(settings, editing); if (active) setItems(result.items); } catch (err) { if (active) setError(message(err)); } };
    void refresh(); const timer = window.setInterval(() => { if (!document.hidden) void refresh(); }, 10000);
    return () => { active = false; window.clearInterval(timer); };
  }, [editing, settings, tick]);

  async function action(work: () => Promise<void>) {
    setBusy(true); setError(""); setNotice("");
    try { await work(); setTick((value) => value + 1); onChanged(); }
    catch (err) { setError(message(err)); }
    finally { setBusy(false); }
  }
  function reset() { setEditing(null); setForm({ ...emptyForm }); setItems([]); }
  async function edit(user: AdminUser) {
    if (!user.ai_account_id) return;
    await action(async () => {
      const { account, user: profile } = await getAIAccount(settings, user.ai_account_id!);
      setEditing(account.id);
      setForm({ ...emptyForm, ...account, email: profile.email, username: profile.username, first_name: profile.first_name, last_name: profile.last_name,
        about: profile.about, enabled: account.enabled, topic: account.topic, description: account.description,
        system_prompt: account.system_prompt, style_prompt: account.style_prompt,
        min_posts_per_day: account.min_posts_per_day, max_posts_per_day: account.max_posts_per_day });
      setItems([]); formPanel.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    await action(async () => {
      if (form.model_options.length === 0) throw new Error("Select at least one model option.");
      if (editing) {
        const { email: _email, username: _username, ...changes } = form;
        await updateAIAccount(settings, editing, { ...changes, source_urls: form.source_urls.map((url) => url.trim()).filter(Boolean) });
        setNotice(`Saved @${form.username}. ${form.enabled ? "Next check scheduled." : "Autonomous posting paused."}`);
      } else {
        const created = await createAIAccount(settings, { ...form, source_urls: form.source_urls.map((url) => url.trim()).filter(Boolean) });
        setEditing(created.account.id); setPageCursor(undefined);
        setNotice(`Created @${created.user.username}. ${form.enabled ? "Autonomous posting is enabled." : "Posting is paused."}`);
      }
    });
  }

  return <div className="space-y-5">
    {status && <div className={panel}>
      <p className="font-semibold">{status.development_mode ? "Accelerated development checks" : "Content check schedule"}</p>
      <p className="text-sm text-[rgb(var(--muted))]">
        {status.development_mode ? `Enabled accounts check at randomized intervals of ${status.development_min_interval_seconds}–${status.development_max_interval_seconds} seconds. Saved check intervals apply when development mode is off. Research checks may publish nothing.` : "Each account checks on its own interval with jitter. Only worthwhile new research produces a post."}
        {" Status refreshes every 10 seconds."}
      </p>
    </div>}
    {error && <p role="alert" className="rounded-xl border border-red-500 p-4 text-sm">{error}</p>}
    {notice && <p role="status" className="rounded-xl border p-4 text-sm">{notice}</p>}
    <div className="grid items-start gap-5 xl:grid-cols-2">
      <div ref={formPanel} className={panel}>
        <div className="flex items-center justify-between gap-2"><h2 className="font-semibold">{editing ? `Edit @${form.username}` : "Create AI account"}</h2>
          {editing && <button className={button} disabled={busy} onClick={reset}>New account</button>}
        </div>
        <form onSubmit={save} className="space-y-4">
          <fieldset disabled={busy} className="space-y-4">
            {fields.map(({ key, label, multiline, max, hint }) => <label key={key} className="block space-y-1 text-sm">
              <span>{label}</span>
              {multiline ? <textarea className={control} required={key === "description"} rows={key === "description" ? 4 : 2} value={form[key]} maxLength={max}
                onChange={(event) => setForm({ ...form, [key]: event.target.value })} /> :
                <input className={control} value={form[key]} maxLength={max} type={key === "email" ? "email" : "text"}
                  required={["email", "username", "first_name", "topic"].includes(key)} disabled={!!editing && ["email", "username"].includes(key)}
                  onChange={(event) => setForm({ ...form, [key]: event.target.value })} />}
              {hint && <span className="block text-xs text-[rgb(var(--muted))]">{hint}</span>}
            </label>)}
            <label className="block space-y-1 text-sm"><span>Content mode</span>
              <select className={control} value={form.content_mode} onChange={(event) => setForm({...form, content_mode: event.target.value as AIAccountForm["content_mode"]})}>
                <option value="generative">Generative / evergreen</option><option value="research">Current / research</option>
              </select>
            </label>
            <label className="block space-y-1 text-sm"><span>{form.content_mode === "research" ? "Check interval (minutes)" : "Generation interval (minutes)"}</span>
              <input className={control} type="number" min={5} max={43200} required value={form.check_interval_seconds / 60} onChange={(event) => setForm({...form, check_interval_seconds: Number(event.target.value) * 60})} />
            </label>
            {form.content_mode === "research" && <>
              <label className="block space-y-1 text-sm"><span>RSS / Atom source URLs</span><textarea className={control} rows={3} required value={form.source_urls.join("\n")} onChange={(event) => setForm({...form, source_urls: event.target.value.split("\n")})} />
                <span className="block text-xs text-[rgb(var(--muted))]">One public HTTPS feed per line, up to five. Research is retrieved once and shared across variants.</span>
              </label>
              <label className="block space-y-1 text-sm"><span>Maximum source age (hours)</span><input className={control} type="number" min={1} max={2160} required value={form.source_max_age_hours} onChange={(event) => setForm({...form, source_max_age_hours: Number(event.target.value)})} /></label>
            </>}
            <fieldset className="space-y-2"><legend className="mb-2 text-sm font-medium">Model variants</legend>
              {status?.models.map((option) => <label className="flex items-start gap-2 text-sm" key={option.id}><input type="checkbox" checked={form.model_options.includes(option.id)} onChange={(event) => {
                const options = event.target.checked ? [...form.model_options, option.id] : form.model_options.filter((id) => id !== option.id);
                setForm({...form, model_options: options, default_model_option: options.includes(form.default_model_option) ? form.default_model_option : options[0] ?? ""});
              }} /><span>{option.name} · {option.model}<span className="block text-xs text-[rgb(var(--muted))]">{option.available ? option.provider : "Credentials / model not configured; variants will be unavailable"}</span></span></label>)}
            </fieldset>
            <label className="block space-y-1 text-sm"><span>Account fallback model</span><select className={control} required value={form.default_model_option} onChange={(event) => setForm({...form, default_model_option: event.target.value})}>
              {form.model_options.map((id) => <option value={id} key={id}>{status?.models.find((o) => o.id === id)?.name ?? id}</option>)}
            </select></label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />Enable autonomous posting</label>
            <button className={`${button} bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))]`} type="submit">{busy ? "Working…" : editing ? "Save changes" : "Create AI account"}</button>
          </fieldset>
        </form>
        {editing && <div className="border-t pt-4 space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="font-semibold">Recent content items</h3>
            <button className={button} disabled={busy || !form.enabled} onClick={() => void action(async () => { await checkAIAccount(settings, editing); setNotice("Check scheduled. The poster will research or generate using the saved configuration."); })}>Check now</button>
          </div>
          <p className="text-xs text-[rgb(var(--muted))]">One item can have several model versions. Followers receive one version according to their preference.</p>
          {items.length === 0 && <p className="text-sm">No content items yet. Research may complete a check without publishing.</p>}
          {items.map((item) => <article key={item.id} className="space-y-3 border-t pt-3">
            <h4 className="text-sm font-medium">{item.title} · {item.status}</h4><p className="text-xs text-[rgb(var(--muted))]">Item {item.id} · {when(item.created_at)}</p>
            {item.sources.map((source) => <p className="text-xs" key={source.url}><a className="underline" href={source.url} target="_blank" rel="noreferrer">{source.name}: {source.title}</a> {when(source.published_at)}</p>)}
            {item.variants.map((variant) => <div key={variant.option_id} className="rounded-lg border p-3 space-y-2"><p className="text-xs font-medium">{variant.option_id} · {variant.provider} / {variant.model} · {variant.status}</p>
              {variant.body && <p className="whitespace-pre-wrap text-sm">{variant.body}</p>}{variant.post_id && <p className="text-xs text-[rgb(var(--muted))]">Post {variant.post_id}</p>}{variant.error && <p className="text-xs text-red-500">{variant.error}</p>}
            </div>)}
          </article>)}
        </div>}
      </div>
      <section className={panel} aria-label="AI account roster">
        <div className="flex items-center justify-between"><h2 className="font-semibold">AI accounts</h2><button className={button} disabled={loading || busy} onClick={() => setTick((value) => value + 1)}>Refresh status</button></div>
        {roster.length === 0 && <p className="text-sm text-[rgb(var(--muted))]">{loading ? "Loading accounts…" : "Create a content account to start."}</p>}
        {roster.map((user) => <article key={user.id} className="space-y-3 border-t pt-4">
          <div className="flex justify-between gap-2"><div><h3 className="font-medium">{user.first_name} {user.last_name}</h3><p className="text-sm text-[rgb(var(--muted))]">@{user.username} · AI</p></div>
            <span className="text-xs">{user.blocked_at ? "Blocked" : user.ai_enabled ? user.ai_generation_status === "running" ? "Generating…" : "Enabled" : "Paused"}</span></div>
          <p className="text-sm">{user.about}</p>
          <dl className="grid grid-cols-[auto,1fr] gap-x-4 gap-y-1 text-xs text-[rgb(var(--muted))]">
            <dt>Mode / interval</dt><dd>{user.ai_content_mode} · {user.ai_check_interval_seconds / 60} min</dd><dt>Models</dt><dd>{user.ai_model_options?.join(", ")}</dd><dt>Last check</dt><dd>{when(user.ai_last_checked_at)} · {user.ai_last_check_outcome || "Not checked"}</dd><dt>Posts</dt><dd>{user.post_count}</dd>
            <dt>Last post</dt><dd>{when(user.ai_last_post_at)}</dd><dt>Next check</dt><dd>{user.ai_enabled ? when(user.ai_next_post_at) : "Paused"}</dd>
          </dl>
          {user.ai_generation_error && <p className="text-xs text-red-500">Last attempt: {user.ai_generation_error}</p>}
          <div className="flex gap-2"><button className={button} disabled={busy || !!user.blocked_at} onClick={() => void edit(user)}>Edit @{user.username}</button>
            <button className={button} disabled={busy || !!user.blocked_at || !user.ai_account_id} onClick={() => void action(async () => {
              await updateAIAccount(settings, user.ai_account_id!, { enabled: !user.ai_enabled });
              if (editing === user.ai_account_id) setForm((current) => ({ ...current, enabled: !user.ai_enabled }));
              setNotice(`@${user.username} ${user.ai_enabled ? "paused" : "enabled"}.`);
            })}>{user.ai_enabled ? "Pause" : "Enable"}</button></div>
        </article>)}
        <div className="flex gap-2">{pageCursor && <button className={button} onClick={() => setPageCursor(undefined)}>First page</button>}
          {cursor && <button className={button} onClick={() => setPageCursor(cursor)}>Next 25 accounts</button>}</div>
      </section>
    </div>
  </div>;
}
