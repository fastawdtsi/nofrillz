#!/usr/bin/env python3
"""Verify existing local content items via APIs, without asking an LLM to generate.
Creates/reuses two local readers, follows the three demo topic accounts, and
exercises global preferences, per-account overrides, deterministic fallback,
provenance, logical feed uniqueness, profile history, likes and bookmarks.
"""
import json
import os
import time
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlencode, urlparse
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[1]
BASE = os.environ.get('NOFRILLZ_VERIFY_API_URL', 'http://localhost:3100').rstrip('/')
if urlparse(BASE).hostname not in ('localhost', '127.0.0.1'):
    raise SystemExit('This script is for the local development API only.')
config = {}
if (ROOT / '.env').exists():
    for line in (ROOT / '.env').read_text().splitlines():
        if '=' in line and not line.lstrip().startswith('#'):
            key, value = line.split('=', 1)
            config[key] = value.strip().strip('\"\'')
ADMIN = {'X-Admin-API-Key': os.environ.get('NOFRILLZ_ADMIN_API_KEY', config.get('NOFRILLZ_ADMIN_API_KEY', 'dev-admin-key'))}


def call(path, method='GET', body=None, headers=None, expected=200):
    data = None if body is None else json.dumps(body).encode()
    request = Request(BASE + path, data=data, method=method, headers={'Content-Type': 'application/json', **(headers or {})})
    for attempt in range(4):
        try:
            with urlopen(request, timeout=30) as response:
                code, raw = response.status, response.read()
        except HTTPError as error:
            code, raw = error.code, error.read()
        if code != 429 or attempt == 3:
            break
        time.sleep(5)
    if code != expected:
        raise AssertionError(f'{method} {path}: expected {expected}, got {code}: {raw[:200]!r}')
    return json.loads(raw) if raw and raw[:1] in (b'{', b'[') else None


def main():
    call('/health')
    call('/admin/ai/status', expected=401)
    call('/feed/discover', expected=401)
    call('/users/me/ai-preference', expected=401)
    catalog = call('/ai/models')['models']
    people = call('/admin/users?account_type=ai&limit=100', headers=ADMIN)['users']
    names = ['ai_tech', 'healthy_living', 'dad_joke_day']
    accounts = {name: next(p for p in people if p['username'] == name) for name in names}
    items = {}
    for name, account in accounts.items():
        assert isinstance(account['id'], str) and isinstance(account['ai_account_id'], str)
        result = call('/admin/ai/accounts/' + account['ai_account_id'] + '/content', headers=ADMIN)['items']
        complete = [item for item in result if item['status'] == 'complete' and sum(v['status'] == 'published' for v in item['variants']) >= (1 if name == 'ai_tech' else 2)]
        assert complete, f'Wait for a complete item with two successful variants from @{name}'
        assert len({i['dedup_key'] for i in result}) == len(result), 'Duplicate logical content keys'
        item = complete[0]
        variants = [v for v in item['variants'] if v['status'] == 'published']
        if name != 'ai_tech':
            assert {'openai', 'openai_compact'} <= {v['option_id'] for v in variants}
        assert all(v['provider'] == 'openai' and v['model'].startswith('gpt-4.1-') for v in variants)
        assert len({v['post_id'] for v in variants}) == len(variants)
        if name == 'ai_tech':
            assert item['sources'] and all(s['published_at'] and s['discovered_at'] and s['url'].startswith('https://') for s in item['sources'])
        else:
            assert not item['sources'], 'Evergreen account unexpectedly claims external research'
        items[name] = item

    sessions = []
    def reader(suffix, option):
        email, password = f'content.reader.{suffix}@example.invalid', 'LocalContentReader-2026!'
        existing = call('/admin/users?' + urlencode({'q': email}), headers=ADMIN)['users']
        if not any(p['email'] == email for p in existing):
            call('/users', 'POST', {'email': email, 'username': 'content_reader_' + suffix, 'first_name': 'Content', 'last_name': 'Reader ' + suffix.upper(), 'about': 'Local content-model preference verification', 'password': password, 'ai_model_preference': option})
        token = call('/sessions', 'POST', {'email': email, 'password': password})['session_token']
        auth = {'Authorization': 'Bearer ' + token}
        sessions.append(auth)
        call('/users/me/ai-preference', 'PATCH', {'model_option': option}, auth)
        for a in accounts.values():
            call('/users/' + a['id'] + '/follow', 'POST', headers=auth)
            effective = call('/ai/accounts/' + a['id'] + '/models', headers=auth)
            if effective['override'] is not None:
                call('/ai/accounts/' + a['id'] + '/preference', 'DELETE', headers=auth)
        return auth

    def selected(auth, path='/feed/discover?limit=100'):
        posts = call(path, headers=auth)['posts']
        content = [p for p in posts if p.get('content_item_id')]
        assert len(content) == len({p['content_item_id'] for p in content}), 'More than one variant of an item in a feed'
        return {p['content_item_id']: p for p in content}

    try:
        first, second = reader('a', 'openai'), reader('b', 'openai_compact')
        feed_a, feed_b = selected(first), selected(second)
        examples = {}
        for name, item in items.items():
            a, b = feed_a[item['id']], feed_b[item['id']]
            successful = {v['option_id'] for v in item['variants'] if v['status']=='published'}
            assert a['model_option'] == 'openai'
            assert b['model_option'] == ('openai_compact' if 'openai_compact' in successful else 'openai')
            if 'openai_compact' in successful: assert a['id'] != b['id']
            assert a['user']['id'] == b['user']['id'] == accounts[name]['id']
            assert a['source'] == 'ai' and a['user']['account_type'] == 'ai'
            examples[name] = {'content_item_id': item['id'], 'openai': a, 'openai_compact': b, 'sources': item['sources']}
            for auth in (first, second):
                assert item['id'] in selected(auth, '/feed?limit=100')
                profile = call('/users/' + accounts[name]['id'] + '/posts?limit=20', headers=auth)
                entries = profile if isinstance(profile, list) else profile['posts']
                allowed = {v['post_id'] for v in item['variants'] if v.get('post_id')}
                assert sum(str(p['id']) in allowed for p in entries) == 1, 'Profile exposes duplicate variants'
        news = items['ai_tech']
        preference_item = items['healthy_living']
        route = '/ai/accounts/' + accounts['healthy_living']['id'] + '/preference'
        overridden = call(route, 'PATCH', {'model_option': 'openai_compact'}, first)
        assert overridden['effective_model_option'] == 'openai_compact'
        assert selected(first)[preference_item['id']]['id'] == feed_b[preference_item['id']]['id']
        reset = call(route, 'DELETE', headers=first)
        assert reset['override'] is None and reset['effective_model_option'] == 'openai'
        assert selected(first)[preference_item['id']]['id'] == feed_a[preference_item['id']]['id']
        call('/users/me/ai-preference', 'PATCH', {'model_option': 'grok'}, first)
        assert selected(first)[news['id']]['model_option'] == 'openai', 'Unavailable preference failed to fall back'
        call('/users/me/ai-preference', 'PATCH', {'model_option': 'openai'}, first)
        call('/ai/accounts/' + accounts['healthy_living']['id'] + '/preference', 'PATCH', {'model_option':'openai_compact'}, first)
        post_id = feed_b[news['id']]['id']
        provenance = call('/posts/' + post_id + '/ai-content', headers=second)
        assert provenance['content_item_id'] == news['id'] and provenance['sources'] == news['sources']
        call('/posts/' + post_id + '/like', 'POST', headers=second)
        call('/posts/' + post_id + '/bookmark', 'POST', headers=second)
        reacted = selected(second)[news['id']]
        assert reacted['liked'] and reacted['is_bookmarked']
        assert not selected(first)[news['id']]['liked'], 'Reaction leaked across users/variants'
        call('/posts/' + post_id + '/like', 'DELETE', headers=second)
        call('/posts/' + post_id + '/bookmark', 'DELETE', headers=second)
        checks = ['real OpenAI variants (with per-item partial failures allowed)', 'shared logical items', 'source provenance', 'global preferences', 'account override and reset', 'unavailable-provider fallback', 'one variant per Discover/Following/profile item', 'logical deduplication', 'AI disclosure', 'normal likes/bookmarks', 'lossless IDs']
        output = {'catalog':catalog, 'accounts':accounts, 'examples':examples, 'checks':checks}
        (ROOT / 'artifacts').mkdir(exist_ok=True)
        (ROOT / 'artifacts' / 'ai-content-verification.json').write_text(json.dumps(output,ensure_ascii=False,indent=2))
        for name, example in examples.items():
            print('@'+name, 'item',example['content_item_id'])
            for option in ('openai','openai_compact'):
                print(' ', option + ' preference ->', example[option]['model_option'], example[option]['id'], example[option]['body'])
        print('PASS:',', '.join(checks))
    finally:
        for auth in sessions:
            call('/sessions/current','DELETE',headers=auth)


if __name__ == '__main__':
    main()
