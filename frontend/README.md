# gowkhtmltopdf: Documentation site (React)

A minimal editorial product-site for the **gowkhtmltopdf** project: a pure-Go,
no-cgo HTML-to-PDF/PNG renderer with no browser or external converter process,
and a clean-room work-alike of
[wkhtmltopdf](https://github.com/wkhtmltopdf/wkhtmltopdf).

The site tells two stories:

1. **What the product is**: Overview, Getting Started, CLI, Library API,
   Architecture, Compatibility, Fonts, Security, Performance, About, and a
   first-class **Benchmarks** tab (generic CLI versus wkhtmltopdf). The
   long-form content in `src/data/content/*.json` is grounded in the
   repository documentation and the checked-in benchmark snapshot.
2. **What the open issue backlog looks like**: the **Issue Dossier** page
   tracks **all 1,329 open issues** of `wkhtmltopdf/wkhtmltopdf`. The first 100
   are tagged with whether **gowkhtmltopdf** already implements, partially
   implements, or does not implement the underlying capability (green / amber /
   red); the remaining 1,229 rows are unassessed raw metadata. The dossier is
   filterable by coverage status and area, and paginated (10, 25, 50, or 100
   per page).

The site ships with a **light and dark theme** (persisted in
`localStorage`, respecting `prefers-color-scheme` by default).

Built to deploy as a static site on **GitHub Pages** from the repo's `docs/`
folder (base path `/gowkhtmltopdf/`).

## Stack

- [Vite](https://vite.dev) + [React](https://react.dev) + [React Router](https://reactrouter.com)
- Plain CSS in `src/styles/global.css`: editorial warm-monochrome style, no UI framework, with a `[data-theme=dark]` variable block

## Folder structure

```
frontend/
├── index.html                 # Vite entry shell
├── vite.config.js             # base path + build config
├── scripts/
│   └── copy-to-docs.mjs       # copies dist/ to ../docs + .nojekyll
└── src/
    ├── main.jsx               # React bootstrap
    ├── App.jsx                # router + route table
    ├── components/
    │   ├── RichText.jsx       # turn `backticks` into themed inline code
    │   ├── SiteNav.jsx        # sticky nav + theme toggle
    │   ├── Footer.jsx
    │   ├── PageTitle.jsx      # per-route document title
    │   ├── Pagination.jsx     # dossier pagination controls
    │   ├── blocks/            # generic content-block renderers
    │   │   ├── ContentBlocks.jsx   # dispatches by block.type
    │   │   ├── HeroBlock.jsx / StatsBlock.jsx / CardsBlock.jsx
    │   │   ├── ProseBlock.jsx / CodeBlock.jsx / TableBlock.jsx
    │   │   ├── BulletsBlock.jsx / CalloutBlock.jsx / TocBlock.jsx
    │   │   └── slugify.js     # derive anchor ids from headings
    │   ├── FilterChips.jsx    # dossier: status + area filter chips
    │   ├── IssueCard.jsx      # dossier: colour-coded issue card
    │   └── StatsSidebar.jsx   # dossier: coverage / area / severity breakdown
    ├── pages/
    │   ├── ContentPage.jsx    # renders any content JSON page by id
    │   ├── BenchmarksPage.jsx # CLI vs wkhtmltopdf comparison
    │   └── DossierPage.jsx    # interactive, paginated issue dashboard
    └── data/
        ├── issues.json        # all 1,329 open issues (100 analyzed deeply)
        ├── issues.js          # data helpers
        ├── benchmarks.js      # CLI vs wkhtmltopdf snapshot for the Benchmarks tab
        ├── constants.js       # status / severity / category metadata
        └── content/           # long-form page content (page-*.json)
```

## Content block schema

Each `src/data/content/page-*.json` is `{ id, nav, content: [...] }` where
`content` is an array of blocks: `toc`, `hero`, `stats`, `bullets`, `cards`,
`prose`, `code`, `table`, or `callout`. See the generator schema in
`/tmp/opencode/content/SCHEMA.md` for the exact shapes. Adding a new page =
drop a `page-*.json` file, add a route in `App.jsx`, and a nav link in
`SiteNav.jsx`.

## Commands

```sh
npm install
npm run dev       # local dev server (http://localhost:5173/gowkhtmltopdf/)
npm run build     # build → dist/ then copy → ../docs (GitHub Pages)
npm run preview   # preview the production build
```

## Deploying to GitHub Pages

1. Run `npm run build` (populates `../docs` with the compiled site + `.nojekyll`).
2. On GitHub, enable Pages with **Source: Deploy from a branch → branch
   `main` → folder `/docs`**.
3. The site is served at `https://<user>.github.io/gowkhtmltopdf/`. Override
   the base path with `VITE_BASE_PATH=/whatever/` if you republish elsewhere.

## Updating the data

- **Dossier data:** replace the array in `src/data/issues.json` (shape:
  `number`, `title`, `summary`, `category`, `severity`, `status`,
  `workaround`, `key_detail`, `evidence`), then rebuild.
- **Product content:** edit the matching `src/data/content/page-*.json`, then
  rebuild. Keep the ASCII-only rule (no em dashes, no emojis) that the content
  was generated under.
