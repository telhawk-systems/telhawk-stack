import { useState, useEffect, useRef, useMemo } from 'react';
import { Client } from '../types';

const ClientIcon = () => (
  <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
  </svg>
);

const SearchIcon = () => (
  <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
  </svg>
);

const CloseIcon = () => (
  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const CheckIcon = () => (
  <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
);

const NoneIcon = () => (
  <svg className="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
  </svg>
);

interface ClientSelectorModalProps {
  isOpen: boolean;
  onClose: () => void;
  clients: Client[];
  selectedClientId: string | null;
  onSelectClient: (client: Client) => void;
  onClearClient: () => void;
  organizationName?: string;
}

export function ClientSelectorModal({
  isOpen,
  onClose,
  clients,
  selectedClientId,
  onSelectClient,
  onClearClient,
  organizationName,
}: ClientSelectorModalProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const searchInputRef = useRef<HTMLInputElement>(null);
  const modalRef = useRef<HTMLDivElement>(null);

  // Reset search when modal opens
  useEffect(() => {
    if (isOpen) {
      setSearchQuery('');
      // Focus search input when modal opens
      setTimeout(() => searchInputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Handle click outside modal
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (modalRef.current && !modalRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen, onClose]);

  // Filter clients based on search query
  const filteredClients = useMemo(() => {
    if (!searchQuery.trim()) return clients;
    const query = searchQuery.toLowerCase();
    return clients.filter(client =>
      client.name.toLowerCase().includes(query)
    );
  }, [clients, searchQuery]);

  if (!isOpen) return null;

  const handleSelectClient = (client: Client) => {
    onSelectClient(client);
    onClose();
  };

  const handleClearClient = () => {
    onClearClient();
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={modalRef}
        className="bg-gray-800 rounded-xl shadow-2xl border border-gray-700 w-full max-w-md mx-4 max-h-[80vh] flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-700">
          <div>
            <h2 className="text-lg font-semibold text-white">Select Client</h2>
            {organizationName && (
              <p className="text-sm text-gray-400 mt-0.5">
                Organization: {organizationName}
              </p>
            )}
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
          >
            <CloseIcon />
          </button>
        </div>

        {/* Search Bar */}
        <div className="px-5 py-3 border-b border-gray-700">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <SearchIcon />
            </div>
            <input
              ref={searchInputRef}
              type="text"
              placeholder="Search clients..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2.5 bg-gray-900 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <p className="text-xs text-gray-500 mt-2">
            {filteredClients.length} of {clients.length} clients
          </p>
        </div>

        {/* Client List */}
        <div className="flex-1 overflow-y-auto">
          {/* None option */}
          <button
            onClick={handleClearClient}
            className={`w-full flex items-center justify-between px-5 py-3 hover:bg-gray-700 transition-colors ${
              !selectedClientId ? 'bg-gray-700/50' : ''
            }`}
          >
            <div className="flex items-center gap-3">
              <NoneIcon />
              <span className="text-gray-300">None (view all)</span>
            </div>
            {!selectedClientId && <CheckIcon />}
          </button>

          {/* Divider */}
          {filteredClients.length > 0 && (
            <div className="border-t border-gray-700"></div>
          )}

          {/* Clients */}
          {filteredClients.map(client => (
            <button
              key={client.id}
              onClick={() => handleSelectClient(client)}
              className={`w-full flex items-center justify-between px-5 py-3 hover:bg-gray-700 transition-colors ${
                selectedClientId === client.id ? 'bg-gray-700/50' : ''
              }`}
            >
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <ClientIcon />
                <span className="text-white text-left break-words">{client.name}</span>
              </div>
              {selectedClientId === client.id && (
                <div className="ml-3 flex-shrink-0">
                  <CheckIcon />
                </div>
              )}
            </button>
          ))}

          {/* No results */}
          {filteredClients.length === 0 && searchQuery && (
            <div className="px-5 py-8 text-center">
              <p className="text-gray-400">No clients match "{searchQuery}"</p>
              <button
                onClick={() => setSearchQuery('')}
                className="mt-2 text-sm text-blue-400 hover:text-blue-300"
              >
                Clear search
              </button>
            </div>
          )}

          {/* Empty state */}
          {clients.length === 0 && (
            <div className="px-5 py-8 text-center text-gray-400">
              No clients in this organization
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
