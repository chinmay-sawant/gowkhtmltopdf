export default function TableBlock({ block }) {
  return (
    <section className="table-block">
      {block.heading && <h3 className="table-block-heading">{block.heading}</h3>}
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              {block.headers.map((h, i) => (
                <th key={i}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {block.rows.map((row, i) => (
              <tr key={i}>
                {row.map((cell, j) => (
                  <td key={j}>{cell}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
