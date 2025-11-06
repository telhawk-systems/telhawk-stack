import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from './AuthProvider';

interface LayoutProps {
  children: ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const { user, logout } = useAuth();
  const location = useLocation();

  const handleLogout = async () => {
    try {
      await logout();
    } catch (err) {
      console.error('Logout failed', err);
    }
  };

  const isAdmin = user?.roles.includes('admin');

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">TelHawk SIEM</h1>
              <p className="text-sm text-gray-500 mt-1">OCSF-compliant Security Event Management</p>
            </div>
            <div className="flex items-center space-x-4">
              <nav className="flex space-x-4">
                <Link
                  to="/"
                  className={`px-3 py-2 rounded-md text-sm font-medium ${
                    location.pathname === '/'
                      ? 'bg-blue-100 text-blue-700'
                      : 'text-gray-600 hover:bg-gray-100'
                  }`}
                >
                  Search
                </Link>
                {isAdmin && (
                  <Link
                    to="/users"
                    className={`px-3 py-2 rounded-md text-sm font-medium ${
                      location.pathname === '/users'
                        ? 'bg-blue-100 text-blue-700'
                        : 'text-gray-600 hover:bg-gray-100'
                    }`}
                  >
                    Users
                  </Link>
                )}
              </nav>
              <div className="text-right">
                <p className="text-sm font-medium text-gray-900">{user?.user_id}</p>
                <p className="text-xs text-gray-500">
                  Roles: {user?.roles.join(', ')}
                </p>
              </div>
              <button
                onClick={handleLogout}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium transition-colors"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  );
}
