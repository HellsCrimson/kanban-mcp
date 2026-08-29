import { useCallback, useEffect, useState } from 'react'
import { api } from './api.js'
import ProjectList from './components/ProjectList.jsx'
import Board from './components/Board.jsx'

export default function App() {
  const [projects, setProjects] = useState(null)
  const [selectedId, setSelectedId] = useState(null)
  const [error, setError] = useState('')

  const loadProjects = useCallback(async () => {
    try {
      setProjects(await api.projects())
      setError('')
    } catch (e) {
      setError(e.message)
    }
  }, [])

  useEffect(() => {
    loadProjects()
  }, [loadProjects])

  if (selectedId) {
    return <Board key={selectedId} projectId={selectedId} onBack={() => setSelectedId(null)} />
  }

  return <ProjectList projects={projects} error={error} onSelect={setSelectedId} onChanged={loadProjects} />
}
