#!/usr/bin/env python3
"""Idempotently seed the local AI catalog through authenticated admin APIs.

Default: validate live feeds and show the plan. --apply performs the plan.
Uses normal account/user creation and update paths; never writes SQL.
"""
import argparse
import datetime as dt
import json
import os
import re
import subprocess
import time
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlencode, urlparse
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "catalog" / "ai-accounts.json"
ACCOUNT_FIELDS = (
    "content_mode", "topic", "description", "system_prompt", "style_prompt",
    "exclusions", "check_interval_seconds", "source_max_age_hours", "source_urls",
)


def load_settings():
    settings = {}
    for line in (ROOT / ".env").read_text().splitlines() if (ROOT / ".env").exists() else []:
        if "=" in line and not line.lstrip().startswith("#"):
            key, value = line.split("=", 1)
            settings[key.strip()] = value.strip().strip("\"'")
    return settings


class Client:
    def __init__(self):
        self.base = os.environ.get("NOFRILLZ_CATALOG_API_URL", "http://localhost:3100").rstrip("/")
        parsed = urlparse(self.base)
        if parsed.hostname not in ("localhost", "127.0.0.1", "::1") or parsed.scheme not in ("http", "https") or parsed.username or parsed.query or parsed.fragment:
            raise ValueError("Catalog seeding is restricted to the local development API")
        self.key = os.environ.get("NOFRILLZ_ADMIN_API_KEY", load_settings().get("NOFRILLZ_ADMIN_API_KEY", "dev-admin-key"))

    def call(self, path, method="GET", body=None):
        request = Request(self.base + path, method=method,
                          data=json.dumps(body).encode() if body is not None else None,
                          headers={"Content-Type": "application/json", "X-Admin-API-Key": self.key})
        for attempt in range(4):
            try:
                with urlopen(request, timeout=30) as response:
                    return json.load(response)
            except HTTPError as error:
                if error.code == 429 and attempt < 3:
                    time.sleep(5)
                    continue
                # Do not echo arbitrary response bodies or credentials.
                raise RuntimeError(f"{method} {path}: HTTP {error.code}") from None

    def users(self):
        cursor = ""
        users = []
        while True:
            page = self.call("/admin/users?" + urlencode({"limit": 100, "cursor": cursor}))
            users.extend(page["users"])
            cursor = page.get("next_cursor", "")
            if not cursor:
                return users


def validate_catalog(catalog):
    accounts = catalog["accounts"]
    if catalog["version"] != 1 or len(accounts) != 24:
        raise ValueError("Expected the version 1, 24-account catalog")
    handles = set()
    for entry in accounts:
        for handle in [entry["handle"], *entry["existing_handles"]]:
            if handle in handles or not re.fullmatch(r"[a-z0-9_.-]{3,32}", handle):
                raise ValueError(f"Duplicate/invalid catalog handle: {handle}")
            handles.add(handle)
        if entry["check_interval_seconds"] != 86400:
            raise ValueError("This catalog must preserve the requested daily schedule")
        if entry["content_mode"] not in ("research", "generative") or not entry["description"] or not entry["topic"]:
            raise ValueError(f"Invalid content mission: {entry['handle']}")
        if entry["enabled"] and entry["content_mode"] == "research" and not entry["source_urls"]:
            raise ValueError(f"Enabled research needs sources: {entry['handle']}")
        if not entry["enabled"] and not entry["disabled_reason"]:
            raise ValueError(f"Disabled account needs a reason: {entry['handle']}")
        if entry["content_mode"] == "generative" and entry["source_urls"]:
            raise ValueError("Evergreen accounts must not imply current external research")
        if len(entry["source_urls"]) > 5:
            raise ValueError("Too many sources")
    return accounts


def find_existing(entry, users):
    names = {entry["handle"], *entry["existing_handles"]}
    matches = [u for u in users if u["username"].lower() in names]
    if len(matches) > 1:
        raise ValueError(f"Multiple existing identities match {entry['handle']}; refusing to duplicate or merge")
    if not matches:
        return None
    user = matches[0]
    if user["account_type"] != "ai" or not user.get("ai_account_id") or user.get("blocked_at"):
        raise ValueError(f"Handle belongs to another or blocked account: {user['username']}")
    return user


def available_models(status):
    # Provider-neutral: newly configured real options automatically participate.
    options = sorted(m["id"] for m in status["models"] if m["available"] and m["provider"] != "mock")
    if not options or len(options) > 8:
        raise ValueError("Configure between one and eight available real model options")
    return options


def desired_payload(entry, options, existing=None):
    account = existing["account"] if existing else {}
    default = account.get("default_model_option")
    if default not in options:
        default = options[0]
    return {
        **{key: entry[key] for key in ACCOUNT_FIELDS},
        "first_name": entry["name"], "last_name": "", "about": entry["description"],
        "enabled": entry["enabled"], "min_posts_per_day": 1, "max_posts_per_day": 1,
        "model_options": options, "default_model_option": default,
    }


def needs_update(record, payload):
    for key, value in payload.items():
        current = record["user" if key in ("first_name", "last_name", "about") else "account"].get(key)
        if current != value:
            return True
    return False


def check_sources(accounts):
    urls = sorted({url for a in accounts if a["enabled"] for url in a["source_urls"]})
    if not urls:
        return []
    # This runs the SAME URL protections, timestamp rules, size limit and parser
    # as the deployed poster. No model calls occur here.
    result = subprocess.run(["go", "run", "./cmd/check-ai-sources", "-max-age-hours", "168", *urls],
                            cwd=ROOT.parent / "nofrillz-go", capture_output=True, text=True, check=True)
    checks = json.loads(result.stdout)
    (ROOT / "artifacts").mkdir(exist_ok=True)
    (ROOT / "artifacts" / "catalog-source-preflight.json").write_text(json.dumps(checks, indent=2) + "\n")
    failed = [c["url"] for c in checks if c.get("error") or c["candidates"] == 0]
    if failed:
        raise ValueError("Source preflight has no usable current candidates; review before enabling: " + ", ".join(failed))
    return checks


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true", help="Create/update accounts after preflight")
    args = parser.parse_args()
    entries = validate_catalog(json.loads(MANIFEST.read_text()))
    client = Client()
    status = client.call("/admin/ai/status")
    if status["development_mode"]:
        raise ValueError("Disable accelerated development timing before seeding this daily catalog")
    options = available_models(status)
    users = client.users()
    plan = []
    # Resolve every collision before the first mutation.
    for entry in entries:
        user = find_existing(entry, users)
        record = client.call("/admin/ai/accounts/" + user["ai_account_id"]) if user else None
        payload = desired_payload(entry, options, record)
        action = "create" if record is None else ("update" if needs_update(record, payload) else "unchanged")
        plan.append((entry, user, record, payload, action))
    checks = check_sources(entries)
    records = []
    for index, (entry, user, record, payload, action) in enumerate(plan):
        if args.apply:
            if action == "create":
                # Start new accounts one full daily interval from now, with a
                # small stagger. Seeding itself never triggers a generation.
                when = dt.datetime.now(dt.timezone.utc) + dt.timedelta(seconds=86400 + index * 10)
                create = {**payload, "username": entry["handle"], "email": entry["handle"] + ".catalog@example.invalid"}
                if entry["enabled"]:
                    create["next_generate_at"] = when.isoformat().replace("+00:00", "Z")
                record = client.call("/admin/ai/accounts", "POST", create)
            elif action == "update":
                record = client.call("/admin/ai/accounts/" + user["ai_account_id"], "PATCH", payload)
            if needs_update(record, payload):
                raise RuntimeError(f"Saved configuration does not match {entry['handle']}")
            if user and (record["user"]["id"] != user["id"] or record["account"]["id"] != user["ai_account_id"]):
                raise RuntimeError("Existing account identity changed")
        name = record["user"]["username"] if record else entry["handle"]
        row = {"name": entry["name"], "handle": name, "action": action, "mode": entry["content_mode"],
               "enabled": entry["enabled"], "disabled_reason": entry["disabled_reason"], "model_options": options,
               "source_urls": entry["source_urls"], "check_interval_seconds": 86400}
        if record:
            row.update(user_id=record["user"]["id"], account_id=record["account"]["id"], next_check=record["account"]["next_generate_at"])
        records.append(row)
        print(f"{action:9} @{name:20} {entry['content_mode']:10} {'enabled' if entry['enabled'] else 'disabled'}")
    report = {"applied": args.apply, "checked_at": dt.datetime.now(dt.timezone.utc).isoformat(), "sources": checks, "accounts": records}
    (ROOT / "artifacts").mkdir(exist_ok=True)
    (ROOT / "artifacts" / ("catalog-seed.json" if args.apply else "catalog-plan.json")).write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"{'Applied' if args.apply else 'Planned'} {len(records)} accounts; model options: {', '.join(options)}")


if __name__ == "__main__":
    main()
