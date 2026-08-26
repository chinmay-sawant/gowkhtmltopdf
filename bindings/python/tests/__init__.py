"""Test package bootstrap.

Puts the src layout on sys.path so `unittest discover` works from the
repo root without installing the package first.
"""

import os
import sys

_SRC = os.path.abspath(
    os.path.join(os.path.dirname(__file__), os.pardir, "src")
)
if _SRC not in sys.path:
    sys.path.insert(0, _SRC)
