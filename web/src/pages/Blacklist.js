import React, { useState, useEffect } from 'react';
import { Table, Card, Tag, Button, Space, Modal, Form, Input, InputNumber, message, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { useAPI } from '../contexts/APIContext';

const Blacklist = () => {
  const { blacklist } = useAPI();
  const [loading, setLoading] = useState(true);
  const [blacklistData, setBlacklistData] = useState([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    loadBlacklist();
  }, []);

  const loadBlacklist = async () => {
    try {
      setLoading(true);
      const response = await blacklist.list();
      setBlacklistData(response.data.blacklist || []);
    } catch (error) {
      message.error('Failed to load blacklist');
      console.error('Error loading blacklist:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = async (values) => {
    try {
      await blacklist.add(values);
      message.success('IP added to blacklist');
      setModalVisible(false);
      form.resetFields();
      loadBlacklist();
    } catch (error) {
      message.error('Failed to add IP to blacklist');
      console.error('Error adding to blacklist:', error);
    }
  };

  const handleRemove = async (ip) => {
    try {
      await blacklist.remove(ip);
      message.success('IP removed from blacklist');
      loadBlacklist();
    } catch (error) {
      message.error('Failed to remove IP from blacklist');
      console.error('Error removing from blacklist:', error);
    }
  };

  const columns = [
    {
      title: 'IP Address',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: 'Reason',
      dataIndex: 'reason',
      key: 'reason',
    },
    {
      title: 'Duration',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration) => `${duration} seconds`,
    },
    {
      title: 'Expires At',
      dataIndex: 'expires_at',
      key: 'expires_at',
      render: (date) => new Date(date).toLocaleString(),
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date) => new Date(date).toLocaleString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record) => (
        <Popconfirm
          title="Are you sure you want to remove this IP from the blacklist?"
          onConfirm={() => handleRemove(record.ip)}
          okText="Yes"
          cancelText="No"
        >
          <Button type="text" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1>IP Blacklist</h1>
          <p>Manage globally blocked IP addresses</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadBlacklist} loading={loading}>
            Refresh
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            Add IP
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={blacklistData}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => 
              `${range[0]}-${range[1]} of ${total} blocked IPs`,
          }}
        />
      </Card>

      <Modal
        title="Add IP to Blacklist"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleAdd}
        >
          <Form.Item
            name="ip"
            label="IP Address"
            rules={[
              { required: true, message: 'Please enter IP address' },
              { pattern: /^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/, message: 'Please enter valid IP address' }
            ]}
          >
            <Input placeholder="192.168.1.1" />
          </Form.Item>

          <Form.Item
            name="reason"
            label="Reason"
            rules={[{ required: true, message: 'Please enter reason' }]}
          >
            <Input placeholder="DDoS attack" />
          </Form.Item>

          <Form.Item
            name="duration"
            label="Duration (seconds)"
            rules={[{ required: true, message: 'Please enter duration' }]}
          >
            <InputNumber min={1} max={86400} placeholder="3600" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalVisible(false)}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit">
                Add to Blacklist
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Blacklist;
