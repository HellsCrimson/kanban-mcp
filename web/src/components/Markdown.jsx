import { useMemo } from 'react'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

// breaks: a single newline is a line break — descriptions are written like
// notes, not like prose that reflows.
marked.use({ gfm: true, breaks: true })

// Links in a description point somewhere else; open them in a new tab so the
// board (and any unsaved edit) stays put.
DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A' && node.getAttribute('href')) {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
})

export default function Markdown({ text }) {
  const html = useMemo(() => DOMPurify.sanitize(marked.parse(text || '')), [text])
  return <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />
}
