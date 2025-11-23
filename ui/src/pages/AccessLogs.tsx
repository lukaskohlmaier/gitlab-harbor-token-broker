import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { AccessLog } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Input } from "../components/Input";
import { Button } from "../components/Button";

export function AccessLogs() {
  const [logs, setLogs] = useState<AccessLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState({
    gitlab_project: "",
    harbor_project: "",
    status: "",
  });

  const limit = 20;

  useEffect(() => {
    loadLogs();
  }, [page, filters]);

  const loadLogs = async () => {
    try {
      setLoading(true);
      const response = await api.getAccessLogs({
        page,
        limit,
        ...filters,
      });
      setLogs(response.logs || []);
      setTotal(response.total);
    } catch (error) {
      console.error("Failed to load access logs:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
    setPage(1);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Access Logs</h1>
        <p className="text-gray-500">View token request history and audit trail</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <label className="text-sm font-medium">GitLab Project</label>
              <Input
                placeholder="e.g., mygroup/myproject"
                value={filters.gitlab_project}
                onChange={(e) => handleFilterChange("gitlab_project", e.target.value)}
              />
            </div>
            <div>
              <label className="text-sm font-medium">Harbor Project</label>
              <Input
                placeholder="e.g., backend-project"
                value={filters.harbor_project}
                onChange={(e) => handleFilterChange("harbor_project", e.target.value)}
              />
            </div>
            <div>
              <label className="text-sm font-medium">Status</label>
              <select
                className="flex h-10 w-full rounded-md border border-gray-200 bg-white px-3 py-2 text-sm"
                value={filters.status}
                onChange={(e) => handleFilterChange("status", e.target.value)}
              >
                <option value="">All</option>
                <option value="success">Success</option>
                <option value="denied">Denied</option>
              </select>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 text-center">Loading...</div>
          ) : logs.length === 0 ? (
            <div className="p-6 text-center text-gray-500">No logs found</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="border-b bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium">Timestamp</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">GitLab Project</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Harbor Project</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Permission</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Status</th>
                    <th className="px-4 py-3 text-left text-sm font-medium">Robot Name</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm">{formatDate(log.timestamp)}</td>
                      <td className="px-4 py-3 text-sm">{log.gitlab_project}</td>
                      <td className="px-4 py-3 text-sm">{log.harbor_project}</td>
                      <td className="px-4 py-3 text-sm">
                        <span className="rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
                          {log.permission}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <span
                          className={`rounded-full px-2 py-1 text-xs font-medium ${
                            log.status === "success"
                              ? "bg-green-100 text-green-800"
                              : "bg-red-100 text-red-800"
                          }`}
                        >
                          {log.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm font-mono text-xs">
                        {log.robot_name || "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-gray-500">
            Showing {(page - 1) * limit + 1} to {Math.min(page * limit, total)} of {total} logs
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page - 1)}
              disabled={page === 1}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(page + 1)}
              disabled={page >= totalPages}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
