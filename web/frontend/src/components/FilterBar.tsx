import { useState } from 'react';
import { FilterChip } from './FilterChip';
import { EventClassDropdown } from './EventClassDropdown';
import { FieldFilterDropdown } from './FieldFilterDropdown';
import { EventClassConfig, getEventClassFilters, FieldFilterConfig } from '../types/eventClasses';

export interface Filter {
  id: string;
  type: 'event-class' | 'field';
  field?: string;
  value: string;
  label: string;
  icon?: string;
}

export interface FilterBarProps {
  onFiltersChange: (filters: Filter[]) => void;
  disabled?: boolean;
}

/**
 * FilterBar - Progressive disclosure filter interface
 *
 * Design Philosophy:
 * Level 1: Event class filter (default state)
 * Level 2: Type-specific filters (after class selection)
 * Level 3: Active filters (chip-based display)
 *
 * Everything is visible at a glance. No hidden filters.
 */
export function FilterBar({ onFiltersChange, disabled = false }: FilterBarProps) {
  const [filters, setFilters] = useState<Filter[]>([]);
  const [showEventClassDropdown, setShowEventClassDropdown] = useState(false);
  const [activeFieldFilter, setActiveFieldFilter] = useState<FieldFilterConfig | null>(null);
  const [showSecondaryFilters, setShowSecondaryFilters] = useState(false);

  const selectedEventClass = filters.find(f => f.type === 'event-class');
  const selectedClassUid = selectedEventClass ? parseInt(selectedEventClass.value) : null;

  // Get available field filters based on selected event class
  const primaryFilters = selectedClassUid ? getEventClassFilters(selectedClassUid, true, false) : [];
  const secondaryFilters = selectedClassUid ? getEventClassFilters(selectedClassUid, false, true) : [];

  const handleEventClassSelect = (eventClass: EventClassConfig) => {
    // Remove existing event class filter and all field filters
    const eventClassFilter: Filter = {
      id: `event-class-${eventClass.classUid}`,
      type: 'event-class',
      value: String(eventClass.classUid),
      label: eventClass.name,
      icon: eventClass.icon,
    };

    // When changing event class, clear all filters and start fresh
    const updatedFilters = [eventClassFilter];
    setFilters(updatedFilters);
    onFiltersChange(updatedFilters);
    setShowSecondaryFilters(false);
  };

  const handleFieldFilterSelect = (fieldConfig: FieldFilterConfig, value: string) => {
    const fieldFilter: Filter = {
      id: `field-${fieldConfig.id}-${Date.now()}`, // Unique ID allows multiple values for same field
      type: 'field',
      field: fieldConfig.id,
      value: value,
      label: `${fieldConfig.label}: ${value}`,
    };

    const updatedFilters = [...filters, fieldFilter];
    setFilters(updatedFilters);
    onFiltersChange(updatedFilters);
    setActiveFieldFilter(null);
  };

  const handleRemoveFilter = (filterId: string) => {
    const updatedFilters = filters.filter(f => f.id !== filterId);
    setFilters(updatedFilters);
    onFiltersChange(updatedFilters);
  };

  return (
    <div className={`bg-white rounded-lg shadow-md p-6 space-y-4 ${disabled ? 'opacity-60' : ''}`}>
      <h2 className="text-2xl font-bold text-gray-800">Search Events</h2>

      {/* Active Filters Section */}
      {filters.length > 0 && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Active Filters:
          </label>
          <div className="flex flex-wrap gap-2">
            {filters.map((filter) => (
              <FilterChip
                key={filter.id}
                label={filter.label}
                icon={filter.icon}
                variant={filter.type === 'event-class' ? 'event-class' : 'default'}
                onRemove={() => handleRemoveFilter(filter.id)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Add Filter Section */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Add Filter:
        </label>

        {/* Event Class Filter Button */}
        <div className="relative inline-block mb-3">
          <button
            onClick={() => setShowEventClassDropdown(!showEventClassDropdown)}
            disabled={disabled}
            className="px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
          >
            {selectedEventClass ? (
              <>
                <span>{selectedEventClass.icon}</span>
                <span>Change Event Class</span>
              </>
            ) : (
              <>
                <span>Filter by Event Class</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </>
            )}
          </button>

          <EventClassDropdown
            isOpen={showEventClassDropdown}
            onSelect={handleEventClassSelect}
            onClose={() => setShowEventClassDropdown(false)}
          />
        </div>

        {/* Type-Specific Field Filter Buttons (Level 2) */}
        {selectedEventClass && primaryFilters.length > 0 && (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2">
              {primaryFilters.map((fieldConfig) => (
                <div key={fieldConfig.id} className="relative">
                  <button
                    onClick={() => setActiveFieldFilter(activeFieldFilter?.id === fieldConfig.id ? null : fieldConfig)}
                    disabled={disabled}
                    className={`px-3 py-2 rounded-md text-sm font-medium transition-colors flex items-center gap-1 ${
                      activeFieldFilter?.id === fieldConfig.id
                        ? 'bg-blue-100 text-blue-700 border border-blue-300'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200 border border-gray-300'
                    }`}
                  >
                    <span>{fieldConfig.label}</span>
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>

                  <FieldFilterDropdown
                    fieldConfig={fieldConfig}
                    isOpen={activeFieldFilter?.id === fieldConfig.id}
                    onSelect={(value) => handleFieldFilterSelect(fieldConfig, value)}
                    onClose={() => setActiveFieldFilter(null)}
                  />
                </div>
              ))}

              {/* Show More Button */}
              {secondaryFilters.length > 0 && (
                <button
                  onClick={() => setShowSecondaryFilters(!showSecondaryFilters)}
                  disabled={disabled}
                  className="px-3 py-2 bg-gray-100 text-gray-600 rounded-md hover:bg-gray-200 text-sm font-medium transition-colors"
                >
                  {showSecondaryFilters ? '− Less' : '+ More'}
                </button>
              )}
            </div>

            {/* Secondary Filters (Hidden by default) */}
            {showSecondaryFilters && secondaryFilters.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2 border-t border-gray-200">
                {secondaryFilters.map((fieldConfig) => (
                  <div key={fieldConfig.id} className="relative">
                    <button
                      onClick={() => setActiveFieldFilter(activeFieldFilter?.id === fieldConfig.id ? null : fieldConfig)}
                      disabled={disabled}
                      className={`px-3 py-2 rounded-md text-sm font-medium transition-colors flex items-center gap-1 ${
                        activeFieldFilter?.id === fieldConfig.id
                          ? 'bg-blue-100 text-blue-700 border border-blue-300'
                          : 'bg-gray-100 text-gray-700 hover:bg-gray-200 border border-gray-300'
                      }`}
                    >
                      <span>{fieldConfig.label}</span>
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>

                    <FieldFilterDropdown
                      fieldConfig={fieldConfig}
                      isOpen={activeFieldFilter?.id === fieldConfig.id}
                      onSelect={(value) => handleFieldFilterSelect(fieldConfig, value)}
                      onClose={() => setActiveFieldFilter(null)}
                    />
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Helper Text */}
      <div className="text-xs text-gray-500 border-t border-gray-200 pt-3">
        <p>Start by selecting an event class to filter events. More filter options will appear based on your selection.</p>
      </div>
    </div>
  );
}
