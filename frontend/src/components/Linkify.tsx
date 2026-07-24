import { Fragment } from 'react';

// Captures the URL in a single group so String.split() interleaves matches at
// odd indices -- avoids re-testing the (stateful, lastIndex-tracking) global
// regex per chunk. Stops at whitespace and characters that commonly trail a
// URL in prose (closing punctuation/quotes) without escaping it.
const URL_PATTERN = /(https?:\/\/[^\s<>"')\]]+|www\.[^\s<>"')\]]+)/gi;

/** Renders `text` as plain text with bare URLs turned into clickable links. */
export function Linkify({ text }: { text: string }) {
  const parts = text.split(URL_PATTERN);
  return (
    <>
      {parts.map((part, i) => {
        if (i % 2 === 0) return <Fragment key={i}>{part}</Fragment>;
        const href = part.startsWith('www.') ? `https://${part}` : part;
        return (
          <a key={i} href={href} target="_blank" rel="noopener noreferrer">
            {part}
          </a>
        );
      })}
    </>
  );
}
