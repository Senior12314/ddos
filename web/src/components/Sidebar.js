import React from 'react';
import { Layout, Menu } from 'antd';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  DashboardOutlined,
  ServerOutlined,
  NodeIndexOutlined,
  SecurityScanOutlined,
  SettingOutlined,
  ShieldCheckOutlined,
} from '@ant-design/icons';

const { Sider } = Layout;

const Sidebar = ({ collapsed }) => {
  const navigate = useNavigate();
  const location = useLocation();

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: 'Dashboard',
    },
    {
      key: '/endpoints',
      icon: <ServerOutlined />,
      label: 'Protected Endpoints',
    },
    {
      key: '/nodes',
      icon: <NodeIndexOutlined />,
      label: 'Edge Nodes',
    },
    {
      key: '/blacklist',
      icon: <SecurityScanOutlined />,
      label: 'Blacklist',
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: 'Settings',
    },
  ];

  const handleMenuClick = ({ key }) => {
    navigate(key);
  };

  return (
    <Sider 
      trigger={null} 
      collapsible 
      collapsed={collapsed}
      style={{
        overflow: 'auto',
        height: '100vh',
        position: 'fixed',
        left: 0,
        top: 0,
        bottom: 0,
      }}
    >
      <div style={{ 
        height: 32, 
        margin: 16, 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'center',
        color: '#fff',
        fontSize: collapsed ? '16px' : '18px',
        fontWeight: 'bold'
      }}>
        {collapsed ? <ShieldCheckOutlined /> : 'CloudNordSP'}
      </div>
      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={[location.pathname]}
        items={menuItems}
        onClick={handleMenuClick}
      />
    </Sider>
  );
};

export default Sidebar;
