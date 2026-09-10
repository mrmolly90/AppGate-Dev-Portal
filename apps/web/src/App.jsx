import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout.jsx'
import Register from './pages/Register.jsx'
import Dashboard from './pages/Dashboard.jsx'
import ApiKeys from './pages/ApiKeys.jsx'
import Policies from './pages/Policies.jsx'

/**
 * App — root component with route definitions.
 * Register is the landing page; Dashboard is the management view.
 * All routes are wrapped in the Layout (Android frame + bottom nav).
 */
export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Register />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/api-keys" element={<ApiKeys />} />
        <Route path="/policies" element={<Policies />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}