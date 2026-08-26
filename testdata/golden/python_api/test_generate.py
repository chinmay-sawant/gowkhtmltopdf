#!/usr/bin/env python3
"""Path-resolution tests for the Python API architecture generator.

These tests do not require libgowkhtmltopdf; they only exercise the
resolver helpers in generate.py.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import generate  # noqa: E402


class ResolveTemplatePathsTest(unittest.TestCase):
    def test_resolve_from_python_api_directory(self):
        input_path, output_path, repo_root = generate.resolve_template_paths(
            working_dir=_HERE,
            source_file=_HERE / "generate.py",
        )
        self.assertEqual(input_path, _HERE / generate.INPUT_NAME)
        self.assertEqual(
            output_path,
            repo_root / generate.SAMPLE_DIRECTORY / generate.OUTPUT_NAME,
        )
        self.assertEqual(repo_root, _HERE.parent.parent.parent)
        self.assertTrue(generate.is_python_api_template(input_path))

    def test_resolve_from_repo_root(self):
        repo_root = _HERE.parent.parent.parent
        input_path, output_path, resolved_root = generate.resolve_template_paths(
            working_dir=repo_root,
            source_file=_HERE / "generate.py",
        )
        self.assertEqual(
            input_path,
            repo_root / generate.API_DIRECTORY / generate.INPUT_NAME,
        )
        self.assertEqual(
            output_path,
            repo_root / generate.SAMPLE_DIRECTORY / generate.OUTPUT_NAME,
        )
        self.assertEqual(resolved_root, repo_root)

    def test_unique_paths_drops_duplicates_and_empty(self):
        with tempfile.TemporaryDirectory() as tmp:
            first = str(Path(tmp) / "a.pdf")
            second = str(Path(tmp) / "b.pdf")
            got = generate.unique_paths(first, "", first, second)
            self.assertEqual(len(got), 2)
            self.assertEqual(Path(got[0]).name, "a.pdf")

    def test_same_path(self):
        with tempfile.TemporaryDirectory() as tmp:
            a = str(Path(tmp) / "file.html")
            b = str(Path(tmp) / "." / "file.html")
            self.assertTrue(generate.same_path(a, b))
            self.assertFalse(
                generate.same_path(a, str(Path(tmp) / "other.html"))
            )

    def test_is_python_api_template_rejects_corpus_and_go_api(self):
        python_html = Path("testdata") / "golden" / "python_api" / generate.INPUT_NAME
        corpus_html = Path("testdata") / "golden" / generate.INPUT_NAME
        go_api_html = Path("testdata") / "golden" / "api" / generate.INPUT_NAME

        self.assertTrue(generate.is_python_api_template(python_html))
        self.assertFalse(generate.is_python_api_template(corpus_html))
        self.assertFalse(generate.is_python_api_template(go_api_html))


if __name__ == "__main__":
    unittest.main()
