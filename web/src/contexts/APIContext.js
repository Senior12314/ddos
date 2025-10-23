import React, { createContext, useContext } from 'react';
import api from '../services/api';

const APIContext = createContext();

export const useAPI = () => {
  const context = useContext(APIContext);
  if (!context) {
    throw new Error('useAPI must be used within an APIProvider');
  }
  return context;
};

export const APIProvider = ({ children }) => {
  const getAuthHeaders = () => {
    const token = localStorage.getItem('token');
    return token ? { Authorization: `Bearer ${token}` } : {};
  };

  const endpoints = {
    // Endpoint management
    list: () => api.get('/endpoints', { headers: getAuthHeaders() }),
    get: (id) => api.get(`/endpoints/${id}`, { headers: getAuthHeaders() }),
    create: (data) => api.post('/endpoints', data, { headers: getAuthHeaders() }),
    update: (id, data) => api.put(`/endpoints/${id}`, data, { headers: getAuthHeaders() }),
    delete: (id) => api.delete(`/endpoints/${id}`, { headers: getAuthHeaders() }),
    getMetrics: (id, since = '1h') => api.get(`/endpoints/${id}/metrics?since=${since}`, { headers: getAuthHeaders() }),
    
    // Whitelist management
    addToWhitelist: (endpointId, data) => api.post(`/endpoints/${endpointId}/whitelist`, data, { headers: getAuthHeaders() }),
    removeFromWhitelist: (endpointId, ip) => api.delete(`/endpoints/${endpointId}/whitelist/${ip}`, { headers: getAuthHeaders() }),
    getWhitelist: (endpointId) => api.get(`/endpoints/${endpointId}/whitelist`, { headers: getAuthHeaders() }),
  };

  const nodes = {
    list: () => api.get('/nodes', { headers: getAuthHeaders() }),
    get: (id) => api.get(`/nodes/${id}`, { headers: getAuthHeaders() }),
    getStatus: (id) => api.get(`/nodes/${id}/status`, { headers: getAuthHeaders() }),
  };

  const blacklist = {
    list: () => api.get('/blacklist', { headers: getAuthHeaders() }),
    add: (data) => api.post('/blacklist', data, { headers: getAuthHeaders() }),
    remove: (ip) => api.delete(`/blacklist/${ip}`, { headers: getAuthHeaders() }),
  };

  const system = {
    getStatus: () => api.get('/system/status', { headers: getAuthHeaders() }),
    getStats: () => api.get('/system/stats', { headers: getAuthHeaders() }),
  };

  const users = {
    getProfile: () => api.get('/users/profile', { headers: getAuthHeaders() }),
    updateProfile: (data) => api.put('/users/profile', data, { headers: getAuthHeaders() }),
  };

  const organizations = {
    list: () => api.get('/organizations', { headers: getAuthHeaders() }),
    get: (id) => api.get(`/organizations/${id}`, { headers: getAuthHeaders() }),
    update: (id, data) => api.put(`/organizations/${id}`, data, { headers: getAuthHeaders() }),
  };

  const value = {
    endpoints,
    nodes,
    blacklist,
    system,
    users,
    organizations,
  };

  return (
    <APIContext.Provider value={value}>
      {children}
    </APIContext.Provider>
  );
};
