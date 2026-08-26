"""Build shim for version stamping.

pyproject.toml carries a static version so the package builds standalone.
When the repo-root VERSION file exists (normal in-tree and sdist-from-repo
builds), setup.py overrides that static value so the wheel or sdist always
matches the release stamp in VERSION. setuptools gives setup() keyword
arguments precedence over [project] static metadata, which makes this the
single-source-of-truth bridge required by plans/0.2.5 Phase 46.3.
"""

from pathlib import Path

from setuptools import setup

_ROOT_VERSION = Path(__file__).resolve().parent.parent.parent / "VERSION"

if _ROOT_VERSION.is_file():
    setup(version=_ROOT_VERSION.read_text(encoding="utf-8").strip())
else:
    setup()
