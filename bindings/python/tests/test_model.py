"""Model-level tests. These never touch the shared library."""

import dataclasses
import unittest

from gowkhtmltopdf import (
    Content,
    Crop,
    Document,
    ErrEmptyContent,
    ErrInvalidContent,
    ErrInvalidOrientation,
    ErrInvalidPDFVersion,
    ErrInvalidPageSize,
    ErrNoPageObjects,
    ImageDocument,
    ImageOptions,
    InvalidArgumentError,
    Margin,
    NetworkPolicy,
    PDFOptions,
    Page,
    compatible_network_policy,
    restricted_network_policy,
)
from gowkhtmltopdf.exceptions import (
    ConversionError,
    error_from_status,
    sniff_sentinel,
)


class ContentValidationTest(unittest.TestCase):
    def test_no_source_raises_empty(self):
        with self.assertRaises(ErrEmptyContent):
            Content().validate()

    def test_two_sources_raise_invalid(self):
        with self.assertRaises(ErrInvalidContent):
            Content(html=b"<p>x</p>", url="https://example.com").validate()

    def test_all_three_sources_raise_invalid(self):
        content = Content(
            html=b"<p>x</p>",
            file="a.html",
            url="https://example.com",
        )
        with self.assertRaises(ErrInvalidContent):
            content.validate()

    def test_empty_html_bytes_raise(self):
        with self.assertRaises(ErrEmptyContent):
            Content(html="").validate()

    def test_base_requires_html(self):
        with self.assertRaises(ErrInvalidContent):
            Content(file="page.html", base="/assets").validate()

    def test_non_http_url_rejected(self):
        with self.assertRaises(ErrInvalidContent):
            Content(url="ftp://example.com/doc").validate()

    def test_relative_url_rejected(self):
        with self.assertRaises(ErrInvalidContent):
            Content(url="example.com/doc").validate()

    def test_valid_html_and_url_pass(self):
        Content(html="<h1>ok</h1>").validate()
        Content(url="https://example.com/doc").validate()

    def test_html_bytes_property_encodes_utf8(self):
        self.assertEqual(Content(html="caf\u00e9").html_bytes,
                         "caf\u00e9".encode("utf-8"))
        self.assertEqual(Content(html=b"raw").html_bytes, b"raw")

    def test_classmethod_constructors(self):
        self.assertEqual(Content.from_html(b"x").kind, "html")
        self.assertEqual(Content.from_file("f.html").kind, "file")
        self.assertEqual(Content.from_url("https://e.com").kind, "url")


class DocumentValidationTest(unittest.TestCase):
    def _page(self):
        return Page(source=Content(html=b"<h1>ok</h1>"))

    def test_no_pages_or_cover_raises(self):
        with self.assertRaises(ErrNoPageObjects):
            Document(pages=[]).validate()

    def test_cover_alone_is_enough(self):
        Document(cover=self._page()).validate()

    def test_unknown_page_size_rejected(self):
        doc = Document(pages=[self._page()], page_size="Bogus")
        with self.assertRaises(ErrInvalidPageSize):
            doc.validate()

    def test_page_size_case_insensitive(self):
        Document(pages=[self._page()], page_size="LETTER").validate()

    def test_bad_orientation_rejected(self):
        doc = Document(pages=[self._page()], orientation="sideways")
        with self.assertRaises(ErrInvalidOrientation):
            doc.validate()

    def test_orientation_accepts_both_values(self):
        Document(pages=[self._page()], orientation="portrait").validate()
        Document(pages=[self._page()], orientation="LANDSCAPE").validate()

    def test_bad_pdf_version_rejected(self):
        doc = Document(pages=[self._page()], pdf_version="1.3")
        with self.assertRaises(ErrInvalidPDFVersion):
            doc.validate()

    def test_pdf_versions_accepted(self):
        for version in ("", "1.4", "1.7", "2.0"):
            Document(pages=[self._page()], pdf_version=version).validate()

    def test_copies_bounds(self):
        doc = Document(pages=[self._page()], copies=1001)
        with self.assertRaises(InvalidArgumentError):
            doc.validate()
        doc = Document(pages=[self._page()], copies=-1)
        with self.assertRaises(InvalidArgumentError):
            doc.validate()
        doc = Document(pages=[self._page()], copies=True)
        with self.assertRaises(InvalidArgumentError):
            doc.validate()

    def test_copies_zero_and_max_ok(self):
        Document(pages=[self._page()], copies=0).validate()
        Document(pages=[self._page()], copies=1000).validate()


class ImageDocumentValidationTest(unittest.TestCase):
    def _doc(self, format_value):
        return ImageDocument(
            source=Content(html=b"<h1>badge</h1>"), format=format_value
        )

    def test_supported_formats_pass(self):
        for value in ("png", "jpg", "jpeg", "", "PNG"):
            self._doc(value).validate()

    def test_unsupported_format_rejected(self):
        with self.assertRaises(InvalidArgumentError):
            self._doc("gif").validate()


class SentinelSniffingTest(unittest.TestCase):
    def test_each_substring_maps_to_its_sentinel(self):
        from gowkhtmltopdf import exceptions as exc_module

        expectations = [
            ("gowkhtmltopdf: empty HTML", exc_module.ErrEmptyContent),
            (
                "exactly one of HTML, File, or URL is required",
                exc_module.ErrInvalidContent,
            ),
            ("gowkhtmltopdf: no page objects added",
             exc_module.ErrNoPageObjects),
            ("no renderable PDF objects", exc_module.ErrNoPageObjects),
            ('gowkhtmltopdf: invalid page size: "Bogus"',
             exc_module.ErrInvalidPageSize),
            ('invalid orientation "diagonal"',
             exc_module.ErrInvalidOrientation),
            ("pdf version 9.9 not supported",
             exc_module.ErrInvalidPDFVersion),
            ("profile pdf/x-9 unknown", exc_module.ErrInvalidPDFProfile),
            ("missing output sink", exc_module.ErrMissingOutput),
        ]
        for message, expected in expectations:
            err = error_from_status(1, message)
            self.assertIsInstance(err, InvalidArgumentError, message)
            self.assertIs(err.sentinel, expected, message)

    def test_unmatched_message_has_no_sentinel(self):
        err = error_from_status(1, "something odd went wrong")
        self.assertIsNone(err.sentinel)

    def test_sniff_is_case_insensitive(self):
        self.assertIs(
            sniff_sentinel("Page Size unknown"), ErrInvalidPageSize
        )

    def test_status_codes_map_to_classes(self):
        mapping = {
            2: "LoadDeniedError",
            3: "RenderError",
            4: "ConversionTimeoutError",
            5: "ResourceLimitError",
            6: "InternalEngineError",
        }
        for code, name in mapping.items():
            err = error_from_status(code, "msg")
            self.assertEqual(type(err).__name__, name)
            self.assertEqual(err.code, code)
            self.assertEqual(err.message, "msg")

    def test_unknown_code_returns_plain_conversion_error(self):
        err = error_from_status(9, "weird")
        self.assertNotIsInstance(err, InvalidArgumentError)
        self.assertIsInstance(err, ConversionError)
        self.assertEqual(err.code, 9)


class OptionsDefaultsTest(unittest.TestCase):
    def test_pdf_options_defaults(self):
        opts = PDFOptions()
        self.assertEqual(opts.page_size, "A4")
        self.assertEqual(opts.orientation, "portrait")
        self.assertEqual(opts.width_mm, 0)
        self.assertEqual(opts.height_mm, 0)
        self.assertIsNone(opts.margin)
        self.assertEqual(opts.title, "")
        self.assertEqual(opts.pdf_version, "")
        self.assertEqual(opts.pdf_profile, "")
        self.assertEqual(opts.copies, 0)
        self.assertFalse(opts.grayscale)
        self.assertIsNone(opts.allow)
        self.assertFalse(opts.allow_local_files)
        self.assertIsNone(opts.network)
        self.assertIsNone(opts.base_url)
        self.assertEqual(opts.timeout_ms, 0)

    def test_image_options_defaults(self):
        opts = ImageOptions()
        self.assertEqual(opts.format, "png")
        self.assertEqual(opts.width, 0)
        self.assertEqual(opts.height, 0)
        self.assertEqual(opts.quality, 94)
        self.assertIsNone(opts.smart_width)
        self.assertFalse(opts.transparent)
        self.assertIsNone(opts.crop)
        self.assertEqual(opts.zoom, 0)

    def test_update_returns_new_instance(self):
        base = PDFOptions(page_size="A4")
        changed = base.update(page_size="Letter", title="t")
        self.assertEqual(changed.page_size, "Letter")
        self.assertEqual(changed.title, "t")
        self.assertEqual(base.page_size, "A4")
        self.assertIsNot(base, changed)

    def test_to_kwargs_drops_timeout_ms(self):
        kwargs = PDFOptions(timeout_ms=5).to_kwargs()
        self.assertNotIn("timeout_ms", kwargs)
        self.assertIn("page_size", kwargs)


class NetworkPolicyFactoryTest(unittest.TestCase):
    def test_compatible_policy_is_permissive(self):
        policy = compatible_network_policy()
        self.assertIsInstance(policy, NetworkPolicy)
        self.assertFalse(policy.block_private_networks)
        self.assertFalse(policy.block_cross_host_redirects)
        self.assertFalse(policy.restricted)

    def test_restricted_policy_blocks(self):
        policy = restricted_network_policy()
        self.assertTrue(policy.block_private_networks)
        self.assertTrue(policy.block_cross_host_redirects)
        self.assertTrue(policy.restricted)


class DataclassShapeTest(unittest.TestCase):
    def test_margin_defaults_zero(self):
        margin = Margin()
        self.assertEqual(
            (margin.top, margin.right, margin.bottom, margin.left),
            (0, 0, 0, 0),
        )

    def test_crop_defaults_no_crop_axes(self):
        crop = Crop()
        self.assertEqual((crop.left, crop.top, crop.width, crop.height),
                         (-1, -1, -1, -1))

    def test_struct_size_gate_fields_exist_on_model_dataclasses(self):
        # Cheap guard that the dataclasses keep their declared fields.
        self.assertEqual(len(dataclasses.fields(PDFOptions)), 15)
        self.assertEqual(len(dataclasses.fields(ImageOptions)), 13)


if __name__ == "__main__":
    unittest.main()
