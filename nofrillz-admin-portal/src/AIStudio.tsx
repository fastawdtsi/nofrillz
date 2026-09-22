import { FormEvent, useEffect, useRef, useState } from "react";
import { createAIAccount, createAIPost, getAIAccount, getAIStatus, listUsers, previewAIPost, updateAIAccount } from "./api";
import type { AIAccountForm, AIStatus, AdminUser, ConnectionSettings } from "./types";

const emptyForm: AIAccountForm = {
  email: "", username: "", first_name: "", last_name: "", about: "", enabled: true,
  topic: "", description: "", system_prompt: "", style_prompt: "", min_posts_per_day: 1, max_posts_per_day: 3,
};
const panel = "rounded-[18px] border bg-[rgb(var(--surface))] p-5 space-y-4";
const control = "w-full rounded-[12px] border bg-[rgb(var(--surface-muted)/0.82)] px-4 py-3 text-sm disabled:opacity-50";
const button = "rounded-full border px-4 py-2 text-sm font-medium disabled:opacity-50 hover:bg-[rgb(var(--surface-muted))]";
const when = (value?: string | null) => value ? new Date(value).toLocaleString() : "—";
const message = (error: unknown) => error instanceof Error ? error.message : "Request failed";
type TextField = Exclude<keyof AIAccountForm, "enabled" | "min_posts_per_day" | "max_posts_per_day">;
const fields: { key: TextField; label: string; multiline?: boolean; max: number; hint?: string }[] = [
  { key: "email", label: "Email", max: 254 }, { key: "username", label: "Username", max: 50 },
  { key: "first_name", label: "First name", max: 100 }, { key: "last_name", label: "Last name", max: 100 },
  { key: "about", label: "Public bio", multiline: true, max: 1024 },
  { key: "topic", label: "Interests and topics", max: 128, hint: "Separate interests with commas." },
  { key: "description", label: "Persona", multiline: true, max: 4000, hint: "Describe their background, personality, and everyday interests." },
  { key: "style_prompt", label: "Writing style", multiline: true, max: 4000, hint: "Voice, tone, vocabulary, and habits that distinguish this account." },
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
  const [draft, setDraft] = useState("");
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

  async function action(work: () => Promise<void>) {
    setBusy(true); setError(""); setNotice("");
    try { await work(); setTick((value) => value + 1); onChanged(); }
    catch (err) { setError(message(err)); }
    finally { setBusy(false); }
  }
  function reset() { setEditing(null); setForm({ ...emptyForm }); setDraft(""); }
  async function edit(user: AdminUser) {
    if (!user.ai_account_id) return;
    await action(async () => {
      const { account, user: profile } = await getAIAccount(settings, user.ai_account_id!);
      setEditing(account.id);
      setForm({ email: profile.email, username: profile.username, first_name: profile.first_name, last_name: profile.last_name,
        about: profile.about, enabled: account.enabled, topic: account.topic, description: account.description,
        system_prompt: account.system_prompt, style_prompt: account.style_prompt,
        min_posts_per_day: account.min_posts_per_day, max_posts_per_day: account.max_posts_per_day });
      setDraft(""); formPanel.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    await action(async () => {
      if (form.min_posts_per_day > form.max_posts_per_day) throw new Error("Maximum posts must be at least the minimum.");
      if (editing) {
        const { email: _email, username: _username, ...changes } = form;
        await updateAIAccount(settings, editing, changes);
        setNotice(`Saved @${form.username}. ${form.enabled ? "Next post scheduled." : "Autonomous posting paused."}`);
      } else {
        const created = await createAIAccount(settings, form);
        setEditing(created.account.id); setPageCursor(undefined);
        setNotice(`Created @${created.user.username}. ${form.enabled ? "Autonomous posting is enabled." : "Posting is paused."}`);
      }
    });
  }

  return <div className="space-y-5">
    {status && <div className={panel}>
      <p className="font-semibold">{status.development_mode ? "Accelerated development posting" : "Daily posting schedule"}</p>
      <p className="text-sm text-[rgb(var(--muted))]">
        {status.development_mode ? `Enabled accounts post at randomized intervals of ${status.development_min_interval_seconds}–${status.development_max_interval_seconds} seconds. Saved daily rates apply when development mode is off.` : "Each enabled account follows its own randomized daily rate."}
        {` Provider: ${status.provider}${status.provider === "openai" ? ` · ${status.model}` : " (test content)"}. Status refreshes every 10 seconds.`}
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
              {multiline ? <textarea className={control} rows={key === "description" ? 4 : 2} value={form[key]} maxLength={max}
                onChange={(event) => setForm({ ...form, [key]: event.target.value })} /> :
                <input className={control} value={form[key]} maxLength={max} type={key === "email" ? "email" : "text"}
                  required={["email", "username", "topic"].includes(key)} disabled={!!editing && ["email", "username"].includes(key)}
                  onChange={(event) => setForm({ ...form, [key]: event.target.value })} />}
              {hint && <span className="block text-xs text-[rgb(var(--muted))]">{hint}</span>}
            </label>)}
            <div className="grid grid-cols-2 gap-3">
              <label className="space-y-1 text-sm"><span>Minimum posts / day</span><input className={control} type="number" min={1} max={48} required value={form.min_posts_per_day}
                onChange={(event) => setForm({ ...form, min_posts_per_day: Number(event.target.value) })} /></label>
              <label className="space-y-1 text-sm"><span>Maximum posts / day</span><input className={control} type="number" min={form.min_posts_per_day || 1} max={48} required value={form.max_posts_per_day}
                onChange={(event) => setForm({ ...form, max_posts_per_day: Number(event.target.value) })} /></label>
            </div>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />Enable autonomous posting</label>
            <button className={`${button} bg-[rgb(var(--control-fill))] text-[rgb(var(--control-fill-foreground))]`} type="submit">{busy ? "Working…" : editing ? "Save changes" : "Create AI account"}</button>
          </fieldset>
        </form>
        {editing && <div className="border-t pt-4 space-y-3">
          <p className="text-sm">Test the saved persona with a preview. Previewing uses the configured provider and does not publish.</p>
          <button className={button} disabled={busy} onClick={() => void action(async () => {
            const result = await previewAIPost(settings, editing); setDraft(result.generation.candidate_body); setNotice("Preview ready; no post published.");
          })}>Generate preview</button>
          {draft && <><label className="block text-sm">Post draft<textarea className={`${control} mt-2`} rows={5} maxLength={1200} value={draft} onChange={(event) => setDraft(event.target.value)} /></label>
            <button className={button} disabled={busy || !draft.trim()} onClick={() => void action(async () => {
              await createAIPost(settings, editing, draft); setDraft(""); setNotice("Post published.");
            })}>Publish this draft</button></>}
        </div>}
      </div>
      <section className={panel} aria-label="AI account roster">
        <div className="flex items-center justify-between"><h2 className="font-semibold">AI accounts</h2><button className={button} disabled={loading || busy} onClick={() => setTick((value) => value + 1)}>Refresh status</button></div>
        {roster.length === 0 && <p className="text-sm text-[rgb(var(--muted))]">{loading ? "Loading accounts…" : "Create your first persona to start posting."}</p>}
        {roster.map((user) => <article key={user.id} className="space-y-3 border-t pt-4">
          <div className="flex justify-between gap-2"><div><h3 className="font-medium">{user.first_name} {user.last_name}</h3><p className="text-sm text-[rgb(var(--muted))]">@{user.username} · AI</p></div>
            <span className="text-xs">{user.blocked_at ? "Blocked" : user.ai_enabled ? user.ai_generation_status === "running" ? "Generating…" : "Enabled" : "Paused"}</span></div>
          <p className="text-sm">{user.about}</p>
          <dl className="grid grid-cols-[auto,1fr] gap-x-4 gap-y-1 text-xs text-[rgb(var(--muted))]">
            <dt>Daily rate</dt><dd>{user.ai_min_posts_per_day}–{user.ai_max_posts_per_day}</dd><dt>Posts</dt><dd>{user.post_count}</dd>
            <dt>Last post</dt><dd>{when(user.ai_last_post_at)}</dd><dt>Next post</dt><dd>{user.ai_enabled ? when(user.ai_next_post_at) : "Paused"}</dd>
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
