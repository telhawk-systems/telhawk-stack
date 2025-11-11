import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from './AuthProvider';
import logo from '../assets/logo.svg';

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
      <header className="bg-gradient-to-r from-gray-900 via-gray-800 to-gray-900 shadow-lg border-b border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-3">
          <div className="flex justify-between items-center">
            {/* Logo and Brand */}
            <div className="flex items-center space-x-3">
              <img src={logo} alt="TelHawk" className="h-10 w-10" />
              <div>
                <h1 className="text-2xl font-bold text-white tracking-tight">TelHawk</h1>
                <p className="text-xs text-gray-400">Security Intelligence Platform</p>
              </div>
            </div>

            {/* Navigation and User Section */}
            <div className="flex items-center space-x-6">
              {/* Navigation */}
              <nav className="flex space-x-1">
                <Link
                  to="/"
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === '/'
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                  }`}
                >
                  Search
                </Link>
                <Link
                  to="/events"
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === '/events'
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                  }`}
                >
                  Events
                </Link>
                <Link
                  to="/saved-searches"
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === '/saved-searches'
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                  }`}
                >
                  Saved Searches
                </Link>
                <Link
                  to="/rules"
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === '/rules'
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                  }`}
                >
                  Rules
                </Link>
                <Link
                  to="/alerts"
                  className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                    location.pathname === '/alerts'
                      ? 'bg-blue-600 text-white shadow-md'
                      : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                  }`}
                >
                  Alerts
                </Link>
                {isAdmin && (
                  <>
                    <Link
                      to="/users"
                      className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                        location.pathname === '/users'
                          ? 'bg-blue-600 text-white shadow-md'
                          : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                      }`}
                    >
                      Users
                    </Link>
                    <Link
                      to="/tokens"
                      className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                        location.pathname === '/tokens'
                          ? 'bg-blue-600 text-white shadow-md'
                          : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                      }`}
                    >
                      Tokens
                    </Link>
                  </>
                )}
              </nav>

              {/* User Info */}
              <div className="flex items-center space-x-4 border-l border-gray-700 pl-6">
                <div className="text-right">
                  <p className="text-sm font-medium text-white">{user?.user_id}</p>
                  <p className="text-xs text-gray-400">
                    {user?.roles.join(', ')}
                  </p>
                </div>
                <button
                  onClick={handleLogout}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium transition-colors shadow-md hover:shadow-lg"
                >
                  Logout
                </button>
              </div>
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
