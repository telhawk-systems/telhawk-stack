import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '../AuthProvider';
import { useScope } from '../ScopeProvider';
import { ScopePicker } from '../ScopePicker';
import logo from '../../assets/logo.svg';

interface NavItem {
  path: string;
  label: string;
  icon: React.ReactNode;
  adminOnly?: boolean;
  requiresClient?: boolean;  // Only show when a client is selected
}

// Icon components (inline SVGs for simplicity)
const DashboardIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
  </svg>
);

const EventsIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
  </svg>
);

const AlertsIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

const RulesIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
);

const SavedSearchIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
  </svg>
);

const UsersIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
  </svg>
);

const TokensIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
  </svg>
);

const LogoutIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
  </svg>
);

const primaryNavItems: NavItem[] = [
  { path: '/', label: 'Dashboard', icon: <DashboardIcon /> },
  { path: '/events', label: 'Events', icon: <EventsIcon />, requiresClient: true },
  { path: '/alerts', label: 'Alerts', icon: <AlertsIcon />, requiresClient: true },
  { path: '/rules', label: 'Rules', icon: <RulesIcon />, requiresClient: true },
  { path: '/saved-searches', label: 'Saved Searches', icon: <SavedSearchIcon />, requiresClient: true },
];

const adminNavItems: NavItem[] = [
  { path: '/users', label: 'Users', icon: <UsersIcon />, adminOnly: true },
  { path: '/tokens', label: 'Tokens', icon: <TokensIcon />, adminOnly: true },
];

export function Sidebar() {
  const { user, logout } = useAuth();
  const { hasClientSelected } = useScope();
  const location = useLocation();
  const isAdmin = user?.roles.includes('admin');
  const clientSelected = hasClientSelected();

  const handleLogout = async () => {
    try {
      await logout();
    } catch (err) {
      console.error('Logout failed', err);
    }
  };

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  // Filter nav items based on whether they require a client to be selected
  const visibleNavItems = primaryNavItems.filter(item => {
    if (item.requiresClient && !clientSelected) {
      return false;
    }
    return true;
  });

  const NavLink = ({ item }: { item: NavItem }) => (
    <Link
      to={item.path}
      className={`flex items-center gap-3 px-4 py-2.5 text-sm font-medium transition-colors ${
        isActive(item.path)
          ? 'bg-sidebar-hover text-sidebar-active border-l-3 border-sidebar-accent'
          : 'text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-active'
      }`}
    >
      {item.icon}
      <span>{item.label}</span>
    </Link>
  );

  return (
    <aside className="fixed left-0 top-0 h-screen w-sidebar bg-sidebar flex flex-col z-50">
      {/* Logo/Brand */}
      <div className="flex items-center gap-3 px-4 py-5 border-b border-sidebar-hover">
        <img src={logo} alt="TelHawk" className="h-10 w-10" />
        <div>
          <h1 className="text-lg font-bold text-white tracking-tight">TelHawk</h1>
          <p className="text-xs text-sidebar-text">Security Intelligence</p>
        </div>
      </div>

      {/* Scope Picker */}
      <ScopePicker />

      {/* Primary Navigation */}
      <nav className="flex-1 py-4 overflow-y-auto sidebar-scrollbar">
        <div className="space-y-1">
          {visibleNavItems.map((item) => (
            <NavLink key={item.path} item={item} />
          ))}
        </div>

        {/* Client selection hint when items are hidden */}
        {!clientSelected && (
          <div className="mt-4 mx-4 p-3 bg-sidebar-hover/50 rounded-lg border border-sidebar-hover">
            <p className="text-xs text-gray-400">
              Select a client to view Events, Alerts, Rules, and Saved Searches.
            </p>
          </div>
        )}

        {/* Admin Section */}
        {isAdmin && (
          <div className="mt-6">
            <div className="px-4 py-2">
              <span className="text-xs font-semibold text-sidebar-text uppercase tracking-wider">
                Administration
              </span>
            </div>
            <div className="space-y-1">
              {adminNavItems.map((item) => (
                <NavLink key={item.path} item={item} />
              ))}
            </div>
          </div>
        )}
      </nav>

      {/* User Section */}
      <div className="border-t border-sidebar-hover p-4">
        <div className="flex items-center gap-3 mb-3">
          <div className="w-9 h-9 rounded-full bg-sidebar-accent flex items-center justify-center text-white font-semibold text-sm">
            {user?.user_id?.charAt(0).toUpperCase() || 'U'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-white truncate">{user?.user_id}</p>
            <p className="text-xs text-sidebar-text truncate">{user?.roles.join(', ')}</p>
          </div>
        </div>
        <button
          onClick={handleLogout}
          className="flex items-center gap-2 w-full px-3 py-2 text-sm text-sidebar-text hover:text-white hover:bg-sidebar-hover rounded-lg transition-colors"
        >
          <LogoutIcon />
          <span>Logout</span>
        </button>
      </div>
    </aside>
  );
}
