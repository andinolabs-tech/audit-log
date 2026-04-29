import { BrowserRouter, Route, Routes } from 'react-router-dom'
import QueryPage from './pages/QueryPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<QueryPage />} />
        <Route path="/events" element={<QueryPage />} />
      </Routes>
    </BrowserRouter>
  )
}
