import React, { createContext, useContext, useState, useEffect } from 'react';
import { message } from 'antd';
import api from '../services/api';

const AuthContext = createContext();

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check if user is already authenticated
    const token = localStorage.getItem('token');
    if (token) {
      // Validate token with server
      validateToken(token);
    } else {
      setLoading(false);
    }
  }, []);

  const validateToken = async (token) => {
    try {
      const response = await api.get('/users/profile', {
        headers: {
          Authorization: `Bearer ${token}`
        }
      });
      
      setUser(response.data);
      setIsAuthenticated(true);
    } catch (error) {
      console.error('Token validation failed:', error);
      localStorage.removeItem('token');
      setIsAuthenticated(false);
    } finally {
      setLoading(false);
    }
  };

  const login = async (email, password) => {
    try {
      setLoading(true);
      
      // For demo purposes, we'll use a simple token
      // In production, this would be a real login endpoint
      if (email === 'admin@cloudnordsp.com' && password === 'admin123') {
        const token = 'valid-token';
        localStorage.setItem('token', token);
        
        // Set user data
        const userData = {
          id: 'user-123',
          email: email,
          username: 'admin',
          role: 'admin',
          active: true
        };
        
        setUser(userData);
        setIsAuthenticated(true);
        message.success('Login successful');
        return true;
      } else {
        message.error('Invalid credentials');
        return false;
      }
    } catch (error) {
      console.error('Login error:', error);
      message.error('Login failed');
      return false;
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setUser(null);
    setIsAuthenticated(false);
    message.success('Logged out successfully');
  };

  const value = {
    user,
    isAuthenticated,
    loading,
    login,
    logout
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};
