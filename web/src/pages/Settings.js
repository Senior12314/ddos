import React from 'react';
import { Card, Form, Input, Button, Switch, Divider, Typography, Space } from 'antd';
import { SaveOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

const Settings = () => {
  const [form] = Form.useForm();

  const handleSubmit = (values) => {
    console.log('Settings updated:', values);
  };

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <h1>Settings</h1>
        <p>Configure your account and system preferences</p>
      </div>

      <Card title="Profile Settings" style={{ marginBottom: 24 }}>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            username: 'admin',
            email: 'admin@cloudnordsp.com',
            notifications: true,
            maintenance_alerts: true,
          }}
        >
          <Form.Item
            name="username"
            label="Username"
            rules={[{ required: true, message: 'Please enter username' }]}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name="email"
            label="Email"
            rules={[
              { required: true, message: 'Please enter email' },
              { type: 'email', message: 'Please enter valid email' }
            ]}
          >
            <Input />
          </Form.Item>

          <Divider />

          <Title level={4}>Notifications</Title>
          
          <Form.Item
            name="notifications"
            label="Enable Notifications"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name="maintenance_alerts"
            label="Maintenance Alerts"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
              Save Settings
            </Button>
          </Form.Item>
        </Form>
      </Card>

      <Card title="System Information">
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          <div>
            <Text strong>Version:</Text> 1.0.0
          </div>
          <div>
            <Text strong>Build:</Text> 2024-01-01
          </div>
          <div>
            <Text strong>API Version:</Text> v1
          </div>
          <div>
            <Text strong>Status:</Text> <Text type="success">Operational</Text>
          </div>
        </Space>
      </Card>
    </div>
  );
};

export default Settings;
