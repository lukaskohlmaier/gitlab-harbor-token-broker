import { BrowserRouter as Router, Routes, Route, Link, useLocation } from "react-router-dom";
import { AccessLogs } from "./pages/AccessLogs";
import { Policies } from "./pages/Policies";
import { FileText, Shield } from "lucide-react";

function Navigation() {
  const location = useLocation();

  const isActive = (path: string) => location.pathname === path;

  return (
    <nav className="border-b bg-white">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          <div className="flex items-center space-x-8">
            <div className="text-xl font-bold">Harbor Token Broker</div>
            <div className="flex space-x-4">
              <Link
                to="/"
                className={`flex items-center space-x-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  isActive("/")
                    ? "bg-gray-100 text-gray-900"
                    : "text-gray-600 hover:bg-gray-50 hover:text-gray-900"
                }`}
              >
                <FileText className="h-4 w-4" />
                <span>Access Logs</span>
              </Link>
              <Link
                to="/policies"
                className={`flex items-center space-x-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  isActive("/policies")
                    ? "bg-gray-100 text-gray-900"
                    : "text-gray-600 hover:bg-gray-50 hover:text-gray-900"
                }`}
              >
                <Shield className="h-4 w-4" />
                <span>Policies</span>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </nav>
  );
}

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-50">
        <Navigation />
        <main className="container mx-auto px-4 py-8">
          <Routes>
            <Route path="/" element={<AccessLogs />} />
            <Route path="/policies" element={<Policies />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
