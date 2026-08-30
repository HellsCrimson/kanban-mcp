async function parse(r) {
  if (!r.ok) {
    let msg = r.statusText
    try {
      const e = await r.json()
      if (e.error) msg = e.error
    } catch {}
    throw new Error(msg)
  }
  if (r.status === 204) return null
  return r.json()
}

const json = (method, url, body) =>
  fetch(url, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  }).then(parse)

export const api = {
  projects: () => json('GET', '/api/projects'),
  createProject: (name, description) => json('POST', '/api/projects', { name, description }),
  deleteProject: (id) => json('DELETE', `/api/projects/${id}`),
  board: (id) => json('GET', `/api/projects/${id}/board`),
  addColumn: (projectId, name) => json('POST', `/api/projects/${projectId}/columns`, { name }),
  updateColumn: (columnId, patch) => json('PATCH', `/api/columns/${columnId}`, patch),
  deleteColumn: (columnId) => json('DELETE', `/api/columns/${columnId}`),
  createCard: (projectId, data) => json('POST', `/api/projects/${projectId}/cards`, data),
  card: (id) => json('GET', `/api/cards/${id}`),
  updateCard: (id, patch) => json('PATCH', `/api/cards/${id}`, patch),
  moveCard: (id, columnId, position) => json('POST', `/api/cards/${id}/move`, { column_id: columnId, position }),
  deleteCard: (id) => json('DELETE', `/api/cards/${id}`),
  addUpdate: (cardId, content) => json('POST', `/api/cards/${cardId}/updates`, { content }),
  addRelationship: (data) => json('POST', '/api/relationships', data),
  deleteRelationship: (id) => json('DELETE', `/api/relationships/${id}`),
}
