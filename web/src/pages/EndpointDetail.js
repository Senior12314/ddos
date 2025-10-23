import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Row, Col, Statistic, Table, Tag, Button, Space, Alert, Tabs } from 'antd';
import { ArrowLeftOutlined, EditOutlined, PlayCircleOutlined, PauseCircleOutlined } from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { useAPI } from '../contexts/APIContext';

const EndpointDetail = ({ socket }) => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { endpoints } = useAPI();
  const [endpoint, setEndpoint] = useState(null);
  const [metrics, setMetrics] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadEndpointData();
    
    if (socket) {
      socket.on('metrics_update', handleMetricsUpdate);
    }

    return () => {
      if (socket) {
        socket.off('metrics_update', handleMetricsUpdate);
      }
    };
  }, [id, socket]);

  const loadEndpointData = async () => {
    try {
      setLoading(true);
      const [endpointRes, metricsRes] = await Promise.all([
        endpoints.get(id),
        endpoints.getMetrics(id, '24h')
      ]);
      
      setEndpoint(endpointRes.data);
      setMetrics(metricsRes.data.metrics || []);
    } catch (error) {
      console.error('Failed to load endpoint data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleMetricsUpdate = (data) => {
    if (data.endpoint_id === id) {
      setMetrics(prev => [...prev, data]);
    }
  };

  if (loading) {
    return <div>Loading endpoint details...</div>;
  }

  if (!endpoint) {
    return <div>Endpoint not found</div>;
  }

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

  const metricsColumns = [
    {
      title: 'Timestamp',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (date) => new Date(date).toLocaleString(),
    },
    {
      title: 'Allowed Packets',
      dataIndex: 'allowed_packets',
      key: 'allowed_packets',
      render: (value) => value.toLocaleString(),
    },
    {
      title: 'Blocked Packets',
      key: 'blocked_packets',
      render: (_, record) => (
        record.blocked_rate_limit + record.blocked_blacklist + 
        record.blocked_invalid_proto + record.blocked_challenge
      ).toLocaleString(),
    },
    {
      title: 'Total Packets',
      dataIndex: 'total_packets',
      key: 'total_packets',
      render: (value) => value.toLocaleString(),
    },
  ];

  const tabItems = [
    {
      key: 'overview',
      label: 'Overview',
      children: (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={8}>
            <Card>
              <Statistic
                title="Allowed Packets"
                value={metrics.reduce((sum, m) => sum + m.allowed_packets, 0)}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card>
              <Statistic
                title="Blocked Packets"
                value={metrics.reduce((sum, m) => sum + m.blocked_rate_limit + m.blocked_blacklist + m.blocked_invalid_proto + m.blocked_challenge, 0)}
                valueStyle={{ color: '#ff4d4f' }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card>
              <Statistic
                title="Total Packets"
                value={metrics.reduce((sum, m) => sum + m.total_packets, 0)}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
        </Row>
      ),
    },
    {
      key: 'metrics',
      label: 'Metrics',
      children: (
        <div>
          <Card title="Packet Flow (Last 24 Hours)" style={{ marginBottom: 16 }}>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={metrics}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="timestamp" />
                <YAxis />
                <Tooltip />
                <Line type="monotone" dataKey="allowed_packets" stroke="#52c41a" name="Allowed Packets" />
                <Line type="monotone" dataKey="blocked_rate_limit" stroke="#ff4d4f" name="Blocked Packets" />
              </LineChart>
            </ResponsiveContainer>
          </Card>
          
          <Card title="Recent Metrics">
            <Table
              columns={metricsColumns}
              dataSource={metrics.slice(-10)}
              pagination={false}
              size="small"
              rowKey="timestamp"
            />
          </Card>
        </div>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Button 
          icon={<ArrowLeftOutlined />} 
          onClick={() => navigate('/endpoints')}
          style={{ marginBottom: 16 }}
        >
          Back to Endpoints
        </Button>
        
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h1>{endpoint.name}</h1>
            <p>Endpoint ID: {endpoint.id}</p>
          </div>
          <Space>
            <Button 
              icon={<EditOutlined />} 
              onClick={() => navigate(`/endpoints/${id}/edit`)}
            >
              Edit
            </Button>
            <Button 
              icon={endpoint.active ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              type={endpoint.active ? 'default' : 'primary'}
            >
              {endpoint.active ? 'Deactivate' : 'Activate'}
            </Button>
          </Space>
        </div>
      </div>

      <Alert
        message={`Status: ${endpoint.maintenance_mode ? 'Maintenance Mode' : (endpoint.active ? 'Active' : 'Inactive')}`}
        type={endpoint.maintenance_mode ? 'warning' : (endpoint.active ? 'success' : 'error')}
        showIcon
        style={{ marginBottom: 24 }}
      />

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={6}>
          <Card>
            <Statistic
              title="Frontend Address"
              value={`${endpoint.front_ip}:${endpoint.front_port}`}
              valueStyle={{ fontSize: 16 }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={6}>
          <Card>
            <Statistic
              title="Origin Address"
              value={`${endpoint.origin_ip}:${endpoint.origin_port}`}
              valueStyle={{ fontSize: 16 }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={6}>
          <Card>
            <Statistic
              title="Protocol"
              value={<Tag color={getProtocolColor(endpoint.protocol)}>{endpoint.protocol.toUpperCase()}</Tag>}
            />
          </Card>
        </Col>
        <Col xs={24} lg={6}>
          <Card>
            <Statistic
              title="Rate Limit"
              value={`${endpoint.rate_limit}/s`}
              suffix={`Burst: ${endpoint.burst_limit}`}
            />
          </Card>
        </Col>
      </Row>

      <Tabs items={tabItems} />
    </div>
  );
};

export default EndpointDetail;
