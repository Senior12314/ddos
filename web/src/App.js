import React, { useState, useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout, ConfigProvider, message } from 'antd';
import { io } from 'socket.io-client';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import Dashboard from './pages/Dashboard';
import Endpoints from './pages/Endpoints';
import EndpointDetail from './pages/EndpointDetail';
import Nodes from './pages/Nodes';
import Blacklist from './pages/Blacklist';
import Settings from './pages/Settings';
import Login from './pages/Login';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { APIProvider } from './contexts/APIContext';
import './App.css';

const { Content } = Layout;

function AppContent() {
  const { isAuthenticated } = useAuth();
  const [socket, setSocket] = useState(null);
  const [collapsed, setCollapsed] = useState(false);

  useEffect(() => {
    if (isAuthenticated) {
      // Initialize WebSocket connection for real-time updates
      const newSocket = io('ws://localhost:8080', {
        transports: ['websocket'],
        auth: {
          token: localStorage.getItem('token')
        }
      });

      newSocket.on('connect', () => {
        console.log('Connected to WebSocket');
        message.success('Connected to real-time updates');
      });

      newSocket.on('disconnect', () => {
        console.log('Disconnected from WebSocket');
        message.warning('Disconnected from real-time updates');
      });

      newSocket.on('endpoint_update', (data) => {
        console.log('Endpoint update received:', data);
        message.info(`Endpoint ${data.name} updated`);
      });

      newSocket.on('metrics_update', (data) => {
        console.log('Metrics update received:', data);
      });

      newSocket.on('node_status_update', (data) => {
        console.log('Node status update received:', data);
      });

      setSocket(newSocket);

      return () => {
        newSocket.close();
      };
    }
  }, [isAuthenticated]);

  if (!isAuthenticated) {
    return <Login />;
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sidebar collapsed={collapsed} />
      <Layout>
        <Header 
          collapsed={collapsed} 
          onToggle={() => setCollapsed(!collapsed)} 
        />
        <Content style={{ margin: '24px', padding: '24px', background: '#fff', borderRadius: '8px' }}>
          <Routes>
            <Route path="/" element={<Dashboard socket={socket} />} />
            <Route path="/dashboard" element={<Dashboard socket={socket} />} />
            <Route path="/endpoints" element={<Endpoints socket={socket} />} />
            <Route path="/endpoints/:id" element={<EndpointDetail socket={socket} />} />
            <Route path="/nodes" element={<Nodes socket={socket} />} />
            <Route path="/blacklist" element={<Blacklist socket={socket} />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
}

function App() {
  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#1890ff',
          colorSuccess: '#52c41a',
          colorWarning: '#faad14',
          colorError: '#ff4d4f',
          colorInfo: '#1890ff',
        },
      }}
    >
      <AuthProvider>
        <APIProvider>
          <AppContent />
        </APIProvider>
      </AuthProvider>
    </ConfigProvider>
  );
}

export default App;
