import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api.js'
import { useLiveUpdates } from '../useLiveUpdates.js'
import ColumnView from './ColumnView.jsx'
import CardModal from './CardModal.jsx'

export default function Board({ projectId, onBack }) {
  const [board, setBoard] = useState(null)
  const [error, setError] = useState('')
  const [modal, setModal] = useState(null)
  // drag is { kind: 'card' | 'column', id }; target is where it would land now.
  const [drag, setDrag] = useState(null)
  const [cardTarget, setCardTarget] = useState(null)
  const [columnTarget, setColumnTarget] = useState(null)
  const boardRef = useRef(null)

  const load = useCallback(async () => {
    try {
      setBoard(await api.board(projectId))
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }, [projectId])

  useEffect(() => {
    load()
  }, [load])

  // Refreshing mid-drag would pull the cards out from under the pointer, so
  // live updates wait until the drag is over.
  useLiveUpdates(projectId, load, { paused: drag !== null })

  async function mutate(fn) {
    try {
      await fn()
      await load()
    } catch (e) {
      alert(e.message)
    }
  }

  function endDrag() {
    setDrag(null)
    setCardTarget(null)
    setColumnTarget(null)
  }

  /**
   * index is the slot in the column's displayed card list. The API counts
   * positions among the column's *other* cards, so a card moving down within
   * its own column shifts by one once it is lifted out.
   */
  function dropCard(cardId, toColumnId, index) {
    const col = board?.columns.find((c) => c.column.id === toColumnId)
    if (!col) return
    const ids = col.cards.map((c) => c.id)
    const from = ids.indexOf(cardId)
    let position = index ?? ids.length
    if (from !== -1) {
      if (position > from) position -= 1
      if (position === from) return
    }
    position = Math.max(0, Math.min(position, from === -1 ? ids.length : ids.length - 1))
    mutate(() => api.moveCard(cardId, toColumnId, position))
  }

  function dropColumn(columnId, index) {
    if (!board) return
    const ids = board.columns.map((c) => c.column.id)
    const from = ids.indexOf(columnId)
    let position = index
    if (from !== -1) {
      if (position > from) position -= 1
      if (position === from) return
    }
    position = Math.max(0, Math.min(position, ids.length - 1))
    mutate(() => api.updateColumn(columnId, { position }))
  }

  // Insertion slot for a dragged column: the first column whose horizontal
  // midpoint is to the right of the pointer.
  function columnIndexAt(clientX) {
    const el = boardRef.current
    if (!el) return board.columns.length
    const items = [...el.querySelectorAll('[data-column-id]')]
    for (let i = 0; i < items.length; i++) {
      const r = items[i].getBoundingClientRect()
      if (clientX < r.left + r.width / 2) return i
    }
    return items.length
  }

  function addColumn() {
    const name = prompt('Column name:')
    if (!name || !name.trim()) return
    mutate(() => api.addColumn(projectId, name.trim()))
  }

  if (error) return <div className="banner error center">{error}</div>
  if (!board) return <div className="loading">Loading…</div>

  const draggingColumn = drag?.kind === 'column'

  return (
    <div className="board-page">
      <div className="topbar">
        <button className="ghost" onClick={onBack}>← Projects</button>
        <span className="topbar-title">{board.project.name}</span>
        {board.project.description && <span className="topbar-desc">{board.project.description}</span>}
        <span className="spacer" />
        <button onClick={addColumn}>+ Column</button>
      </div>
      <div
        className="board"
        ref={boardRef}
        onDragOver={(e) => {
          if (!draggingColumn) return
          e.preventDefault()
          setColumnTarget(columnIndexAt(e.clientX))
        }}
        onDrop={(e) => {
          if (!draggingColumn) return
          e.preventDefault()
          dropColumn(drag.id, columnIndexAt(e.clientX))
          endDrag()
        }}
        onDragEnd={endDrag}
      >
        {board.columns.map((cc, i) => (
          <ColumnView
            key={cc.column.id}
            column={cc.column}
            cards={cc.cards}
            drag={drag}
            dropIndex={cardTarget?.columnId === cc.column.id ? cardTarget.index : null}
            columnDropBefore={draggingColumn && columnTarget === i}
            columnDropAfter={draggingColumn && columnTarget === board.columns.length && i === board.columns.length - 1}
            onDragStart={setDrag}
            onDragEnd={endDrag}
            onCardDragOver={(index) => setCardTarget({ columnId: cc.column.id, index })}
            onDropCard={(index) => {
              dropCard(drag.id, cc.column.id, index)
              endDrag()
            }}
            onRenameColumn={(name) => mutate(() => api.updateColumn(cc.column.id, { name: name.trim() }))}
            onToggleDone={() => mutate(() => api.updateColumn(cc.column.id, { is_done: !cc.column.is_done }))}
            onDeleteColumn={() => {
              if (confirm('Delete this column?')) mutate(() => api.deleteColumn(cc.column.id))
            }}
            onAddCard={() => setModal({ mode: 'create', projectId, columnId: cc.column.id })}
            onOpenCard={(card) => setModal({ mode: 'edit', projectId, cardId: card.id })}
          />
        ))}
      </div>
      {modal && (
        <CardModal
          initial={modal}
          columns={board.columns.map((c) => c.column)}
          cards={board.columns.flatMap((cc) => cc.cards)}
          onClose={() => setModal(null)}
          onChanged={load}
          onOpenCard={(cardId) => setModal({ mode: 'edit', projectId, cardId })}
        />
      )}
    </div>
  )
}
