import copy
import importlib.util
import json
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location("seed_ai_catalog", Path(__file__).with_name("seed-ai-catalog.py"))
seed = importlib.util.module_from_spec(spec)
spec.loader.exec_module(seed)


class CatalogTest(unittest.TestCase):
    def setUp(self):
        self.catalog = json.loads(seed.MANIFEST.read_text())

    def test_catalog_preserves_daily_schedule_and_requires_enabled_sources(self):
        entries = seed.validate_catalog(self.catalog)
        self.assertEqual(len(entries), 24)
        disabled = [a for a in entries if not a["enabled"]]
        self.assertEqual([a["handle"] for a in disabled], ["today_in_history"])
        self.assertEqual(disabled[0]["source_urls"], [])
        broken = copy.deepcopy(self.catalog)
        broken["accounts"][0]["source_urls"] = []
        with self.assertRaises(ValueError):
            seed.validate_catalog(broken)

    def test_alias_reuses_existing_identity_and_refuses_collision(self):
        entry = next(a for a in self.catalog["accounts"] if a["handle"] == "dad_jokes")
        existing = {"id": "123", "username": "dad_joke_day", "account_type": "ai", "ai_account_id": "456"}
        self.assertIs(seed.find_existing(entry, [existing]), existing)
        with self.assertRaises(ValueError):
            seed.find_existing(entry, [existing, {**existing, "username": "dad_jokes"}])
        with self.assertRaises(ValueError):
            seed.find_existing(entry, [{**existing, "account_type": "human"}])

    def test_model_selection_is_provider_neutral_and_excludes_unavailable_and_mock(self):
        status = {"models": [
            {"id": "claude", "provider": "anthropic", "available": True},
            {"id": "grok", "provider": "xai", "available": True},
            {"id": "openai", "provider": "openai", "available": False},
            {"id": "fixtures", "provider": "mock", "available": True},
        ]}
        self.assertEqual(seed.available_models(status), ["claude", "grok"])

    def test_unchanged_seed_does_not_reset_schedule(self):
        entry = self.catalog["accounts"][0]
        payload = seed.desired_payload(entry, ["claude", "grok"])
        record = {"account": {**payload, "next_generate_at": "2099-01-01T00:00:00Z"},
                  "user": {k: payload[k] for k in ("first_name", "last_name", "about")}}
        self.assertFalse(seed.needs_update(record, payload))
        record["account"]["description"] = "Old fictional-person mission"
        self.assertTrue(seed.needs_update(record, payload))


if __name__ == "__main__":
    unittest.main()
