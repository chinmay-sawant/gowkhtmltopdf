"""ctypes loader for libgowkhtmltopdf and the frozen C ABI structs.

Search order for the shared library:

1. ``GOWKHTMLTOPDF_LIBRARY_PATH`` environment variable (exact path).
2. ``libgowkhtmltopdf.{so,dylib,dll}`` next to this package (wheel layout).
3. ``<repo root>/dist/libgowkhtmltopdf.<ext>`` for in-tree builds.

The first existing candidate wins; when none exists ``find_library_path``
raises ``FileNotFoundError`` listing every path tried.

Memory ownership follows the committed header
``bindings/c/include/gowkhtmltopdf.h``: output bytes and error strings are
allocated by the library and must be released through
``gowkhtmltopdf_free`` / ``gowkhtmltopdf_free_string`` after Python copies
them. Input buffers are borrowed for the duration of a call only.

Engine calls are not documented as thread-affine, so every foreign call is
serialized through a module-wide lock. ctypes releases the GIL around each
CDLL call, which lets other Python threads progress while a long render
runs.
"""

import ctypes
import os
import sys
import threading
from pathlib import Path

from .exceptions import error_from_status

#: ABI revision this binding is compiled against. Must match the header.
ABI_VERSION = 1

_STATUS_OK = 0

_LOAD_LOCK = threading.Lock()
_CALL_LOCK = threading.Lock()

_LOADED_LIBRARY = None  # type: ctypes.CDLL


class GwkPdfOptions(ctypes.Structure):
    """Mirror of GwkPdfOptions from include/gowkhtmltopdf.h.

    Field order is pinned by the ABI contract; do not reorder or insert.
    """

    _fields_ = [
        ("abi_version", ctypes.c_int32),
        ("struct_size", ctypes.c_int32),
        ("page_size", ctypes.c_char_p),
        ("orientation", ctypes.c_char_p),
        ("title", ctypes.c_char_p),
        ("pdf_version", ctypes.c_char_p),
        ("pdf_profile", ctypes.c_char_p),
        ("base_url", ctypes.c_char_p),
        ("allow", ctypes.POINTER(ctypes.c_char_p)),
        ("allow_len", ctypes.c_size_t),
        ("width_mm", ctypes.c_double),
        ("height_mm", ctypes.c_double),
        ("margin_top", ctypes.c_double),
        ("margin_right", ctypes.c_double),
        ("margin_bottom", ctypes.c_double),
        ("margin_left", ctypes.c_double),
        ("copies", ctypes.c_int32),
        ("grayscale", ctypes.c_int32),
        ("enable_local_file_access", ctypes.c_int32),
        ("network_policy", ctypes.c_int32),
        ("timeout_ms", ctypes.c_int32),
    ]

    @classmethod
    def create(cls):
        # type: () -> GwkPdfOptions
        """Return a zeroed struct with the size gate fields filled."""
        instance = cls()
        instance.abi_version = ABI_VERSION
        instance.struct_size = ctypes.sizeof(cls)
        return instance


class GwkImageOptions(ctypes.Structure):
    """Mirror of GwkImageOptions from include/gowkhtmltopdf.h."""

    _fields_ = [
        ("abi_version", ctypes.c_int32),
        ("struct_size", ctypes.c_int32),
        ("format", ctypes.c_char_p),
        ("base_url", ctypes.c_char_p),
        ("allow", ctypes.POINTER(ctypes.c_char_p)),
        ("allow_len", ctypes.c_size_t),
        ("width", ctypes.c_int32),
        ("height", ctypes.c_int32),
        ("quality", ctypes.c_int32),
        ("smart_width", ctypes.c_int32),
        ("transparent", ctypes.c_int32),
        ("crop_left", ctypes.c_int32),
        ("crop_top", ctypes.c_int32),
        ("crop_width", ctypes.c_int32),
        ("crop_height", ctypes.c_int32),
        ("zoom", ctypes.c_double),
        ("enable_local_file_access", ctypes.c_int32),
        ("network_policy", ctypes.c_int32),
        ("timeout_ms", ctypes.c_int32),
    ]

    @classmethod
    def create(cls):
        # type: () -> GwkImageOptions
        """Return a zeroed struct with the size gate fields filled."""
        instance = cls()
        instance.abi_version = ABI_VERSION
        instance.struct_size = ctypes.sizeof(cls)
        return instance


def _library_filename():
    # type: () -> str
    if sys.platform == "darwin":
        return "libgowkhtmltopdf.dylib"
    if sys.platform == "win32":
        return "libgowkhtmltopdf.dll"
    return "libgowkhtmltopdf.so"


def candidate_paths():
    # type: () -> list
    """Return the loader's search candidates, highest priority first."""
    paths = []
    env_path = os.environ.get("GOWKHTMLTOPDF_LIBRARY_PATH")
    if env_path:
        paths.append(Path(env_path))
    filename = _library_filename()
    package_dir = Path(__file__).resolve().parent
    paths.append(package_dir / filename)
    try:
        # _lib.py sits at <root>/bindings/python/src/gowkhtmltopdf/, so
        # parents[4] is the repository root.
        repo_root = Path(__file__).resolve().parents[4]
        paths.append(repo_root / "dist" / filename)
    except IndexError:  # installed outside any repo-like tree
        pass
    return paths


def find_library_path():
    # type: () -> Path
    """Return the first existing shared-library candidate.

    Raises:
        FileNotFoundError: When no candidate exists on disk.
    """
    tried = []
    for path in candidate_paths():
        tried.append(str(path))
        if path.is_file():
            return path
    raise FileNotFoundError(
        "libgowkhtmltopdf not found; build it with"
        " 'CGO_ENABLED=1 go build -buildmode=c-shared -o dist/{0} ./bindings/c'"
        " or set GOWKHTMLTOPDF_LIBRARY_PATH. Tried: {1}".format(
            _library_filename(), ", ".join(tried)
        )
    )


def _bind_prototypes(lib):
    # type: (ctypes.CDLL) -> None
    ubyte_pp = ctypes.POINTER(ctypes.POINTER(ctypes.c_ubyte))
    size_p = ctypes.POINTER(ctypes.c_size_t)
    char_pp = ctypes.POINTER(ctypes.c_char_p)

    fn = lib.gowkhtmltopdf_html_to_pdf
    fn.restype = ctypes.c_int
    fn.argtypes = [
        ctypes.c_char_p,
        ctypes.c_size_t,
        ctypes.POINTER(GwkPdfOptions),
        ubyte_pp,
        size_p,
        char_pp,
    ]

    fn = lib.gowkhtmltopdf_html_to_image
    fn.restype = ctypes.c_int
    fn.argtypes = [
        ctypes.c_char_p,
        ctypes.c_size_t,
        ctypes.POINTER(GwkImageOptions),
        ubyte_pp,
        size_p,
        char_pp,
    ]

    fn = lib.gowkhtmltopdf_free
    fn.restype = None
    fn.argtypes = [ctypes.c_void_p]

    fn = lib.gowkhtmltopdf_free_string
    fn.restype = None
    fn.argtypes = [ctypes.c_char_p]

    fn = lib.gowkhtmltopdf_abi_version
    fn.restype = ctypes.c_int32
    fn.argtypes = []

    # Declared c_void_p instead of c_char_p so the raw pointer survives;
    # the header requires releasing it with gowkhtmltopdf_free_string.
    fn = lib.gowkhtmltopdf_version
    fn.restype = ctypes.c_void_p
    fn.argtypes = []

    fn = lib.gowkhtmltopdf_last_error_length
    fn.restype = ctypes.c_int32
    fn.argtypes = []

    fn = lib.gowkhtmltopdf_last_error
    fn.restype = ctypes.c_int32
    fn.argtypes = [ctypes.c_char_p, ctypes.c_int32]


def load_library():
    # type: () -> ctypes.CDLL
    """Load, prototype-bind, and ABI-check the shared library once.

    Raises:
        ImportError: When the library is missing or reports a foreign ABI.
    """
    global _LOADED_LIBRARY
    with _LOAD_LOCK:
        if _LOADED_LIBRARY is not None:
            return _LOADED_LIBRARY
        path = find_library_path()
        # CDLL uses RTLD_LOCAL by default, keeping Go runtime symbols out
        # of the global namespace.
        lib = ctypes.CDLL(str(path))
        _bind_prototypes(lib)
        reported = int(lib.gowkhtmltopdf_abi_version())
        if reported != ABI_VERSION:
            raise ImportError(
                "ABI mismatch: library {0}, binding expects {1}".format(
                    reported, ABI_VERSION
                )
            )
        _LOADED_LIBRARY = lib
        return _LOADED_LIBRARY


def abi_version():
    # type: () -> int
    """Return the ABI revision reported by the loaded library."""
    return int(load_library().gowkhtmltopdf_abi_version())


def library_version_string():
    # type: () -> str
    """Return the runtime version string, freeing the library allocation."""
    lib = load_library()
    ptr = 0
    try:
        with _CALL_LOCK:
            ptr = int(lib.gowkhtmltopdf_version() or 0)
        if not ptr:
            return ""
        text = ctypes.cast(ptr, ctypes.c_char_p).value
        return (text or b"").decode("utf-8", "replace")
    finally:
        if ptr:
            lib.gowkhtmltopdf_free_string(ctypes.cast(ptr, ctypes.c_char_p))


def _take_error_message(lib, err_ptr, status):
    # type: (ctypes.CDLL, ctypes.POINTER(ctypes.c_char_p), int) -> str
    """Copy and free the out_err string, falling back to the last-error slot."""
    text = b""
    if err_ptr:
        text = err_ptr.value or b""
        lib.gowkhtmltopdf_free_string(err_ptr)
    if not text:
        length = int(lib.gowkhtmltopdf_last_error_length())
        if length > 0:
            buf = ctypes.create_string_buffer(length + 1)
            lib.gowkhtmltopdf_last_error(buf, length + 1)
            text = buf.value
    message = text.decode("utf-8", "replace")
    return message or "conversion failed with status {0}".format(status)


def convert_html_to_pdf(html, opts=None):
    # type: (bytes, GwkPdfOptions) -> bytes
    """Run one PDF conversion and return owned bytes.

    Raises the mapped ConversionError subclass on any non-zero status.
    """
    if not isinstance(html, (bytes, bytearray)):
        raise TypeError("html must be bytes")
    html = bytes(html)
    lib = load_library()
    out_data = ctypes.POINTER(ctypes.c_ubyte)()
    out_len = ctypes.c_size_t(0)
    out_err = ctypes.c_char_p()
    with _CALL_LOCK:
        opts_ptr = ctypes.byref(opts) if opts is not None else None
        status = lib.gowkhtmltopdf_html_to_pdf(
            html, len(html), opts_ptr, ctypes.byref(out_data),
            ctypes.byref(out_len), ctypes.byref(out_err),
        )
    if status != _STATUS_OK:
        raise error_from_status(status, _take_error_message(lib, out_err, status))
    try:
        return ctypes.string_at(out_data, out_len.value)
    finally:
        lib.gowkhtmltopdf_free(ctypes.cast(out_data, ctypes.c_void_p))


def convert_html_to_image(html, opts=None):
    # type: (bytes, GwkImageOptions) -> bytes
    """Run one image conversion and return owned bytes."""
    if not isinstance(html, (bytes, bytearray)):
        raise TypeError("html must be bytes")
    html = bytes(html)
    lib = load_library()
    out_data = ctypes.POINTER(ctypes.c_ubyte)()
    out_len = ctypes.c_size_t(0)
    out_err = ctypes.c_char_p()
    with _CALL_LOCK:
        opts_ptr = ctypes.byref(opts) if opts is not None else None
        status = lib.gowkhtmltopdf_html_to_image(
            html, len(html), opts_ptr, ctypes.byref(out_data),
            ctypes.byref(out_len), ctypes.byref(out_err),
        )
    if status != _STATUS_OK:
        raise error_from_status(status, _take_error_message(lib, out_err, status))
    try:
        return ctypes.string_at(out_data, out_len.value)
    finally:
        lib.gowkhtmltopdf_free(ctypes.cast(out_data, ctypes.c_void_p))
