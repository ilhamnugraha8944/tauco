import type { SourceReference } from "@/features/content";

export function SourceList({
  sources,
  heading = "Referensi",
}: {
  sources: SourceReference[];
  heading?: string;
}) {
  return (
    <section className="source-list" aria-labelledby="source-heading">
      <h2 id="source-heading">{heading}</h2>
      <ol>
        {sources.map((source) => (
          <li key={source.url}>
            <a href={source.url} target="_blank" rel="noreferrer">
              <span>{source.label}</span>
              <small>{source.publisher}</small>
            </a>
          </li>
        ))}
      </ol>
    </section>
  );
}
