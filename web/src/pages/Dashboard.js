import React, { useState, useEffect } from 'react';
import { Row, Col, Card, Statistic, Table, Tag, Button, Space, Alert } from 'antd';
import { 
  ServerOutlined, 
  NodeIndexOutlined, 
  SecurityScanOutlined,
  EyeOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { useAPI } from '../contexts/APIContext';
import { useNavigate } from 'react-router-dom';

const Dashboard = ({ socket }) => {
  const { endpoints, nodes, system } = useAPI();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [systemStatus, setSystemStatus] = useState(null);
  const [systemStats, setSystemStats] = useState(null);
  const [endpointsList, setEndpointsList] = useState([]);
  const [nodesList, setNodesList] = useState([]);
  const [metricsData, setMetricsData] = useState([]);

  useEffect(() => {
    loadDashboardData();
    
    // Set up real-time updates
    if (socket) {
      socket.on('metrics_update', handleMetricsUpdate);
      socket.on('endpoint_update', handleEndpointUpdate);
      socket.on('node_status_update', handleNodeUpdate);
    }

    return () => {
      if (socket) {
        socket.off('metrics_update', handleMetricsUpdate);
        socket.off('endpoint_update', handleEndpointUpdate);
        socket.off('node_status_update', handleNodeUpdate);
      }
    };
  }, [socket]);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      
      const [statusRes, statsRes, endpointsRes, nodesRes] = await Promise.all([
        system.getStatus(),
        system.getStats(),
        endpoints.list(),
        nodes.list()
      ]);

      setSystemStatus(statusRes.data);
      setSystemStats(statsRes.data);
      setEndpointsList(endpointsRes.data.endpoints || []);
      setNodesList(nodesRes.data.nodes || []);

      // Load metrics for the first few endpoints
      if (endpointsRes.data.endpoints?.length > 0) {
        loadMetricsData(endpointsRes.data.endpoints.slice(0, 3));
      }
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadMetricsData = async (endpointsToLoad) => {
    try {
      const metricsPromises = endpointsToLoad.map(endpoint => 
        endpoints.getMetrics(endpoint.id, '24h')
      );
      
      const metricsResponses = await Promise.all(metricsPromises);
      const allMetrics = metricsResponses.flatMap(res => res.data.metrics || []);
      
      // Process metrics data for charts
      const processedMetrics = processMetricsData(allMetrics);
      setMetricsData(processedMetrics);
    } catch (error) {
      console.error('Failed to load metrics data:', error);
    }
  };

  const processMetricsData = (metrics) => {
    // Group metrics by hour and aggregate
    const hourlyData = {};
    
    metrics.forEach(metric => {
      const hour = new Date(metric.timestamp).toISOString().slice(0, 13);
      if (!hourlyData[hour]) {
        hourlyData[hour] = {
          hour: hour,
          allowedPackets: 0,
          blockedPackets: 0,
          totalPackets: 0
        };
      }
      
      hourlyData[hour].allowedPackets += metric.allowed_packets;
      hourlyData[hour].blockedPackets += metric.blocked_rate_limit + metric.blocked_blacklist + 
                                        metric.blocked_invalid_proto + metric.blocked_challenge;
      hourlyData[hour].totalPackets += metric.total_packets;
    });

    return Object.values(hourlyData).sort((a, b) => a.hour.localeCompare(b.hour));
  };

  const handleMetricsUpdate = (data) => {
    // Update metrics data in real-time
    setMetricsData(prev => {
      const newData = [...prev];
      // Add new data point
      newData.push({
        hour: new Date().toISOString().slice(0, 13),
        allowedPackets: data.allowed_packets || 0,
        blockedPackets: data.blocked_packets || 0,
        totalPackets: data.total_packets || 0
      });
      
      // Keep only last 24 hours
      return newData.slice(-24);
    });
  };

  const handleEndpointUpdate = (data) => {
    // Update endpoints list
    setEndpointsList(prev => 
      prev.map(endpoint => 
        endpoint.id === data.id ? { ...endpoint, ...data } : endpoint
      )
    );
  };

  const handleNodeUpdate = (data) => {
    // Update nodes list
    setNodesList(prev => 
      prev.map(node => 
        node.id === data.id ? { ...node, ...data } : node
      )
    );
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'active': return 'success';
      case 'inactive': return 'error';
      case 'maintenance': return 'warning';
      default: return 'default';
    }
  };

  const getProtocolColor = (protocol) => {
    switch (protocol) {
      case 'java': return 'blue';
      case 'bedrock': return 'green';
      default: return 'default';
    }
  };

  const endpointColumns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Button 
          type="link" 
          onClick={() => navigate(`/endpoints/${record.id}`)}
          style={{ padding: 0 }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: 'Frontend',
      key: 'frontend',
      render: (_, record) => `${record.front_ip}:${record.front_port}`,
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
      title: 'Status',
      dataIndex: 'active',
      key: 'status',
      render: (active, record) => (
        <Space>
          <Tag color={active ? 'success' : 'error'}>
            {active ? 'Active' : 'Inactive'}
          </Tag>
          {record.maintenance_mode && (
            <Tag color="warning">Maintenance</Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Rate Limit',
      dataIndex: 'rate_limit',
      key: 'rate_limit',
      render: (rate) => `${rate}/s`,
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
          />
          <Button 
            type="text" 
            icon={record.active ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
          />
        </Space>
      ),
    },
  ];

  const nodeColumns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'IP Address',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getStatusColor(status)}>
          {status.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'CPU Usage',
      dataIndex: 'cpu_usage',
      key: 'cpu_usage',
      render: (usage) => `${usage.toFixed(1)}%`,
    },
    {
      title: 'Memory Usage',
      dataIndex: 'memory_usage',
      key: 'memory_usage',
      render: (usage) => `${usage.toFixed(1)}%`,
    },
    {
      title: 'Packet Rate',
      dataIndex: 'packet_rate',
      key: 'packet_rate',
      render: (rate) => `${rate.toLocaleString()}/s`,
    },
  ];

  if (loading) {
    return <div>Loading dashboard...</div>;
  }

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <h1>Dashboard</h1>
        <p>Monitor your Minecraft server protection in real-time</p>
      </div>

      {systemStatus && (
        <Alert
          message={`System Status: ${systemStatus.status.toUpperCase()}`}
          description={`${systemStatus.active_nodes} active nodes, ${systemStatus.connections} active connections`}
          type={systemStatus.status === 'healthy' ? 'success' : 'warning'}
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}

      {/* Statistics Cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Protected Endpoints"
              value={endpointsList.length}
              prefix={<ServerOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Active Nodes"
              value={nodesList.filter(n => n.status === 'active').length}
              prefix={<NodeIndexOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Active Connections"
              value={systemStatus?.connections || 0}
              prefix={<SecurityScanOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Total Packet Rate"
              value={systemStats?.performance?.total_packet_rate || 0}
              suffix="/s"
              valueStyle={{ color: '#eb2f96' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Charts */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card title="Packet Flow (Last 24 Hours)" extra={
            <Button 
              type="text" 
              icon={<ReloadOutlined />} 
              onClick={() => loadDashboardData()}
            />
          }>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={metricsData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="hour" />
                <YAxis />
                <Tooltip />
                <Line 
                  type="monotone" 
                  dataKey="allowedPackets" 
                  stroke="#52c41a" 
                  name="Allowed Packets"
                />
                <Line 
                  type="monotone" 
                  dataKey="blockedPackets" 
                  stroke="#ff4d4f" 
                  name="Blocked Packets"
                />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Node Performance">
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={nodesList}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="cpu_usage" fill="#1890ff" name="CPU Usage %" />
                <Bar dataKey="memory_usage" fill="#52c41a" name="Memory Usage %" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* Tables */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card 
            title="Recent Endpoints" 
            extra={
              <Button type="link" onClick={() => navigate('/endpoints')}>
                View All
              </Button>
            }
          >
            <Table
              columns={endpointColumns}
              dataSource={endpointsList.slice(0, 5)}
              pagination={false}
              size="small"
              rowKey="id"
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card 
            title="Node Status" 
            extra={
              <Button type="link" onClick={() => navigate('/nodes')}>
                View All
              </Button>
            }
          >
            <Table
              columns={nodeColumns}
              dataSource={nodesList}
              pagination={false}
              size="small"
              rowKey="id"
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
