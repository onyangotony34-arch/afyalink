import { BrowserRouter, Routes, Route } from "react-router-dom"
import Landing from "./Landing"
import Login from "./login"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing />} />
        <Route path="/login" element={<Login />} />
      </Routes>
    </BrowserRouter>
  )
}