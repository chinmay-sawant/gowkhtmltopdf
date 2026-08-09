export default function CodeBlock({ block }) {
  return (
    <section className="code-block">
      {block.heading && <h3 className="code-block-heading">{block.heading}</h3>}
      <pre className={`lang-${block.language}`}>
        <code>{block.code}</code>
      </pre>
    </section>
  )
}
