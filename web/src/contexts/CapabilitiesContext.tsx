import { createContext, useContext, ReactNode } from 'react'
import { useCapabilities } from '../api/client'
import type { Capabilities } from '../types'

// Default capabilities - all enabled (for local development / when API fails)
const defaultCapabilities: Capabilities = {
  exec: true,
  logs: true,
  portForward: true,
  secrets: true,
}

const CapabilitiesContext = createContext<Capabilities>(defaultCapabilities)

export function CapabilitiesProvider({ children }: { children: ReactNode }) {
  const { data: capabilities } = useCapabilities()

  // Use fetched capabilities, fall back to defaults if not loaded yet
  const value = capabilities ?? defaultCapabilities

  return (
    <CapabilitiesContext.Provider value={value}>
      {children}
    </CapabilitiesContext.Provider>
  )
}

export function useCapabilitiesContext(): Capabilities {
  return useContext(CapabilitiesContext)
}

// Convenience hooks for specific capabilities
export function useCanExec(): boolean {
  return useContext(CapabilitiesContext).exec
}

export function useCanViewLogs(): boolean {
  return useContext(CapabilitiesContext).logs
}

export function useCanPortForward(): boolean {
  return useContext(CapabilitiesContext).portForward
}

export function useCanViewSecrets(): boolean {
  return useContext(CapabilitiesContext).secrets
}
