import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { clsx } from 'clsx'
import { ChevronDown, Search, X } from 'lucide-react'

interface Namespace {
  name: string
}

interface NamespaceSelectorProps {
  value: string
  onChange: (value: string) => void
  namespaces: Namespace[] | undefined
  className?: string
}

export function NamespaceSelector({
  value,
  onChange,
  namespaces,
  className,
}: NamespaceSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [highlightedIndex, setHighlightedIndex] = useState(0)
  const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 })

  const triggerRef = useRef<HTMLButtonElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)

  // Sort and filter namespaces
  const sortedNamespaces = useMemo(() => {
    if (!namespaces) return []
    return [...namespaces].sort((a, b) => a.name.localeCompare(b.name))
  }, [namespaces])

  const filteredNamespaces = useMemo(() => {
    if (!search.trim()) return sortedNamespaces
    const searchLower = search.toLowerCase()
    return sortedNamespaces.filter((ns) =>
      ns.name.toLowerCase().includes(searchLower)
    )
  }, [sortedNamespaces, search])

  // Build options list: "All Namespaces" + filtered namespaces
  const options = useMemo(() => {
    const allOption = { value: '', label: 'All Namespaces' }
    const nsOptions = filteredNamespaces.map((ns) => ({ value: ns.name, label: ns.name }))

    // Only include "All Namespaces" if it matches the search or search is empty
    if (!search.trim() || 'all namespaces'.includes(search.toLowerCase())) {
      return [allOption, ...nsOptions]
    }
    return nsOptions
  }, [filteredNamespaces, search])

  // Update position when dropdown opens
  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    const dropdownWidth = Math.max(rect.width, 200) // Minimum width of 200px for better search UX
    // Align dropdown to the right edge of the button
    const left = rect.right - dropdownWidth
    setDropdownPosition({
      top: rect.bottom + 4,
      left: Math.max(8, left), // Ensure at least 8px from screen edge
      width: dropdownWidth,
    })
  }, [])

  // Open dropdown
  const openDropdown = useCallback(() => {
    setIsOpen(true)
    setSearch('')
    setHighlightedIndex(0)
    updatePosition()
  }, [updatePosition])

  // Close dropdown
  const closeDropdown = useCallback(() => {
    setIsOpen(false)
    setSearch('')
  }, [])

  // Select an option
  const selectOption = useCallback((optionValue: string) => {
    onChange(optionValue)
    closeDropdown()
  }, [onChange, closeDropdown])

  // Focus search input when dropdown opens
  useEffect(() => {
    if (isOpen) {
      // Small delay to ensure the dropdown is rendered
      requestAnimationFrame(() => {
        searchInputRef.current?.focus()
      })
    }
  }, [isOpen])

  // Reset highlighted index when filtered options change
  useEffect(() => {
    setHighlightedIndex(0)
  }, [filteredNamespaces])

  // Handle click outside
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node
      if (
        triggerRef.current?.contains(target) ||
        dropdownRef.current?.contains(target)
      ) {
        return
      }
      closeDropdown()
    }

    // Small delay to prevent immediate close on open click
    const timeoutId = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside)
    }, 0)

    return () => {
      clearTimeout(timeoutId)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen, closeDropdown])

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setHighlightedIndex((prev) =>
          prev < options.length - 1 ? prev + 1 : prev
        )
        break
      case 'ArrowUp':
        e.preventDefault()
        setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : 0))
        break
      case 'Enter':
        e.preventDefault()
        if (options[highlightedIndex]) {
          selectOption(options[highlightedIndex].value)
        }
        break
      case 'Escape':
        e.preventDefault()
        closeDropdown()
        break
      case 'Tab':
        closeDropdown()
        break
    }
  }, [options, highlightedIndex, selectOption, closeDropdown])

  // Scroll highlighted item into view
  useEffect(() => {
    if (!isOpen || !dropdownRef.current) return
    const highlighted = dropdownRef.current.querySelector('[data-highlighted="true"]')
    if (highlighted) {
      highlighted.scrollIntoView({ block: 'nearest' })
    }
  }, [highlightedIndex, isOpen])

  // Get display value
  const displayValue = value || 'All Namespaces'

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => (isOpen ? closeDropdown() : openDropdown())}
        className={clsx(
          'appearance-none bg-theme-elevated text-theme-text-primary text-xs rounded px-2 py-1 pr-6 border border-theme-border-light',
          'focus:outline-none focus:ring-1 focus:ring-blue-500 min-w-[100px] text-left relative',
          'hover:bg-theme-hover transition-colors',
          className
        )}
      >
        <span className="block truncate">{displayValue}</span>
        <ChevronDown
          className={clsx(
            'absolute right-1.5 top-1/2 -translate-y-1/2 w-3 h-3 text-theme-text-secondary transition-transform',
            isOpen && 'rotate-180'
          )}
        />
      </button>

      {isOpen &&
        createPortal(
          <div
            ref={dropdownRef}
            className="fixed z-[9999] bg-theme-elevated border border-theme-border rounded-md shadow-lg overflow-hidden"
            style={{
              top: dropdownPosition.top,
              left: dropdownPosition.left,
              width: dropdownPosition.width,
            }}
            onKeyDown={handleKeyDown}
          >
            {/* Search input */}
            <div className="p-2 border-b border-theme-border">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-theme-text-tertiary" />
                <input
                  ref={searchInputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search namespaces..."
                  className="w-full bg-theme-base text-theme-text-primary text-xs rounded px-2 py-1.5 pl-7 pr-7 border border-theme-border-light focus:outline-none focus:ring-1 focus:ring-blue-500 placeholder:text-theme-text-tertiary"
                />
                {search && (
                  <button
                    type="button"
                    onClick={() => setSearch('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-theme-text-tertiary hover:text-theme-text-secondary"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>

            {/* Options list */}
            <div className="max-h-[240px] overflow-y-auto">
              {options.length === 0 ? (
                <div className="px-3 py-6 text-center text-xs text-theme-text-tertiary">
                  No namespaces match "{search}"
                </div>
              ) : (
                options.map((option, index) => (
                  <button
                    key={option.value}
                    type="button"
                    data-highlighted={index === highlightedIndex}
                    onClick={() => selectOption(option.value)}
                    onMouseEnter={() => setHighlightedIndex(index)}
                    className={clsx(
                      'w-full text-left px-3 py-1.5 text-xs transition-colors',
                      option.value === value
                        ? 'text-blue-400 bg-blue-500/10'
                        : 'text-theme-text-primary',
                      index === highlightedIndex && 'bg-theme-hover'
                    )}
                  >
                    {option.label}
                  </button>
                ))
              )}
            </div>

            {/* Namespace count */}
            {sortedNamespaces.length > 0 && (
              <div className="px-3 py-1.5 text-[10px] text-theme-text-tertiary border-t border-theme-border bg-theme-base">
                {filteredNamespaces.length === sortedNamespaces.length
                  ? `${sortedNamespaces.length} namespaces`
                  : `${filteredNamespaces.length} of ${sortedNamespaces.length} namespaces`}
              </div>
            )}
          </div>,
          document.body
        )}
    </>
  )
}
