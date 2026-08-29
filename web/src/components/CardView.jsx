export default function CardView({ card, draggingId, onOpen, onDropBefore, onDragStart, onDragEnd }) {
  const isEpic = card.type === 'epic'
  return (
    <div
      className={'card' + (isEpic ? ' epic' : '') + (draggingId === card.id ? ' dragging' : '')}
      draggable
      onDragStart={(e) => {
        e.dataTransfer.effectAllowed = 'move'
        e.dataTransfer.setData('text/plain', card.id)
        onDragStart(card.id)
      }}
      onDragEnd={onDragEnd}
      onDragOver={(e) => {
        e.preventDefault()
        e.stopPropagation()
      }}
      onDrop={(e) => {
        e.preventDefault()
        e.stopPropagation()
        if (draggingId && card.id !== draggingId) onDropBefore()
      }}
      onClick={onOpen}
    >
      <div className="card-top">
        <span className="card-number">#{card.number}</span>
        {isEpic && (
          <span className="badge epic-badge">
            EPIC{card.sub_total > 0 ? ` ${card.sub_done}/${card.sub_total}` : ''}
          </span>
        )}
      </div>
      <div className="card-title">{card.title}</div>
      <div className="card-bottom">
        <span className={'pill priority-' + card.priority}>{card.priority}</span>
        <span className="chip" title="Complexity">C{card.complexity}</span>
      </div>
    </div>
  )
}
