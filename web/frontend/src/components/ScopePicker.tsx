import { useState, useRef, useEffect } from 'react';
import { useScope } from './ScopeProvider';
import { Organization, Client } from '../types';

// Icons
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

const NoneIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
  </svg>
);

interface DropdownProps {
  label: string;
  value: string;
  icon: React.ReactNode;
  isOpen: boolean;
  onToggle: () => void;
  onClose: () => void;
  children: React.ReactNode;
  disabled?: boolean;
}

function Dropdown({ label, value, icon, isOpen, onToggle, onClose, children, disabled }: DropdownProps) {
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  return (
    <div className="relative" ref={dropdownRef}>
      <label className="block text-xs text-gray-400 mb-1 px-1">{label}</label>
      <button
        onClick={onToggle}
        disabled={disabled}
        className={`w-full flex items-center justify-between gap-2 px-3 py-2 text-sm text-sidebar-text hover:text-white hover:bg-sidebar-hover rounded-lg transition-colors ${
          disabled ? 'opacity-50 cursor-not-allowed' : ''
        }`}
      >
        <div className="flex items-center gap-2 min-w-0">
          {icon}
          <span className="truncate">{value}</span>
        </div>
        <ChevronIcon open={isOpen} />
      </button>

      {isOpen && (
        <div className="absolute left-0 right-0 mt-1 bg-gray-800 border border-gray-700 rounded-lg shadow-xl z-50 max-h-60 overflow-y-auto">
          {children}
        </div>
      )}
    </div>
  );
}

interface DropdownItemProps {
  icon: React.ReactNode;
  label: string;
  selected: boolean;
  onClick: () => void;
  indent?: boolean;
}

function DropdownItem({ icon, label, selected, onClick, indent }: DropdownItemProps) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center justify-between px-3 py-2.5 text-sm hover:bg-gray-700 transition-colors ${
        selected ? 'text-white bg-gray-700' : 'text-gray-300'
      } ${indent ? 'pl-8' : ''}`}
    >
      <div className="flex items-center gap-2">
        {icon}
        <span>{label}</span>
      </div>
      {selected && <CheckIcon />}
    </button>
  );
}

export function ScopePicker() {
  const {
    scope,
    userScope,
    loading,
    setScopeToOrganization,
    setScopeToClient,
    clearOrganization,
    clearClient,
    canViewPlatform,
  } = useScope();

  const [orgDropdownOpen, setOrgDropdownOpen] = useState(false);
  const [clientDropdownOpen, setClientDropdownOpen] = useState(false);

  if (loading || !userScope) {
    return (
      <div className="px-4 py-3 border-b border-sidebar-hover">
        <div className="animate-pulse space-y-3">
          <div className="h-4 bg-sidebar-hover rounded w-16"></div>
          <div className="h-8 bg-sidebar-hover rounded"></div>
          <div className="h-4 bg-sidebar-hover rounded w-16"></div>
          <div className="h-8 bg-sidebar-hover rounded"></div>
        </div>
      </div>
    );
  }

  // Get available organizations
  const organizations = userScope.organizations;

  // Get clients for the currently selected organization
  const getAvailableClients = (): Client[] => {
    if (!scope.organization_id) return [];
    return userScope.clients.filter(c => c.organization_id === scope.organization_id);
  };

  const availableClients = getAvailableClients();

  // Current selection display values
  const selectedOrgName = scope.organization_name || 'None';
  const selectedClientName = scope.client_name || 'None';

  const handleSelectOrganization = (org: Organization) => {
    setScopeToOrganization(org);
    setOrgDropdownOpen(false);
  };

  const handleClearOrganization = () => {
    clearOrganization();
    setOrgDropdownOpen(false);
  };

  const handleSelectClient = (client: Client) => {
    const org = userScope.organizations.find(o => o.id === client.organization_id);
    if (org) {
      setScopeToClient(client, org);
    }
    setClientDropdownOpen(false);
  };

  const handleClearClient = () => {
    clearClient();
    setClientDropdownOpen(false);
  };

  // Determine if user can clear organization (platform users can)
  const canClearOrg = canViewPlatform();

  // Client dropdown is disabled if no organization is selected
  const clientDropdownDisabled = !scope.organization_id;

  return (
    <div className="px-4 py-3 border-b border-sidebar-hover space-y-3">
      {/* Organization Selector */}
      <Dropdown
        label="Organization"
        value={selectedOrgName}
        icon={scope.organization_id ? <OrgIcon /> : <NoneIcon />}
        isOpen={orgDropdownOpen}
        onToggle={() => setOrgDropdownOpen(!orgDropdownOpen)}
        onClose={() => setOrgDropdownOpen(false)}
      >
        {/* None option for platform users */}
        {canClearOrg && (
          <DropdownItem
            icon={<NoneIcon />}
            label="None"
            selected={!scope.organization_id}
            onClick={handleClearOrganization}
          />
        )}

        {/* Divider */}
        {canClearOrg && organizations.length > 0 && (
          <div className="border-t border-gray-700 my-1"></div>
        )}

        {/* Organizations list */}
        {organizations.map(org => (
          <DropdownItem
            key={org.id}
            icon={<OrgIcon />}
            label={org.name}
            selected={scope.organization_id === org.id}
            onClick={() => handleSelectOrganization(org)}
          />
        ))}
      </Dropdown>

      {/* Client Selector */}
      <Dropdown
        label="Client"
        value={clientDropdownDisabled ? 'Select org first' : selectedClientName}
        icon={scope.client_id ? <ClientIcon /> : <NoneIcon />}
        isOpen={clientDropdownOpen}
        onToggle={() => !clientDropdownDisabled && setClientDropdownOpen(!clientDropdownOpen)}
        onClose={() => setClientDropdownOpen(false)}
        disabled={clientDropdownDisabled}
      >
        {/* None option - always available when org is selected */}
        <DropdownItem
          icon={<NoneIcon />}
          label="None"
          selected={!scope.client_id}
          onClick={handleClearClient}
        />

        {/* Divider */}
        {availableClients.length > 0 && (
          <div className="border-t border-gray-700 my-1"></div>
        )}

        {/* Clients list */}
        {availableClients.map(client => (
          <DropdownItem
            key={client.id}
            icon={<ClientIcon />}
            label={client.name}
            selected={scope.client_id === client.id}
            onClick={() => handleSelectClient(client)}
          />
        ))}

        {/* Empty state */}
        {availableClients.length === 0 && (
          <div className="px-3 py-2 text-sm text-gray-500 italic">
            No clients in this organization
          </div>
        )}
      </Dropdown>
    </div>
  );
}
