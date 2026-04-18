import { createContext, useContext, useState, useEffect } from 'react';
import { getCurrentUser, login as loginAPI, logout as logoutAPI, register as registerAPI } from '../services/auth.js';

const AuthContext = createContext();

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  const initUser = async () => {
    try {
      const data = await getCurrentUser();
      setUser(data.user ?? null);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    initUser();
  }, []);

  const login = async (data) => {
    try {
      const userData = await loginAPI(data);
      setUser(userData.user);
      return { success: true };
    } catch (error) {
      return { success: false, error: error.message || 'Login failed' };
    }
  };

  const register = async (input) => {
    try {
      const data = await registerAPI(input);
      setUser(data.user);
      return { success: true};
    } catch (error) {
      return { success: false, error: error.message || 'Register failed'}
    }
  }

  const logout = async () => {
    try {
      await logoutAPI();
    } catch {
      // Ignore errors on logout
    }
    setUser(null);
  };

  const value = {
    user,
    loading,
    login,
    logout,
    register,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
