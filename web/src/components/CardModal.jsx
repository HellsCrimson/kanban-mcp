import { useCallback, useEffect, useState } from 'react'
import { api } from '../api.js'
import { useLiveUpdates } from '../useLiveUpdates.js'

const PRIORITIES = ['low', 'medium', 'high', 'urgent']
const REL_TYPES = [
  ['blocks', 'blocks'],
  ['depends_on', 'depends on'],
  ['relates_to', 'relates to'],
]

// Native <select> widens itself to its longest option; keep labels short so
// the control stays inside the modal.
function truncate(text, max = 40) {
  return text.length > max ? text.slice(0, max - 1) + '…' : text
}

function relLabel(rc) {
  const other = `#${rc.number} ${rc.title}`
  if (rc.type === 'relates_to') return `related to ${other}`
  if (rc.direction === 'outgoing') {
    return rc.type === 'blocks' ? `blocks ${other}` : `depends on ${other}`
  }
  return rc.type === 'blocks' ? `blocked by ${other}` : `depended on by ${other}`
}

export default function CardModal({ initial, columns, cards, onClose, onChanged, onOpenCard }) {
  const isCreate = initial.mode === 'create'
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [complexity, setComplexity] = useState(1)
  const [priority, setPriority] = useState('medium')
  const [type, setType] = useState(initial.cardType ?? 'task')
  const [detail, setDetail] = useState(null)
  const [subTitle, setSubTitle] = useState('')
  const [subColumn, setSubColumn] = useState('')
  const [comment, setComment] = useState('')
  const [relType, setRelType] = useState('blocks')
  const [relTarget, setRelTarget] = useState('')
  const [error, setError] = useState('')

  // withForm is false for background refreshes so a change coming in over SSE
  // never overwrites what the user is currently typing.
  const loadDetail = useCallback(
    async (withForm) => {
      try {
        const d = await api.card(initial.cardId)
        setDetail(d)
        if (withForm) {
          setTitle(d.card.title)
          setDescription(d.card.description)
          setComplexity(d.card.complexity)
          setPriority(d.card.priority)
          setType(d.card.type)
          setSubColumn(d.card.column_id)
        }
      } catch (e) {
        setError(e.message)
      }
    },
    [initial.cardId],
  )

  useEffect(() => {
    if (isCreate) return
    setDetail(null)
    loadDetail(true)
  }, [isCreate, loadDetail])

  useLiveUpdates(initial.projectId, () => loadDetail(false), { enabled: !isCreate })

  async function submit(e) {
    e.preventDefault()
    if (!title.trim()) return
    setError('')
    try {
      if (isCreate) {
        await api.createCard(initial.projectId, {
          column_id: initial.columnId,
          title: title.trim(),
          description,
          complexity,
          priority,
          type,
        })
      } else {
        const patch = {}
        if (detail) {
          if (title.trim() !== detail.card.title) patch.title = title.trim()
          if (description !== detail.card.description) patch.description = description
          if (complexity !== detail.card.complexity) patch.complexity = complexity
          if (priority !== detail.card.priority) patch.priority = priority
          if (type !== detail.card.type) patch.type = type
        }
        if (Object.keys(patch).length > 0) await api.updateCard(initial.cardId, patch)
      }
      onChanged()
      onClose()
    } catch (err) {
      setError(err.message)
    }
  }

  async function addSubcard(e) {
    e.preventDefault()
    if (!subTitle.trim()) return
    setError('')
    try {
      await api.createCard(initial.projectId, {
        column_id: subColumn || detail.card.column_id,
        title: subTitle.trim(),
        parent_id: initial.cardId,
      })
      setSubTitle('')
      const d = await api.card(initial.cardId)
      setDetail(d)
      onChanged()
    } catch (err) {
      setError(err.message)
    }
  }

  async function remove() {
    if (!confirm('Delete this card? Sub-cards of an epic are deleted with it.')) return
    try {
      await api.deleteCard(initial.cardId)
      onChanged()
      onClose()
    } catch (err) {
      setError(err.message)
    }
  }

  async function refresh() {
    await loadDetail(false)
  }

  async function addComment() {
    if (!comment.trim()) return
    setError('')
    try {
      await api.addUpdate(initial.cardId, comment.trim())
      setComment('')
      await refresh()
    } catch (err) {
      setError(err.message)
    }
  }

  async function addRelationship() {
    if (!relTarget) return
    setError('')
    try {
      await api.addRelationship({ source_id: initial.cardId, target_id: relTarget, type: relType })
      setRelTarget('')
      await refresh()
    } catch (err) {
      setError(err.message)
    }
  }

  async function removeRelationship(relId) {
    try {
      await api.deleteRelationship(relId)
      await refresh()
    } catch (err) {
      setError(err.message)
    }
  }

  const isEpic = type === 'epic'
  const targetCards = (cards || [])
    .filter((c) => c.id !== initial.cardId)
    .sort((a, b) => a.number - b.number)

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <form onSubmit={submit}>
          <div className="modal-header">
            <h2>{isCreate ? 'New card' : `#${detail?.card.number ?? ''} ${detail ? '' : '…'}`}</h2>
            <button type="button" className="icon-btn" onClick={onClose}>×</button>
          </div>

          {error && <div className="banner error">{error}</div>}

          <label>
            Title
            <input value={title} onChange={(e) => setTitle(e.target.value)} autoFocus required />
          </label>

          <div className="field-row">
            <label>
              Type
              <select value={type} onChange={(e) => setType(e.target.value)}>
                <option value="task">Task</option>
                <option value="epic">Epic</option>
              </select>
            </label>
            <label>
              Priority
              <select value={priority} onChange={(e) => setPriority(e.target.value)}>
                {PRIORITIES.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
            </label>
            <label>
              Complexity (1-10)
              <input type="number" min="1" max="10" value={complexity} onChange={(e) => setComplexity(Number(e.target.value))} />
            </label>
          </div>

          <label>
            Description
            <textarea rows="4" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Details, acceptance criteria…" />
          </label>

          {!isCreate && isEpic && detail && (
            <div className="subcards">
              <div className="subcards-header">
                Sub-cards ({detail.card.sub_done}/{detail.card.sub_total})
              </div>
              {detail.subcards.map((sc) => (
                <div key={sc.card.id} className={'subcard' + (sc.done ? ' done' : '')} onClick={() => onOpenCard(sc.card.id)}>
                  <span className="card-number">#{sc.card.number}</span>
                  <span className="subcard-title">{sc.card.title}</span>
                  <span className="muted">{sc.column_name}</span>
                </div>
              ))}
              <div className="subcard-form">
                <input
                  value={subTitle}
                  onChange={(e) => setSubTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      addSubcard(e)
                    }
                  }}
                  placeholder="New sub-card title"
                />
                <select value={subColumn} onChange={(e) => setSubColumn(e.target.value)}>
                  {columns.map((c) => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
                <button type="button" disabled={!subTitle.trim()} onClick={addSubcard}>Add</button>
              </div>
            </div>
          )}

          {!isCreate && detail && (
            <div className="section">
              <div className="section-header">Updates</div>
              {detail.updates?.length ? (
                detail.updates.map((u) => (
                  <div key={u.id} className="update-item">
                    <div className="update-content">{u.content}</div>
                    <div className="muted">{new Date(u.created_at).toLocaleString()}</div>
                  </div>
                ))
              ) : (
                <div className="muted">No updates yet.</div>
              )}
              <div className="section-form">
                <textarea
                  rows="2"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      addComment()
                    }
                  }}
                  placeholder="Add a comment… (Enter to submit)"
                />
                <button type="button" disabled={!comment.trim()} onClick={addComment}>Add</button>
              </div>
            </div>
          )}

          {!isCreate && detail && (
            <div className="section">
              <div className="section-header">Relationships</div>
              {detail.relationships?.length ? (
                detail.relationships.map((rc) => (
                  <div key={rc.rel_id} className="rel-row">
                    <span className="rel-text" onClick={() => onOpenCard(rc.card_id)}>{relLabel(rc)}</span>
                    <button type="button" className="icon-btn" title="Remove link" onClick={() => removeRelationship(rc.rel_id)}>×</button>
                  </div>
                ))
              ) : (
                <div className="muted">No relationships yet.</div>
              )}
              <div className="section-form">
                <select className="rel-type" value={relType} onChange={(e) => setRelType(e.target.value)}>
                  {REL_TYPES.map(([value, label]) => (
                    <option key={value} value={value}>{label}</option>
                  ))}
                </select>
                <select className="rel-target" value={relTarget} onChange={(e) => setRelTarget(e.target.value)}>
                  <option value="">Link to…</option>
                  {targetCards.map((c) => (
                    <option key={c.id} value={c.id}>#{c.number} {truncate(c.title)}</option>
                  ))}
                </select>
                <button type="button" disabled={!relTarget} onClick={addRelationship}>Add</button>
              </div>
            </div>
          )}

          <div className="modal-actions">
            {!isCreate && (
              <button type="button" className="danger" onClick={remove}>Delete</button>
            )}
            <span className="spacer" />
            <button type="button" className="ghost" onClick={onClose}>Cancel</button>
            <button type="submit">{isCreate ? 'Create' : 'Save'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
