import React, { useState, useEffect } from 'react';
import { 
  Table, 
  Button, 
  Space, 
  Tag, 
  Modal, 
  Form, 
  Input, 
  Select, 
  InputNumber, 
  Switch, 
  message, 
  Popconfirm,
  Card,
  Row,
  Col,
  Statistic
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined, 
  EyeOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { useAPI } from '../contexts/APIContext';
import { useNavigate } from 'react-router-dom';

const { Option } = Select;

const Endpoints = ({ socket }) => {
  const { endpoints } = useAPI();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [endpointsList, setEndpointsList] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingEndpoint, setEditingEndpoint] = useState(null);
  const [form] = Form.useForm();

  useEffect(() => {
    loadEndpoints();
    
    if (socket) {
      socket.on('endpoint_update', handleEndpointUpdate);
    }

    return () => {
      if (socket) {
        socket.off('endpoint_update', handleEndpointUpdate);
      }
    };
  }, [socket]);

  const loadEndpoints = async () => {
    try {
      setLoading(true);
      const response = await endpoints.list();
      setEndpointsList(response.data.endpoints || []);
    } catch (error) {
      message.error('Failed to load endpoints');
      console.error('Error loading endpoints:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleEndpointUpdate = (data) => {
    setEndpointsList(prev => 
      prev.map(endpoint => 
        endpoint.id === data.id ? { ...endpoint, ...data } : endpoint
      )
    );
  };

  const handleCreate = () => {
    setEditingEndpoint(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (endpoint) => {
    setEditingEndpoint(endpoint);
    form.setFieldsValue({
      name: endpoint.name,
      rate_limit: endpoint.rate_limit,
      burst_limit: endpoint.burst_limit,
      maintenance_mode: endpoint.maintenance_mode,
      active: endpoint.active,
    });
    setModalVisible(true);
  };

  const handleDelete = async (id) => {
    try {
      await endpoints.delete(id);
      message.success('Endpoint deleted successfully');
      loadEndpoints();
    } catch (error) {
      message.error('Failed to delete endpoint');
      console.error('Error deleting endpoint:', error);
    }
  };

  const handleSubmit = async (values) => {
    try {
      if (editingEndpoint) {
        await endpoints.update(editingEndpoint.id, values);
        message.success('Endpoint updated successfully');
      } else {
        // For demo purposes, we'll create a mock endpoint
        const newEndpoint = {
          id: `endpoint-${Date.now()}`,
          name: values.name,
          front_ip: '198.51.100.10',
          front_port: 25565 + endpointsList.length,
          origin_ip: '203.0.113.5',
          origin_port: 25565,
          protocol: 'java',
          rate_limit: values.rate_limit,
          burst_limit: values.burst_limit,
          maintenance_mode: values.maintenance_mode || false,
          active: values.active !== false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        };
        
        setEndpointsList(prev => [newEndpoint, ...prev]);
        message.success('Endpoint created successfully');
      }
      
      setModalVisible(false);
      form.resetFields();
    } catch (error) {
      message.error(editingEndpoint ? 'Failed to update endpoint' : 'Failed to create endpoint');
      console.error('Error saving endpoint:', error);
    }
  };

  const handleToggleStatus = async (endpoint) => {
    try {
      await endpoints.update(endpoint.id, { active: !endpoint.active });
      message.success(`Endpoint ${endpoint.active ? 'deactivated' : 'activated'} successfully`);
      loadEndpoints();
    } catch (error) {
      message.error('Failed to update endpoint status');
      console.error('Error updating endpoint status:', error);
    }
  };

  const getStatusColor = (active, maintenance) => {
    if (maintenance) return 'warning';
    return active ? 'success' : 'error';
  };

  const getProtocolColor = (protocol) => {
    switch (protocol) {
      case 'java': return 'blue';
      case 'bedrock': return 'green';
      default: return 'default';
    }
  };

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Button 
          type="link" 
          onClick={() => navigate(`/endpoints/${record.id}`)}
          style={{ padding: 0, fontWeight: 500 }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: 'Frontend',
      key: 'frontend',
      render: (_, record) => (
        <div>
          <div style={{ fontWeight: 500 }}>{record.front_ip}:{record.front_port}</div>
          <div style={{ fontSize: '12px', color: '#666' }}>
            → {record.origin_ip}:{record.origin_port}
          </div>
        </div>
      ),
    },
    {
      title: 'Protocol',
      dataIndex: 'protocol',
      key: 'protocol',
      render: (protocol) => (
        <Tag color={getProtocolColor(protocol)}>
          {protocol.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Rate Limit',
      key: 'rate_limit',
      render: (_, record) => (
        <div>
          <div>{record.rate_limit}/s</div>
          <div style={{ fontSize: '12px', color: '#666' }}>
            Burst: {record.burst_limit}
          </div>
        </div>
      ),
    },
    {
      title: 'Status',
      key: 'status',
      render: (_, record) => (
        <Space direction="vertical" size="small">
          <Tag color={getStatusColor(record.active, record.maintenance_mode)}>
            {record.maintenance_mode ? 'Maintenance' : (record.active ? 'Active' : 'Inactive')}
          </Tag>
          {record.active && !record.maintenance_mode && (
            <div style={{ fontSize: '12px', color: '#52c41a' }}>
              <span className="realtime-indicator">Live</span>
            </div>
          )}
        </Space>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button 
            type="text" 
            icon={<EyeOutlined />} 
            onClick={() => navigate(`/endpoints/${record.id}`)}
            title="View Details"
          />
          <Button 
            type="text" 
            icon={<EditOutlined />} 
            onClick={() => handleEdit(record)}
            title="Edit"
          />
          <Button 
            type="text" 
            icon={record.active ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
            onClick={() => handleToggleStatus(record)}
            title={record.active ? 'Deactivate' : 'Activate'}
          />
          <Popconfirm
            title="Are you sure you want to delete this endpoint?"
            onConfirm={() => handleDelete(record.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button 
              type="text" 
              icon={<DeleteOutlined />} 
              danger
              title="Delete"
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const activeEndpoints = endpointsList.filter(e => e.active && !e.maintenance_mode);
  const maintenanceEndpoints = endpointsList.filter(e => e.maintenance_mode);
  const inactiveEndpoints = endpointsList.filter(e => !e.active);

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1>Protected Endpoints</h1>
          <p>Manage your Minecraft server protection endpoints</p>
        </div>
        <Space>
          <Button 
            icon={<ReloadOutlined />} 
            onClick={loadEndpoints}
            loading={loading}
          >
            Refresh
          </Button>
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={handleCreate}
          >
            Add Endpoint
          </Button>
        </Space>
      </div>

      {/* Statistics */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Active Endpoints"
              value={activeEndpoints.length}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Maintenance Mode"
              value={maintenanceEndpoints.length}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="Inactive Endpoints"
              value={inactiveEndpoints.length}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Endpoints Table */}
      <Card>
        <Table
          columns={columns}
          dataSource={endpointsList}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => 
              `${range[0]}-${range[1]} of ${total} endpoints`,
          }}
        />
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingEndpoint ? 'Edit Endpoint' : 'Create New Endpoint'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            rate_limit: 1000,
            burst_limit: 5000,
            maintenance_mode: false,
            active: true,
          }}
        >
          <Form.Item
            name="name"
            label="Endpoint Name"
            rules={[{ required: true, message: 'Please enter endpoint name' }]}
          >
            <Input placeholder="My Minecraft Server" />
          </Form.Item>

          <Form.Item
            name="origin_ip"
            label="Origin IP Address"
            rules={[
              { required: true, message: 'Please enter origin IP' },
              { pattern: /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/, message: 'Please enter valid IP address' }
            ]}
          >
            <Input placeholder="203.0.113.5" disabled={!!editingEndpoint} />
          </Form.Item>

          <Form.Item
            name="origin_port"
            label="Origin Port"
            rules={[{ required: true, message: 'Please enter origin port' }]}
          >
            <InputNumber 
              min={1} 
              max={65535} 
              placeholder="25565" 
              style={{ width: '100%' }}
              disabled={!!editingEndpoint}
            />
          </Form.Item>

          <Form.Item
            name="protocol"
            label="Protocol"
            rules={[{ required: true, message: 'Please select protocol' }]}
          >
            <Select placeholder="Select protocol" disabled={!!editingEndpoint}>
              <Option value="java">Java Edition</Option>
              <Option value="bedrock">Bedrock Edition</Option>
            </Select>
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="rate_limit"
                label="Rate Limit (packets/sec)"
                rules={[{ required: true, message: 'Please enter rate limit' }]}
              >
                <InputNumber 
                  min={1} 
                  max={10000} 
                  placeholder="1000" 
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="burst_limit"
                label="Burst Limit"
                rules={[{ required: true, message: 'Please enter burst limit' }]}
              >
                <InputNumber 
                  min={1} 
                  max={50000} 
                  placeholder="5000" 
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="maintenance_mode"
                label="Maintenance Mode"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="active"
                label="Active"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalVisible(false)}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit">
                {editingEndpoint ? 'Update' : 'Create'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Endpoints;
