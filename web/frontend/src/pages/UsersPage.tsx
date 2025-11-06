import { useState, useEffect } from 'react';
import { Layout } from '../components/Layout';
import { apiClient } from '../services/api';
import { UserDetails } from '../types';

export function UsersPage() {
  const [users, setUsers] = useState<UserDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingUser, setEditingUser] = useState<UserDetails | null>(null);
  const [showResetPassword, setShowResetPassword] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const data = await apiClient.listUsers();
      setUsers(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleEnabled = async (user: UserDetails) => {
    try {
      await apiClient.updateUser(user.id, { enabled: !user.enabled });
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update user');
    }
  };

  const handleDeleteUser = async (userId: string) => {
    if (!confirm('Are you sure you want to delete this user?')) return;
    
    try {
      await apiClient.deleteUser(userId);
      await loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user');
    }
  };

  const handleResetPassword = async (userId: string) => {
    if (!newPassword || newPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    try {
      await apiClient.resetPassword(userId, newPassword);
      setShowResetPassword(null);
      setNewPassword('');
      setError(null);
      alert('Password reset successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reset password');
    }
  };

  const handleUpdateRoles = async (user: UserDetails, newRoles: string[]) => {
    try {
      await apiClient.updateUser(user.id, { roles: newRoles });
      await loadUsers();
      setEditingUser(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update roles');
    }
  };

  const roleOptions = ['admin', 'analyst', 'viewer', 'ingester'];

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">Loading users...</div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">User Management</h1>
        <p className="text-gray-600 mt-1">Manage system users and their permissions</p>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800">{error}</p>
        </div>
      )}

      <div className="bg-white shadow-md rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Username
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Email
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Roles
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {users.map((user) => (
              <tr key={user.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="text-sm font-medium text-gray-900">{user.username}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="text-sm text-gray-600">{user.email}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  {editingUser?.id === user.id ? (
                    <div className="flex gap-2 flex-wrap">
                      {roleOptions.map((role) => (
                        <label key={role} className="flex items-center text-sm">
                          <input
                            type="checkbox"
                            checked={editingUser.roles.includes(role)}
                            onChange={(e) => {
                              const newRoles = e.target.checked
                                ? [...editingUser.roles, role]
                                : editingUser.roles.filter((r) => r !== role);
                              setEditingUser({ ...editingUser, roles: newRoles });
                            }}
                            className="mr-1"
                          />
                          {role}
                        </label>
                      ))}
                      <button
                        onClick={() => handleUpdateRoles(user, editingUser.roles)}
                        className="ml-2 px-2 py-1 bg-blue-600 text-white text-xs rounded hover:bg-blue-700"
                      >
                        Save
                      </button>
                      <button
                        onClick={() => setEditingUser(null)}
                        className="px-2 py-1 bg-gray-300 text-gray-700 text-xs rounded hover:bg-gray-400"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <div className="flex gap-1 flex-wrap">
                      {user.roles.map((role) => (
                        <span
                          key={role}
                          className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800"
                        >
                          {role}
                        </span>
                      ))}
                      <button
                        onClick={() => setEditingUser(user)}
                        className="ml-1 text-blue-600 hover:text-blue-800 text-xs"
                      >
                        Edit
                      </button>
                    </div>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <button
                    onClick={() => handleToggleEnabled(user)}
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      user.enabled
                        ? 'bg-green-100 text-green-800 hover:bg-green-200'
                        : 'bg-red-100 text-red-800 hover:bg-red-200'
                    }`}
                  >
                    {user.enabled ? 'Enabled' : 'Disabled'}
                  </button>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                  {new Date(user.created_at).toLocaleDateString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                  {showResetPassword === user.id ? (
                    <div className="flex gap-2 justify-end">
                      <input
                        type="password"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        placeholder="New password"
                        className="px-2 py-1 border rounded text-sm"
                        minLength={8}
                      />
                      <button
                        onClick={() => handleResetPassword(user.id)}
                        className="px-2 py-1 bg-blue-600 text-white text-xs rounded hover:bg-blue-700"
                      >
                        Confirm
                      </button>
                      <button
                        onClick={() => {
                          setShowResetPassword(null);
                          setNewPassword('');
                        }}
                        className="px-2 py-1 bg-gray-300 text-gray-700 text-xs rounded hover:bg-gray-400"
                      >
                        Cancel
                      </button>
                    </div>
                  ) : (
                    <div className="flex gap-2 justify-end">
                      <button
                        onClick={() => setShowResetPassword(user.id)}
                        className="text-blue-600 hover:text-blue-800 font-medium"
                      >
                        Reset Password
                      </button>
                      <button
                        onClick={() => handleDeleteUser(user.id)}
                        className="text-red-600 hover:text-red-800 font-medium"
                      >
                        Delete
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {users.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No users found
          </div>
        )}
      </div>
    </Layout>
  );
}
