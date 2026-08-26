"""Build shim: VERSION stamp + platform wheel tags.

pyproject.toml carries a static version so the package builds standalone.
When the repo-root VERSION file exists (normal in-tree and sdist-from-repo
builds), setup.py overrides that static value so the wheel or sdist always
matches the release stamp in VERSION.

The package ships a prebuilt c-shared library as package data and loads it
with ctypes. That is not a Python C extension, so setuptools would emit
py3-none-any. cibuildwheel rejects pure-Python wheels, so this shim forces
a platform tag (py3-none-<plat>) while keeping abi=none.
"""

from pathlib import Path

from setuptools import setup

try:
    from wheel.bdist_wheel import bdist_wheel as _bdist_wheel
except ImportError:  # pragma: no cover - wheel is listed in build-system.requires
    _bdist_wheel = None


class bdist_wheel(_bdist_wheel):  # type: ignore[misc,valid-type]
    def finalize_options(self):
        super().finalize_options()
        self.root_is_pure = False

    def get_tag(self):
        _python, _abi, plat = super().get_tag()
        # ctypes-loaded native lib: one wheel per OS/arch, any CPython 3.x.
        return "py3", "none", plat


_ROOT_VERSION = Path(__file__).resolve().parent.parent.parent / "VERSION"

_kwargs = {}
if _bdist_wheel is not None:
    _kwargs["cmdclass"] = {"bdist_wheel": bdist_wheel}

if _ROOT_VERSION.is_file():
    _kwargs["version"] = _ROOT_VERSION.read_text(encoding="utf-8").strip()

setup(**_kwargs)
