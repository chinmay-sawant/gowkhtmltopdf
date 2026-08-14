import RichText from '../RichText'
import { slugify } from './slugify'

export default function TableBlock({ block }) {
  return (
    <section className="table-block">
      {block.heading && (
        <h3 className="table-block-heading" id={slugify(block.heading)}>
          <RichText>{block.heading}</RichText>
        </h3>
      )}
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              {block.headers.map((h, i) => (
                <th key={i}>
                  <RichText>{h}</RichText>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {block.rows.map((row, i) => (
              <tr key={i}>
                {row.map((cell, j) => (
                  <td key={j}>
                    <RichText>{cell}</RichText>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
