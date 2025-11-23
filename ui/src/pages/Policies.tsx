import { useEffect, useState } from "react";
import { api } from "../api/client";
import type { PolicyRule } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/Card";
import { Button } from "../components/Button";
import { Input } from "../components/Input";
import { Plus, Trash2, Edit } from "lucide-react";

export function Policies() {
  const [policies, setPolicies] = useState<PolicyRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingPolicy, setEditingPolicy] = useState<PolicyRule | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    gitlab_project: "",
    harbor_projects: "",
    allowed_permissions: [] as string[],
  });

  useEffect(() => {
    loadPolicies();
  }, []);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const data = await api.getPolicies();
      setPolicies(data || []);
    } catch (error) {
      console.error("Failed to load policies:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingPolicy(null);
    setFormData({
      gitlab_project: "",
      harbor_projects: "",
      allowed_permissions: [],
    });
    setShowForm(true);
  };

  const handleEdit = (policy: PolicyRule) => {
    setEditingPolicy(policy);
    setFormData({
      gitlab_project: policy.gitlab_project,
      harbor_projects: policy.harbor_projects.join(", "),
      allowed_permissions: policy.allowed_permissions,
    });
    setShowForm(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm("Are you sure you want to delete this policy?")) return;
    
    try {
      await api.deletePolicy(id);
      await loadPolicies();
    } catch (error) {
      console.error("Failed to delete policy:", error);
      alert("Failed to delete policy");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const policy = {
      gitlab_project: formData.gitlab_project,
      harbor_projects: formData.harbor_projects
        .split(",")
        .map((p) => p.trim())
        .filter(Boolean),
      allowed_permissions: formData.allowed_permissions,
    };

    try {
      if (editingPolicy) {
        await api.updatePolicy(editingPolicy.id, policy);
      } else {
        await api.createPolicy(policy);
      }
      setShowForm(false);
      await loadPolicies();
    } catch (error) {
      console.error("Failed to save policy:", error);
      alert("Failed to save policy");
    }
  };

  const togglePermission = (perm: string) => {
    setFormData((prev) => ({
      ...prev,
      allowed_permissions: prev.allowed_permissions.includes(perm)
        ? prev.allowed_permissions.filter((p) => p !== perm)
        : [...prev.allowed_permissions, perm],
    }));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Policy Rules</h1>
          <p className="text-gray-500">
            Manage authorization policies for GitLab projects
          </p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          Add Policy
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>{editingPolicy ? "Edit Policy" : "New Policy"}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="text-sm font-medium">GitLab Project</label>
                <Input
                  placeholder="e.g., mygroup/myproject"
                  value={formData.gitlab_project}
                  onChange={(e) =>
                    setFormData({ ...formData, gitlab_project: e.target.value })
                  }
                  required
                />
              </div>

              <div>
                <label className="text-sm font-medium">
                  Harbor Projects (comma-separated)
                </label>
                <Input
                  placeholder="e.g., project1, project2"
                  value={formData.harbor_projects}
                  onChange={(e) =>
                    setFormData({ ...formData, harbor_projects: e.target.value })
                  }
                  required
                />
              </div>

              <div>
                <label className="text-sm font-medium">Allowed Permissions</label>
                <div className="mt-2 space-y-2">
                  {["read", "write", "read-write"].map((perm) => (
                    <label key={perm} className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        checked={formData.allowed_permissions.includes(perm)}
                        onChange={() => togglePermission(perm)}
                        className="h-4 w-4 rounded border-gray-300"
                      />
                      <span className="text-sm">{perm}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex gap-2">
                <Button type="submit">Save</Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setShowForm(false)}
                >
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 text-center">Loading...</div>
          ) : policies.length === 0 ? (
            <div className="p-6 text-center text-gray-500">
              No policies configured. Add one to get started.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="border-b bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      GitLab Project
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Harbor Projects
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Allowed Permissions
                    </th>
                    <th className="px-4 py-3 text-left text-sm font-medium">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {policies.map((policy) => (
                    <tr key={policy.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm font-medium">
                        {policy.gitlab_project}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        {policy.harbor_projects.join(", ")}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <div className="flex gap-1">
                          {policy.allowed_permissions.map((perm) => (
                            <span
                              key={perm}
                              className="rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800"
                            >
                              {perm}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <div className="flex gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEdit(policy)}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(policy.id)}
                          >
                            <Trash2 className="h-4 w-4 text-red-500" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
