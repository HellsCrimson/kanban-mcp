import { useEffect, useState } from 'react'
import { api } from '../api.js'

const PRIORITIES = ['low', 'medium', 'high', 'urgent']

export default function CardModal({ initial, columns, onClose, onChanged, onOpenCard }) {
  const isCreate = initial.mode === 'create'
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [complexity, setComplexity] = useState(1)
  const [priority, setPriority] = useState('medium')
  const [type, setType] = useState(isCreate ? initial.cardType : 'task')
  const [detail, setDetail] = useState(null)
  const [subTitle, setSubTitle] = useState('')
  const [subColumn, setSubColumn] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (isCreate) return
    setDetail(null)
    api
      .card(initial.cardId)
      .then((d) => {
        setDetail(d)
        setTitle(d.card.title)
        setDescription(d.card.description)
        setComplexity(d.card.complexity)
        setPriority(d.card.priority)
        setType(d.card.type)
        setSubColumn(d.card.column_id)
      })
      .catch((e) => setError(e.message))
  }, [initial, isCreate])

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

  const isEpic = type === 'epic'

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
