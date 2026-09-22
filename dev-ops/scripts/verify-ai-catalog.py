#!/usr/bin/env python3
"""Verify seeded catalog, real content provenance, and normal feed selection.
Does not trigger generation. Creates/reuses two LOCAL test readers and follows
the enabled catalog. Run after the requested first checks finish.
"""
import importlib.util
import datetime as dt
import json
from pathlib import Path
from urllib.parse import urlencode

ROOT = Path(__file__).resolve().parents[1]


def module(name, file):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(file))
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


seed = module("seed_catalog", "seed-ai-catalog.py")
api = module("verify_content", "verify-ai-content.py")
call = api.call


def timestamp(value):
    # Go omits trailing fractional zeros; Python 3.9 fromisoformat accepts
    # only 3 or 6 fractional digits, whereas strptime handles either form.
    return dt.datetime.strptime(value, "%Y-%m-%dT%H:%M:%S" + (".%f" if "." in value else "") + "Z")


def feed(auth, route, wanted):
    found, cursor, seen = {}, "", set()
    for _ in range(10):
        page = call(route + "?" + urlencode({"limit": 100, "cursor": cursor}), headers=auth)
        for post in page["posts"]:
            item_id = post.get("content_item_id")
            if item_id:
                assert item_id not in seen, "A feed returned two variants of one logical item"
                seen.add(item_id)
                if item_id in wanted:
                    found[item_id] = post
        cursor = page.get("next_cursor", "")
        if wanted <= found.keys() or not cursor:
            break
    assert wanted <= found.keys(), "A published catalog item is missing from the normal feed"
    return found


def main():
    client = seed.Client()
    api.BASE = client.base
    status = client.call("/admin/ai/status")
    assert status["development_mode"] is False
    options = seed.available_models(status)
    assert len(options) >= 2, "This demonstration needs two actually configured model options"
    entries = seed.validate_catalog(json.loads(seed.MANIFEST.read_text()))
    users = client.users()
    before = json.loads((ROOT / "artifacts" / "catalog-before.json").read_text())["accounts"]
    original = {r["user"]["username"]: r for r in before}
    accounts, published, checks = {}, {}, []
    for entry in entries:
        user = seed.find_existing(entry, users)
        assert user, entry["handle"]
        record = client.call("/admin/ai/accounts/" + user["ai_account_id"])
        assert not seed.needs_update(record, seed.desired_payload(entry, options, record))
        account = record["account"]
        if user["username"] in original:
            old = original[user["username"]]
            assert old["account"]["id"] == account["id"] and old["user"]["id"] == user["id"], "Existing identity was duplicated"
        result = client.call("/admin/ai/accounts/" + account["id"] + "/content")["items"]
        if entry["enabled"]:
            assert account["generation_status"] == "idle" and account["last_checked_at"], "Wait for initial checks to finish"
            next_check = timestamp(account["next_generate_at"])
            last_check = timestamp(account["last_checked_at"])
            assert next_check - last_check >= dt.timedelta(days=1), "Daily schedule was accelerated"
        else:
            assert not result and account["next_generate_at"] is None
        for item in result:
            variants = [v for v in item["variants"] if v["status"] == "published"]
            if not variants:
                continue
            assert len({v["option_id"] for v in variants}) == len(variants)
            assert len({v["post_id"] for v in variants}) == len(variants)
            assert all(v["provider"] != "mock" and v["option_id"] in options for v in variants)
            if entry["content_mode"] == "research":
                assert item["sources"] and item["context"]
                assert all(s["url"].startswith("https://") and s.get("published_at") and s["discovered_at"] for s in item["sources"])
            else:
                assert item["context"] and not item["sources"]
            published[item["id"]] = {"item": item, "variants": variants, "account": account, "user": user}
            break  # one representative published item from each account
        accounts[user["username"]] = record
        checks.append({"handle": user["username"], "mode": entry["content_mode"], "enabled": entry["enabled"],
                       "account_id": account["id"], "user_id": user["id"], "outcome": account["last_check_outcome"],
                       "last_checked_at": account["last_checked_at"], "next_check_at": account["next_generate_at"],
                       "latest_item": result[0] if result else None})
    assert len(accounts) == 24
    ai_users = [u for u in users if u["account_type"] == "ai"]
    assert len(ai_users) == 24, "Unexpected duplicate or obsolete visible AI accounts"
    assert {v["account"]["content_mode"] for v in published.values()} == {"generative", "research"}
    paired = [v for v in published.values() if {options[0], options[1]} <= {x["option_id"] for x in v["variants"]}]
    assert paired, "No actual multi-model logical item is available"
    sessions = []

    def reader(suffix, preference):
        email = "catalog.reader." + suffix + "@example.invalid"
        password = "LocalCatalogReader-2026!"
        matches = client.call("/admin/users?" + urlencode({"q": email}))["users"]
        if not any(u["email"] == email for u in matches):
            call("/users", "POST", {"email": email, "username": "catalog_reader_" + suffix, "first_name": "Catalog", "last_name": "Reader " + suffix.upper(), "password": password, "about": "Local AI catalog verification", "ai_model_preference": preference})
        token = call("/sessions", "POST", {"email": email, "password": password})["session_token"]
        auth = {"Authorization": "Bearer " + token}
        sessions.append(auth)
        call("/users/me/ai-preference", "PATCH", {"model_option": preference}, auth)
        for record in accounts.values():
            if not record["account"]["enabled"]:
                continue
            user_id = record["user"]["id"]
            effective = call("/ai/accounts/" + user_id + "/models", headers=auth)
            if not effective["following"]:
                call("/users/" + user_id + "/follow", "POST", headers=auth)
            if effective["override"] is not None:
                call("/ai/accounts/" + user_id + "/preference", "DELETE", headers=auth)
        return auth

    try:
        auth_a, auth_b = reader("a", options[0]), reader("b", options[1])
        wanted = set(published)
        feeds = {}
        for suffix, auth, preference in [("a", auth_a, options[0]), ("b", auth_b, options[1])]:
            feeds[suffix] = feed(auth, "/feed/discover", wanted)
            following = feed(auth, "/feed", wanted)
            for item_id, value in published.items():
                by_option = {v["option_id"]: v for v in value["variants"]}
                expected = next(by_option[o] for o in [preference, value["account"]["default_model_option"], *sorted(by_option)] if o in by_option)
                for visible in (feeds[suffix][item_id], following[item_id]):
                    assert visible["id"] == expected["post_id"] and visible["model_option"] == expected["option_id"]
                    assert visible["source"] == "ai" and visible["user"]["account_type"] == "ai"
                profile = call("/users/" + value["user"]["id"] + "/posts?limit=100", headers=auth)
                posts = profile if isinstance(profile, list) else profile["posts"]
                ids = {v["post_id"] for v in value["variants"]}
                assert sum(str(p["id"]) in ids for p in posts) == 1
        example = next((p for p in paired if p["account"]["content_mode"] == "generative"), paired[0])
        item_id = example["item"]["id"]
        assert feeds["a"][item_id]["id"] != feeds["b"][item_id]["id"]
        override_path = "/ai/accounts/" + example["user"]["id"] + "/preference"
        call(override_path, "PATCH", {"model_option": options[1]}, auth_a)
        assert feed(auth_a, "/feed/discover", {item_id})[item_id]["id"] == feeds["b"][item_id]["id"]
        call(override_path, "DELETE", headers=auth_a)
        for item_id, value in published.items():
            if value["account"]["content_mode"] == "research":
                provenance = call("/posts/" + feeds["a"][item_id]["id"] + "/ai-content", headers=auth_a)
                assert provenance["sources"] == value["item"]["sources"]
        output = {"accounts": checks, "representative_items": published, "feed_a": feeds["a"], "feed_b": feeds["b"],
                  "preference_example_item": example["item"]["id"], "model_options": options,
                  "checks": ["24 unique catalog accounts", "existing identities preserved", "daily timing", "disabled history account has no posts",
                             "real provider variants", "research provenance", "shared generative seed", "one variant per Discover/Following/profile item",
                             "global model choice", "per-account override and reset", "fallback to available variants"]}
        (ROOT / "artifacts" / "catalog-verification.json").write_text(json.dumps(output, ensure_ascii=False, indent=2) + "\n")
        print("PASS:", ", ".join(output["checks"]))
        print("Published account examples:", len(published), "two-model examples:", len(paired))
        print("Preference example:", example["user"]["username"], example["item"]["id"], feeds["a"][example["item"]["id"]]["id"], feeds["b"][example["item"]["id"]]["id"])
    finally:
        for auth in sessions:
            call("/sessions/current", "DELETE", headers=auth)


if __name__ == "__main__":
    main()
