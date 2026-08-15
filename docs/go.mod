// Separate module so docs/ (GitHub Pages build) is not packed into the
// parent module zip (go install of the CLIs does not need the site).
module github.com/chinmay-sawant/gowkhtmltopdf/docs

go 1.26
