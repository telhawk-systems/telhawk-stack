import React, { useState, useRef, useEffect } from 'react';
import { EventClassConfig, searchEventClasses } from '../types/eventClasses';

export interface EventClassDropdownProps {
  onSelect: (eventClass: EventClassConfig) => void;
  onClose: () => void;
  isOpen: boolean;
}

/**
 * EventClassDropdown - Searchable dropdown for selecting OCSF event classes
 *
 * Design philosophy: Progressive disclosure - start simple with event class filter.
 * Type to filter, visual icons for quick scanning.
 */
export function EventClassDropdown({ onSelect, onClose, isOpen }: EventClassDropdownProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [filteredClasses, setFilteredClasses] = useState<EventClassConfig[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) {
      // Focus input when dropdown opens
      inputRef.current?.focus();
      // Initialize with all event classes
      setFilteredClasses(searchEventClasses(''));
    }
  }, [isOpen]);

  useEffect(() => {
    // Filter event classes based on search query
    setFilteredClasses(searchEventClasses(searchQuery));
  }, [searchQuery]);

  useEffect(() => {
    // Close dropdown when clicking outside
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen, onClose]);

  const handleSelect = (eventClass: EventClassConfig) => {
    onSelect(eventClass);
    setSearchQuery('');
    onClose();
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
    } else if (event.key === 'Enter' && filteredClasses.length > 0) {
      handleSelect(filteredClasses[0]);
    }
  };

  if (!isOpen) return null;

  return (
    <div ref={dropdownRef} className="absolute z-10 mt-2 w-96 bg-white rounded-lg shadow-lg border border-gray-200">
      {/* Search Input */}
      <div className="p-3 border-b border-gray-200">
        <input
          ref={inputRef}
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search event classes..."
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {/* Event Class List */}
      <div className="max-h-96 overflow-y-auto">
        {filteredClasses.length === 0 ? (
          <div className="p-4 text-center text-gray-500">
            No event classes found matching "{searchQuery}"
          </div>
        ) : (
          <div className="py-2">
            {filteredClasses.map((eventClass) => (
              <button
                key={eventClass.classUid}
                onClick={() => handleSelect(eventClass)}
                className="w-full text-left px-4 py-3 hover:bg-gray-50 transition-colors border-b border-gray-100 last:border-b-0"
              >
                <div className="flex items-start gap-3">
                  <span className="text-2xl leading-none mt-0.5">{eventClass.icon}</span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900">{eventClass.name}</span>
                      <span className="text-xs text-gray-500">({eventClass.classUid})</span>
                    </div>
                    <p className="text-sm text-gray-600 mt-0.5">{eventClass.description}</p>
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
