import React, { useState, useEffect } from 'react';
import { Table, Card, Tag, Button, Space, Statistic, Row, Col } from 'antd';
import { ReloadOutlined, EyeOutlined } from '@ant-design/icons';
import { useAPI } from '../contexts/APIContext';

const Nodes = ({ socket }) => {
  const { nodes } = useAPI();
  const [loading, setLoading] = useState(true);
  const [nodesList, setNodesList] = useState([]);

  useEffect(() => {
    loadNodes();
    
    if (socket) {
      socket.on('node_status_update', handleNodeUpdate);
    }

    return () => {
      if (socket) {
        socket.off('node_status_update', handleNodeUpdate);
      }
    };
  }, [socket]);

  const loadNodes = async () => {
    try {
      setLoading(true);
      const response = await nodes.list();
      setNodesList(response.data.nodes || []);
    } catch (error) {
      console.error('Error loading nodes:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleNodeUpdate = (data) => {
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

  const columns = [
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
      title: 'Port',
      dataIndex: 'port',
      key: 'port',
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
    {
      title: 'Last Seen',
      dataIndex: 'last_seen',
      key: 'last_seen',
      render: (date) => new Date(date).toLocaleString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button type="text" icon={<EyeOutlined />} />
        </Space>
      ),
    },
  ];

  const activeNodes = nodesList.filter(n => n.status === 'active');
  const totalNodes = nodesList.length;

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1>Edge Nodes</h1>
          <p>Monitor your edge nodes running XDP programs</p>
        </div>
        <Button icon={<ReloadOutlined />} onClick={loadNodes} loading={loading}>
          Refresh
        </Button>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12}>
          <Card>
            <Statistic
              title="Active Nodes"
              value={activeNodes.length}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card>
            <Statistic
              title="Total Nodes"
              value={totalNodes}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Table
          columns={columns}
          dataSource={nodesList}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => 
              `${range[0]}-${range[1]} of ${total} nodes`,
          }}
        />
      </Card>
    </div>
  );
};

export default Nodes;
