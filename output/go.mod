// Separate module so output/ sample PDFs are not packed into the parent
// module zip (go install of the CLIs does not need committed samples).
module github.com/chinmay-sawant/gowkhtmltopdf/output

go 1.26
