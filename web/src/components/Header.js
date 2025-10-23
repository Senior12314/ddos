import React from 'react';
import { Layout, Button, Dropdown, Avatar, Space, Badge } from 'antd';
import { 
  MenuFoldOutlined, 
  MenuUnfoldOutlined, 
  BellOutlined, 
  UserOutlined,
  LogoutOutlined,
  SettingOutlined
} from '@ant-design/icons';
import { useAuth } from '../contexts/AuthContext';

const { Header: AntHeader } = Layout;

const Header = ({ collapsed, onToggle }) => {
  const { user, logout } = useAuth();

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: 'Profile',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: 'Settings',
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Logout',
      onClick: logout,
    },
  ];

  return (
    <AntHeader style={{ 
      padding: '0 24px', 
      background: '#fff', 
      display: 'flex', 
      alignItems: 'center', 
      justifyContent: 'space-between',
      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
      zIndex: 1,
      position: 'sticky',
      top: 0
    }}>
      <Button
        type="text"
        icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
        onClick={onToggle}
        style={{
          fontSize: '16px',
          width: 64,
          height: 64,
        }}
      />
      
      <Space size="middle">
        <Badge count={0} size="small">
          <Button 
            type="text" 
            icon={<BellOutlined />} 
            size="large"
          />
        </Badge>
        
        <Dropdown
          menu={{ items: userMenuItems }}
          placement="bottomRight"
          arrow
        >
          <Space style={{ cursor: 'pointer' }}>
            <Avatar 
              size="small" 
              icon={<UserOutlined />} 
              style={{ backgroundColor: '#1890ff' }}
            />
            <span style={{ fontWeight: 500 }}>
              {user?.username || 'User'}
            </span>
          </Space>
        </Dropdown>
      </Space>
    </AntHeader>
  );
};

export default Header;
