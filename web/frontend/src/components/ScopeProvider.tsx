import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { ViewingScope, UserScope, Organization, Client } from '../types';
import { useAuth } from './AuthProvider';
import { apiClient } from '../services/api';

interface ScopeContextType {
  // Current viewing scope
  scope: ViewingScope;
  // User's available scope options
  userScope: UserScope | null;
  // Loading state
  loading: boolean;
  // Actions
  setScope: (scope: ViewingScope) => void;
  setScopeToOrganization: (org: Organization) => void;
  setScopeToClient: (client: Client, org: Organization) => void;
  setScopeToPlatform: () => void;
  clearOrganization: () => void;
  clearClient: () => void;
  // Helpers
  canViewPlatform: () => boolean;
  canViewOrganization: (orgId: string) => boolean;
  canViewClient: (clientId: string) => boolean;
  hasClientSelected: () => boolean;
}

const ScopeContext = createContext<ScopeContextType | undefined>(undefined);

const SCOPE_STORAGE_KEY = 'telhawk_viewing_scope';

// Default scope for users without platform access
const getDefaultScope = (userScope: UserScope | null): ViewingScope => {
  if (!userScope) {
    return { type: 'platform' };
  }

  // Platform users default to platform view (no org/client selected)
  if (userScope.max_tier === 'platform') {
    return { type: 'platform' };
  }

  // Org users default to their first org (no client selected)
  if (userScope.max_tier === 'organization' && userScope.organizations.length > 0) {
    const org = userScope.organizations[0];
    return {
      type: 'organization',
      organization_id: org.id,
      organization_name: org.name,
    };
  }

  // Client users default to their first client
  if (userScope.clients.length > 0) {
    const client = userScope.clients[0];
    const org = userScope.organizations.find(o => o.id === client.organization_id);
    return {
      type: 'client',
      organization_id: client.organization_id,
      organization_name: org?.name,
      client_id: client.id,
      client_name: client.name,
    };
  }

  return { type: 'platform' };
};

export function ScopeProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const [scope, setInternalScope] = useState<ViewingScope>({ type: 'platform' });
  const [userScope, setUserScope] = useState<UserScope | null>(null);
  const [loading, setLoading] = useState(true);

  // Load user's accessible scope when user changes
  useEffect(() => {
    if (!user) {
      setUserScope(null);
      setLoading(false);
      return;
    }

    const loadUserScope = async () => {
      setLoading(true);
      try {
        // Fetch user's accessible scope from backend
        const fetchedScope = await apiClient.getUserScope();
        setUserScope(fetchedScope);

        // Restore saved scope or use default
        const savedScope = localStorage.getItem(SCOPE_STORAGE_KEY);
        if (savedScope) {
          try {
            const parsed = JSON.parse(savedScope) as ViewingScope;
            // Validate the saved scope is still accessible
            if (isValidScope(parsed, fetchedScope)) {
              setInternalScope(parsed);
            } else {
              setInternalScope(getDefaultScope(fetchedScope));
            }
          } catch {
            setInternalScope(getDefaultScope(fetchedScope));
          }
        } else {
          setInternalScope(getDefaultScope(fetchedScope));
        }
      } catch (error) {
        console.error('Failed to load user scope:', error);
        setUserScope(null);
      } finally {
        setLoading(false);
      }
    };

    loadUserScope();
  }, [user]);

  // Validate that a scope is accessible by the user
  const isValidScope = (scope: ViewingScope, userScope: UserScope): boolean => {
    if (scope.type === 'platform') {
      return userScope.max_tier === 'platform';
    }
    if (scope.type === 'organization') {
      if (userScope.max_tier === 'client') return false;
      if (userScope.max_tier === 'platform') return true;
      return userScope.organizations.some(o => o.id === scope.organization_id);
    }
    if (scope.type === 'client') {
      if (userScope.max_tier === 'platform') return true;
      return userScope.clients.some(c => c.id === scope.client_id);
    }
    return false;
  };

  const setScope = useCallback((newScope: ViewingScope) => {
    setInternalScope(newScope);
    localStorage.setItem(SCOPE_STORAGE_KEY, JSON.stringify(newScope));
    // Sync scope with API client so all requests include scope headers
    apiClient.setScope(newScope);
  }, []);

  // Sync initial scope with API client
  useEffect(() => {
    apiClient.setScope(scope);
  }, [scope]);

  const setScopeToPlatform = useCallback(() => {
    setScope({ type: 'platform' });
  }, [setScope]);

  const setScopeToOrganization = useCallback((org: Organization) => {
    setScope({
      type: 'organization',
      organization_id: org.id,
      organization_name: org.name,
    });
  }, [setScope]);

  const setScopeToClient = useCallback((client: Client, org: Organization) => {
    setScope({
      type: 'client',
      organization_id: org.id,
      organization_name: org.name,
      client_id: client.id,
      client_name: client.name,
    });
  }, [setScope]);

  // Clear organization selection (go back to platform view)
  const clearOrganization = useCallback(() => {
    setScope({ type: 'platform' });
  }, [setScope]);

  // Clear client selection (stay at organization level)
  const clearClient = useCallback(() => {
    if (scope.organization_id && scope.organization_name) {
      setScope({
        type: 'organization',
        organization_id: scope.organization_id,
        organization_name: scope.organization_name,
      });
    }
  }, [setScope, scope.organization_id, scope.organization_name]);

  const canViewPlatform = useCallback(() => {
    return userScope?.max_tier === 'platform';
  }, [userScope]);

  const canViewOrganization = useCallback((orgId: string) => {
    if (!userScope) return false;
    if (userScope.max_tier === 'platform') return true;
    if (userScope.max_tier === 'client') return false;
    return userScope.organizations.some(o => o.id === orgId);
  }, [userScope]);

  const canViewClient = useCallback((clientId: string) => {
    if (!userScope) return false;
    if (userScope.max_tier === 'platform') return true;
    return userScope.clients.some(c => c.id === clientId);
  }, [userScope]);

  // Check if a specific client is selected
  const hasClientSelected = useCallback(() => {
    return scope.type === 'client' && !!scope.client_id;
  }, [scope]);

  return (
    <ScopeContext.Provider value={{
      scope,
      userScope,
      loading,
      setScope,
      setScopeToOrganization,
      setScopeToClient,
      setScopeToPlatform,
      clearOrganization,
      clearClient,
      canViewPlatform,
      canViewOrganization,
      canViewClient,
      hasClientSelected,
    }}>
      {children}
    </ScopeContext.Provider>
  );
}

export function useScope() {
  const context = useContext(ScopeContext);
  if (context === undefined) {
    throw new Error('useScope must be used within a ScopeProvider');
  }
  return context;
}
