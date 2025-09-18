import { Route, Routes } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Plan from './pages/Plan'
import Rest from './pages/Rest'
import Record from './pages/Record'
import Manage from './pages/Manage'
import Timer from './pages/Timer'
import Header from './components/Header'

export default function App() {
  return (
    <>
      <Header />
      <div className="container" style={{ paddingTop: 18 }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/plan" element={<Plan />} />
          <Route path="/rest" element={<Rest />} />
          <Route path="/record" element={<Record />} />
          <Route path="/manage" element={<Manage />} />
          <Route path="/timer" element={<Timer />} />
        </Routes>
      </div>
    </>
  )
}
