/**
 * @fileoverview JSDoc Type Definitions for gowkhtmltopdf documentation frontend.
 */

/**
 * @typedef {'implemented' | 'partial' | 'not-implemented'} CoverageStatus
 */

/**
 * @typedef {'High' | 'Medium' | 'Low'} SeverityLevel
 */

/**
 * @typedef {'CSS/layout' | 'JavaScript/interactive' | 'Images' | 'Fonts/encoding/text' | 'Page size/margins/header-footer' | 'Crash/hang/memory' | 'CLI/args' | 'Feature request' | 'Docs' | 'Other'} IssueCategory
 */

/**
 * @typedef {Object} Issue
 * @property {number} number - Upstream GitHub issue number
 * @property {string} title - Issue title
 * @property {string} summary - Short summary of the bug or capability
 * @property {IssueCategory} category - Issue classification area
 * @property {SeverityLevel} severity - Issue severity rating
 * @property {CoverageStatus} status - Implementation status in gowkhtmltopdf
 * @property {string} [workaround] - Known workaround if applicable
 * @property {string} [key_detail] - Key technical detail
 * @property {string} [evidence] - Concrete codebase path or test verifying the status
 * @property {string} [author] - Issue author
 * @property {string} [created_at] - Creation timestamp
 * @property {number} [comments] - Comment count
 */

/**
 * @typedef {'hero' | 'stats' | 'bullets' | 'cards' | 'prose' | 'code' | 'table' | 'callout' | 'toc'} ContentBlockType
 */

/**
 * @typedef {Object} ContentBlock
 * @property {ContentBlockType} type
 * @property {string} [title]
 * @property {string} [lede]
 * @property {string} [heading]
 * @property {string} [body]
 * @property {string} [code]
 * @property {string} [lang]
 * @property {string} [variant]
 * @property {Array<any>} [items]
 * @property {Array<any>} [sections]
 * @property {Array<string>} [headers]
 * @property {Array<Array<string>>} [rows]
 */

/**
 * @typedef {Object} ContentPage
 * @property {string} id - Unique identifier for the page
 * @property {string} nav - Navigation title
 * @property {Array<ContentBlock>} content - Array of content blocks
 */

/**
 * @typedef {'Invoices & receipts' | 'Reports & tables' | 'Storybooks & posters' | 'CSS & layout fixtures' | 'Architecture & API'} ShowcaseCategory
 */

/**
 * @typedef {Object} ShowcaseItem
 * @property {string} name - Base name of the sample (matches PNG thumb & source HTML)
 * @property {string} file - PDF output filename
 * @property {number} pages - Number of pages in the output
 * @property {string} title - Human readable display title
 * @property {string} desc - Description of the fixture
 * @property {ShowcaseCategory} category - Fixture category
 */

/**
 * @typedef {Object} BenchmarkRow
 * @property {number} pages - Document page count
 * @property {number} gowkMs - gowkhtmltopdf execution time in milliseconds
 * @property {number} wkMs - wkhtmltopdf execution time in milliseconds
 * @property {string} [notes] - Additional benchmark notes
 */

export {}
