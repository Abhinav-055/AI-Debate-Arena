import { useEffect } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { PageShell } from "./components/layout/PageShell";
import Home from "./pages/Home";
import DebateRoom from "./pages/DebateRoom";
import DebateHistory from "./pages/DebateHistory";

export default function App() {
  useEffect(() => {
    const onPopState = () => {
      window.location.reload();
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  return (
    <PageShell>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/debate/:id" element={<DebateRoom />} />
        <Route path="/history" element={<DebateHistory />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </PageShell>
  );
}
