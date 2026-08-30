import { Fragment, useRef, useState } from 'react'
import CardView from './CardView.jsx'

export default function ColumnView({
  column,
  cards,
  drag,
  dropIndex,
  columnDropBefore,
  columnDropAfter,
  onDragStart,
  onDragEnd,
  onCardDragOver,
  onDropCard,
  onRenameColumn,
  onToggleDone,
  onDeleteColumn,
  onAddCard,
  onOpenCard,
}) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(column.name)
  const listRef = useRef(null)

  const draggingCard = drag?.kind === 'card'

  function saveName() {
    setEditing(false)
    if (name.trim() && name.trim() !== column.name) {
      onRenameColumn(name)
    } else {
      setName(column.name)
    }
  }

  // The slot the card would drop into: the first card whose vertical midpoint
  // is below the pointer. Dropping on a card's top half inserts before it, on
  // its bottom half after it — no need to aim at the gaps.
  function cardIndexAt(clientY) {
    const el = listRef.current
    if (!el) return cards.length
    const items = [...el.querySelectorAll('[data-card-id]')]
    for (let i = 0; i < items.length; i++) {
      const r = items[i].getBoundingClientRect()
      if (clientY < r.top + r.height / 2) return i
    }
    return items.length
  }

  return (
    <div
      className={
        'column' +
        (column.is_done ? ' done-column' : '') +
        (draggingCard && dropIndex !== null ? ' over' : '') +
        (drag?.kind === 'column' && drag.id === column.id ? ' dragging' : '') +
        (columnDropBefore ? ' col-drop-before' : '') +
        (columnDropAfter ? ' col-drop-after' : '')
      }
      data-column-id={column.id}
      onDragOver={(e) => {
        if (!draggingCard) return
        e.preventDefault()
        e.stopPropagation()
        onCardDragOver(cardIndexAt(e.clientY))
      }}
      onDrop={(e) => {
        if (!draggingCard) return
        e.preventDefault()
        e.stopPropagation()
        onDropCard(cardIndexAt(e.clientY))
      }}
    >
      <div
        className="column-header"
        draggable={!editing}
        onDragStart={(e) => {
          e.dataTransfer.effectAllowed = 'move'
          e.dataTransfer.setData('text/plain', column.id)
          onDragStart({ kind: 'column', id: column.id })
        }}
        onDragEnd={onDragEnd}
      >
        <span className="drag-handle" title="Drag to reorder the column">⠿</span>
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
          <span
            className="column-name"
            title="Click to rename"
            onClick={() => {
              // Start from the current name: another client may have renamed
              // the column since this component mounted.
              setName(column.name)
              setEditing(true)
            }}
          >
            {column.name}
          </span>
        )}
        <span className="column-count">{cards.length}</span>
        <button
          className={'icon-btn done-toggle' + (column.is_done ? ' active' : '')}
          title={column.is_done ? 'Cards here count as done — click to unset' : 'Count cards here as done'}
          onClick={onToggleDone}
        >
          ✓
        </button>
        {cards.length === 0 && !editing && (
          <button className="icon-btn" title="Delete column" onClick={onDeleteColumn}>
            ×
          </button>
        )}
      </div>
      <div className="column-cards" ref={listRef}>
        {cards.map((c, i) => (
          <Fragment key={c.id}>
            {draggingCard && dropIndex === i && <div className="drop-indicator" />}
            <CardView
              card={c}
              dragging={drag?.kind === 'card' && drag.id === c.id}
              onOpen={() => onOpenCard(c)}
              onDragStart={() => onDragStart({ kind: 'card', id: c.id })}
              onDragEnd={onDragEnd}
            />
          </Fragment>
        ))}
        {draggingCard && dropIndex === cards.length && <div className="drop-indicator" />}
        <div className="column-dropzone" />
      </div>
      <div className="column-footer">
        <button onClick={onAddCard}>+ Task</button>
      </div>
    </div>
  )
}
