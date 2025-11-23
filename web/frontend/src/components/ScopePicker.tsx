import { useState, useRef, useEffect } from 'react';
import { useScope } from './ScopeProvider';
import { Organization, Client } from '../types';

// Icons
const PlatformIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const OrgIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
  </svg>
);

const ClientIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
  </svg>
);

const ChevronIcon = ({ open }: { open: boolean }) => (
  <svg className={`w-4 h-4 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
  </svg>
);

const CheckIcon = () => (
  <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
);

export function ScopePicker() {
  const {
    scope,
    userScope,
    loading,
    setScopeToPlatform,
    setScopeToOrganization,
    setScopeToClient,
    canViewPlatform,
  } = useScope();

  const [isOpen, setIsOpen] = useState(false);
  const [expandedOrg, setExpandedOrg] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (loading || !userScope) {
    return (
      <div className="px-4 py-3 border-b border-sidebar-hover">
        <div className="animate-pulse flex items-center gap-2">
          <div className="w-4 h-4 bg-sidebar-hover rounded"></div>
          <div className="h-4 bg-sidebar-hover rounded w-24"></div>
        </div>
      </div>
    );
  }

  // Get display label for current scope
  const getScopeLabel = () => {
    switch (scope.type) {
      case 'platform':
        return 'All Organizations';
      case 'organization':
        return scope.organization_name || 'Organization';
      case 'client':
        return scope.client_name || 'Client';
    }
  };

  const getScopeIcon = () => {
    switch (scope.type) {
      case 'platform':
        return <PlatformIcon />;
      case 'organization':
        return <OrgIcon />;
      case 'client':
        return <ClientIcon />;
    }
  };

  // Get clients for a specific organization
  const getClientsForOrg = (orgId: string): Client[] => {
    return userScope.clients.filter(c => c.organization_id === orgId);
  };

  const handleSelectPlatform = () => {
    setScopeToPlatform();
    setIsOpen(false);
  };

  const handleSelectOrganization = (org: Organization) => {
    setScopeToOrganization(org);
    setIsOpen(false);
  };

  const handleSelectClient = (client: Client) => {
    const org = userScope.organizations.find(o => o.id === client.organization_id);
    if (org) {
      setScopeToClient(client, org);
    }
    setIsOpen(false);
  };

  const toggleOrgExpand = (orgId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setExpandedOrg(expandedOrg === orgId ? null : orgId);
  };

  return (
    <div className="relative px-4 py-3 border-b border-sidebar-hover" ref={dropdownRef}>
      {/* Trigger button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between gap-2 px-3 py-2 text-sm text-sidebar-text hover:text-white hover:bg-sidebar-hover rounded-lg transition-colors"
      >
        <div className="flex items-center gap-2 min-w-0">
          {getScopeIcon()}
          <span className="truncate font-medium">{getScopeLabel()}</span>
        </div>
        <ChevronIcon open={isOpen} />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div className="absolute left-4 right-4 mt-1 bg-gray-800 border border-gray-700 rounded-lg shadow-xl z-50 max-h-80 overflow-y-auto">
          {/* Platform option */}
          {canViewPlatform() && (
            <button
              onClick={handleSelectPlatform}
              className={`w-full flex items-center justify-between px-3 py-2.5 text-sm hover:bg-gray-700 transition-colors ${
                scope.type === 'platform' ? 'text-white bg-gray-700' : 'text-gray-300'
              }`}
            >
              <div className="flex items-center gap-2">
                <PlatformIcon />
                <span>All Organizations</span>
              </div>
              {scope.type === 'platform' && <CheckIcon />}
            </button>
          )}

          {/* Divider */}
          {canViewPlatform() && userScope.organizations.length > 0 && (
            <div className="border-t border-gray-700 my-1"></div>
          )}

          {/* Organizations and their clients */}
          {userScope.organizations.map(org => {
            const clients = getClientsForOrg(org.id);
            const isExpanded = expandedOrg === org.id;
            const isOrgSelected = scope.type === 'organization' && scope.organization_id === org.id;

            return (
              <div key={org.id}>
                <div className="flex items-center">
                  {/* Expand button if org has clients */}
                  {clients.length > 0 && (
                    <button
                      onClick={(e) => toggleOrgExpand(org.id, e)}
                      className="p-2 text-gray-400 hover:text-white"
                    >
                      <svg className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-90' : ''}`} fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                      </svg>
                    </button>
                  )}

                  {/* Organization button */}
                  <button
                    onClick={() => handleSelectOrganization(org)}
                    className={`flex-1 flex items-center justify-between px-3 py-2.5 text-sm hover:bg-gray-700 transition-colors ${
                      isOrgSelected ? 'text-white bg-gray-700' : 'text-gray-300'
                    } ${clients.length === 0 ? 'pl-8' : ''}`}
                  >
                    <div className="flex items-center gap-2">
                      <OrgIcon />
                      <span>{org.name}</span>
                    </div>
                    {isOrgSelected && <CheckIcon />}
                  </button>
                </div>

                {/* Clients under this org */}
                {isExpanded && clients.map(client => {
                  const isClientSelected = scope.type === 'client' && scope.client_id === client.id;
                  return (
                    <button
                      key={client.id}
                      onClick={() => handleSelectClient(client)}
                      className={`w-full flex items-center justify-between pl-12 pr-3 py-2 text-sm hover:bg-gray-700 transition-colors ${
                        isClientSelected ? 'text-white bg-gray-700' : 'text-gray-400'
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <ClientIcon />
                        <span>{client.name}</span>
                      </div>
                      {isClientSelected && <CheckIcon />}
                    </button>
                  );
                })}
              </div>
            );
          })}

          {/* Direct client list for non-platform users */}
          {userScope.max_tier === 'client' && userScope.clients.map(client => {
            const isClientSelected = scope.type === 'client' && scope.client_id === client.id;
            return (
              <button
                key={client.id}
                onClick={() => handleSelectClient(client)}
                className={`w-full flex items-center justify-between px-3 py-2.5 text-sm hover:bg-gray-700 transition-colors ${
                  isClientSelected ? 'text-white bg-gray-700' : 'text-gray-300'
                }`}
              >
                <div className="flex items-center gap-2">
                  <ClientIcon />
                  <span>{client.name}</span>
                </div>
                {isClientSelected && <CheckIcon />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
