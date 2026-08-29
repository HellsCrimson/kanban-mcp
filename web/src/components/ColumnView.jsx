import { useState } from 'react'
import CardView from './CardView.jsx'

export default function ColumnView({ column, cards, draggingId, onDragStart, onDragEnd, onDropCard, onRenameColumn, onDeleteColumn, onAddCard, onOpenCard }) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(column.name)
  const [over, setOver] = useState(false)

  function saveName() {
    setEditing(false)
    if (name.trim() && name.trim() !== column.name) {
      onRenameColumn(column.id, name)
    } else {
      setName(column.name)
    }
  }

  return (
    <div
      className={'column' + (over ? ' over' : '')}
      onDragOver={(e) => {
        e.preventDefault()
        setOver(true)
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault()
        setOver(false)
        if (draggingId) onDropCard(column.id, null)
      }}
    >
      <div className="column-header">
        {editing ? (
          <input
            className="column-rename"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onBlur={saveName}
            onKeyDown={(e) => e.key === 'Enter' && saveName()}
            autoFocus
          />
        ) : (
          <span className="column-name" title="Click to rename" onClick={() => setEditing(true)}>
            {column.name}
          </span>
        )}
        <span className="column-count">{cards.length}</span>
        {cards.length === 0 && !editing && (
          <button className="icon-btn" title="Delete column" onClick={() => onDeleteColumn(column.id)}>
            ×
          </button>
        )}
      </div>
      <div className="column-cards">
        {cards.map((c) => (
          <CardView
            key={c.id}
            card={c}
            draggingId={draggingId}
            onOpen={() => onOpenCard(c)}
            onDropBefore={() => onDropCard(column.id, c.id)}
            onDragStart={onDragStart}
            onDragEnd={onDragEnd}
          />
        ))}
        <div className="column-dropzone" />
      </div>
      <div className="column-footer">
        <button onClick={() => onAddCard(column.id, 'task')}>+ Task</button>
        <button onClick={() => onAddCard(column.id, 'epic')}>+ Epic</button>
      </div>
    </div>
  )
}
