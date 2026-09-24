# Ghostscript licensing

This page explains when Ghostscript licensing applies to gowkhtmltopdf. It covers this repository's tests and cases where an application distributes or hosts Ghostscript. I checked the linked license pages on 2026-09-24.

## How this repository uses Ghostscript

The `Document` API and native CLI use the in-repo conversion pipeline. They do not start Ghostscript (`README.md:11-16`). The optional visual test starts the separately installed `gs` executable named by `GOWKHTMLTOPDF_VISUAL_GS`, checks its version, and renders PDFs to PNG (`internal/convert/fixturetests/pixel_regression_test.go:62-100`). The corpus test skips when that variable is empty (`internal/convert/fixturetests/pixel_regression_corpus_test.go:80-84`). Ordinary `make test` runs skip the visual comparison unless you set the variable (`output/README.md:70-71`).

## Using gowkhtmltopdf in a commercial application

The repository's `LICENSE` is MIT. It permits commercial use, copying, modification, and distribution. Anyone distributing copies or substantial portions must include the copyright and permission notices (`LICENSE:1-13`). The library and binaries do not call Ghostscript, so using gowkhtmltopdf alone does not require an Artifex license.

Installing Ghostscript on the same computer does not change the license of an application that does not call it. A commercial machine does not, by itself, require a paid Ghostscript license.

## Running the optional tests locally

Ghostscript is available under the GNU Affero General Public License (AGPL) or an Artifex commercial license. Artifex's [Ghostscript FAQ](https://ghostscript.com/faq/) says you may use an unchanged AGPL release if you do not distribute it. This covers local visual tests with a separately installed copy that you do not ship. A computer used for commercial work does not by itself trigger a purchase requirement.

## Distributing or hosting Ghostscript

Artifex's FAQ directs closed-source proprietary distribution and SaaS use to its commercial license. Artifex's [licensing page](https://artifex.com/licensing) says that using its AGPL release as part of a server application or service requires sharing the full application source with users who interact with it. It says you need a commercial license if you cannot meet those AGPL conditions.

GNU AGPL version 3, section 13, says that if you modify the covered program and users interact with that modified version remotely, you must offer those users its corresponding source code ([GNU AGPLv3](https://www.gnu.org/licenses/agpl-3.0.html)). Artifex's Ghostscript-specific guidance also addresses server applications and services. Check with Artifex if your product bundles Ghostscript or calls it at runtime.

The FAQ discusses Ghostscript use in software that is distributed. It does not give a separate-process exception. Treat a product that starts `gs` at runtime as a separate licensing case and confirm it with Artifex before distribution. This is a cautious reading of the FAQ's wording.

## Scenario summary

| Scenario | What the published terms say |
|----------|-------------------------------|
| Use only gowkhtmltopdf's library or CLI in a commercial application | The project uses MIT. Ghostscript is not part of this path. |
| Run local visual tests with unchanged, separately installed Ghostscript and do not distribute Ghostscript | Artifex says its AGPL release may be used without a commercial license in this case. |
| Bundle or distribute Ghostscript with a closed-source product | Artifex directs this use to a commercial license unless the AGPL terms fit and are followed. |
| Use Ghostscript as part of a server application or service | Artifex's published guidance calls for full application source disclosure under AGPL or a commercial license. |
