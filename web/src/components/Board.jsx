import { useCallback, useEffect, useState } from 'react'
import { api } from '../api.js'
import ColumnView from './ColumnView.jsx'
import CardModal from './CardModal.jsx'

export default function Board({ projectId, onBack }) {
  const [board, setBoard] = useState(null)
  const [error, setError] = useState('')
  const [modal, setModal] = useState(null)
  const [draggingId, setDraggingId] = useState(null)

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

  async function mutate(fn) {
    try {
      await fn()
      await load()
    } catch (e) {
      alert(e.message)
    }
  }

  function moveCard(draggedId, toColumnId, beforeCardId) {
    if (!board || draggedId === beforeCardId) return
    const col = board.columns.find((c) => c.column.id === toColumnId)
    if (!col) return
    const ids = col.cards.map((c) => c.id)
    const fromIdx = ids.indexOf(draggedId)
    if (fromIdx !== -1) ids.splice(fromIdx, 1)
    let position = null
    if (beforeCardId) {
      position = ids.indexOf(beforeCardId)
      if (position === -1) position = null
    }
    mutate(() => api.moveCard(draggedId, toColumnId, position))
  }

  function addColumn() {
    const name = prompt('Column name:')
    if (!name || !name.trim()) return
    mutate(() => api.addColumn(projectId, name.trim()))
  }

  if (error) return <div className="banner error center">{error}</div>
  if (!board) return <div className="loading">Loading…</div>

  return (
    <div className="board-page">
      <div className="topbar">
        <button className="ghost" onClick={onBack}>← Projects</button>
        <span className="topbar-title">{board.project.name}</span>
        {board.project.description && <span className="topbar-desc">{board.project.description}</span>}
        <span className="spacer" />
        <button onClick={addColumn}>+ Column</button>
      </div>
      <div className="board">
        {board.columns.map((cc) => (
          <ColumnView
            key={cc.column.id}
            column={cc.column}
            cards={cc.cards}
            draggingId={draggingId}
            onDragStart={setDraggingId}
            onDragEnd={() => setDraggingId(null)}
            onDropCard={(columnId, beforeCardId) => moveCard(draggingId, columnId, beforeCardId)}
            onRenameColumn={(columnId, name) => mutate(() => api.renameColumn(columnId, name.trim()))}
            onDeleteColumn={(columnId) => {
              if (confirm('Delete this column?')) mutate(() => api.deleteColumn(columnId))
            }}
            onAddCard={(columnId, cardType) => setModal({ mode: 'create', projectId, columnId, cardType })}
            onOpenCard={(card) => setModal({ mode: 'edit', projectId, cardId: card.id })}
          />
        ))}
      </div>
      {modal && (
        <CardModal
          initial={modal}
          columns={board.columns.map((c) => c.column)}
          onClose={() => setModal(null)}
          onChanged={() => {
            load()
          }}
          onOpenCard={(cardId) => setModal({ mode: 'edit', projectId, cardId })}
        />
      )}
    </div>
  )
}
