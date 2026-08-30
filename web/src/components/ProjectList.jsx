import { useState } from 'react'
import { api } from '../api.js'
import { useLiveUpdates } from '../useLiveUpdates.js'

export default function ProjectList({ projects, error, onSelect, onChanged }) {
  useLiveUpdates(null, onChanged)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)

  async function create(e) {
    e.preventDefault()
    if (!name.trim()) return
    setBusy(true)
    try {
      await api.createProject(name.trim(), description.trim())
      setName('')
      setDescription('')
      onChanged()
    } catch (err) {
      alert(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function remove(id) {
    if (!confirm('Delete this project and all of its columns and cards?')) return
    try {
      await api.deleteProject(id)
      onChanged()
    } catch (err) {
      alert(err.message)
    }
  }

  return (
    <div className="projects">
      <h1>Projects</h1>
      {error && <div className="banner error">{error}</div>}
      <form className="new-project" onSubmit={create}>
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Project name" autoFocus />
        <input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Description (optional)" />
        <button disabled={busy || !name.trim()}>Create project</button>
      </form>
      <div className="project-grid">
        {(projects ?? []).map((p) => (
          <div key={p.id} className="project-card" onClick={() => onSelect(p.id)}>
            <div className="project-name">{p.name}</div>
            {p.description && <div className="project-desc">{p.description}</div>}
            <button
              className="danger small"
              onClick={(e) => {
                e.stopPropagation()
                remove(p.id)
              }}
            >
              Delete
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
