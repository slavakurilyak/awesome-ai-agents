import importlib.util
import unittest
from pathlib import Path
from unittest.mock import patch


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "03-generate-readme.py"
SPEC = importlib.util.spec_from_file_location("generate_readme", SCRIPT_PATH)
assert SPEC is not None
assert SPEC.loader is not None
generate_readme = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(generate_readme)


class GenerateReadmeMainTests(unittest.TestCase):
    def test_generate_sections_sorts_categories(self) -> None:
        project = generate_readme.Project(
            project="Example",
            project_description="Example project",
            project_is_open_source=True,
            categories=["Tool Calling (Function Calling)", "AI Agents"],
            sources=[
                generate_readme.Source(
                    source="website",
                    source_url="https://example.com",
                )
            ],
        )
        data = generate_readme.JsonData(agents=[project], categories=[])

        result = generate_readme.generate_sections(
            data,
            {
                "AI Agents": "🤖",
                "Tool Calling (Function Calling)": "🔧",
            },
        )

        self.assertIn(
            "<p>🤖 AI Agents | 🔧 Tool Calling (Function Calling)</p>",
            result,
        )

    def test_main_propagates_generation_failures(self) -> None:
        error = RuntimeError("invalid category data")

        with self.assertLogs(level="ERROR"):
            with patch.object(
                generate_readme,
                "load_category_emojis",
                side_effect=error,
            ):
                with self.assertRaisesRegex(RuntimeError, "invalid category data"):
                    generate_readme.main()


if __name__ == "__main__":
    unittest.main()
