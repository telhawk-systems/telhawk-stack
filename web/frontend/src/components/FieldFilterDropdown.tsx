import { useState, useRef, useEffect } from 'react';
import { FieldFilterConfig } from '../types/eventClasses';

export interface FieldFilterDropdownProps {
  fieldConfig: FieldFilterConfig;
  onSelect: (value: string) => void;
  onClose: () => void;
  isOpen: boolean;
}

/**
 * FieldFilterDropdown - Dropdown for selecting field filter values
 *
 * Supports different input types:
 * - text/ip: Free-form text input
 * - number: Numeric input
 * - enum: Predefined list of values
 */
export function FieldFilterDropdown({ fieldConfig, onSelect, onClose, isOpen }: FieldFilterDropdownProps) {
  const [inputValue, setInputValue] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isOpen) {
      inputRef.current?.focus();
      setInputValue('');
    }
  }, [isOpen]);

  useEffect(() => {
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

  const handleSelect = (value: string) => {
    if (value.trim()) {
      onSelect(value.trim());
      setInputValue('');
      onClose();
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
    } else if (event.key === 'Enter' && inputValue.trim()) {
      handleSelect(inputValue);
    }
  };

  if (!isOpen) return null;

  return (
    <div ref={dropdownRef} className="absolute z-10 mt-2 w-80 bg-white rounded-lg shadow-lg border border-gray-200">
      <div className="p-3">
        <label className="block text-xs font-medium text-gray-600 mb-2">
          Enter {fieldConfig.label}
        </label>

        {/* Text/IP/Number Input */}
        {(fieldConfig.type === 'text' || fieldConfig.type === 'ip' || fieldConfig.type === 'number') && (
          <div className="space-y-2">
            <input
              ref={inputRef}
              type={fieldConfig.type === 'number' ? 'number' : 'text'}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={`Enter ${fieldConfig.label.toLowerCase()}...`}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <button
              onClick={() => handleSelect(inputValue)}
              disabled={!inputValue.trim()}
              className="w-full px-3 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Add Filter
            </button>
          </div>
        )}

        {/* Enum Selection */}
        {fieldConfig.type === 'enum' && fieldConfig.enumValues && (
          <div>
            <input
              ref={inputRef}
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Search or enter custom value..."
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent mb-2"
            />

            <div className="max-h-48 overflow-y-auto border-t border-gray-200">
              {fieldConfig.enumValues
                .filter(val =>
                  !inputValue || val.toLowerCase().includes(inputValue.toLowerCase())
                )
                .map((value) => (
                  <button
                    key={value}
                    onClick={() => handleSelect(value)}
                    className="w-full text-left px-3 py-2 hover:bg-gray-50 transition-colors border-b border-gray-100 last:border-b-0"
                  >
                    {value}
                  </button>
                ))}

              {inputValue && !fieldConfig.enumValues.includes(inputValue) && (
                <button
                  onClick={() => handleSelect(inputValue)}
                  className="w-full text-left px-3 py-2 bg-blue-50 text-blue-700 hover:bg-blue-100 transition-colors border-b border-blue-200"
                >
                  Add custom: "{inputValue}"
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
