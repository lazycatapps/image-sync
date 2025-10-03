// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Form, Input, InputNumber, Button, Card, Space, Typography, Select, Tag, Checkbox, Modal, Alert, Collapse, FloatButton, App as AntApp } from 'antd';
import { InfoCircleOutlined, BugOutlined, CopyOutlined, FullscreenOutlined, FullscreenExitOutlined } from '@ant-design/icons';
import 'antd/dist/reset.css';
import './App.css';

const { Title, Text } = Typography;
const APP_VERSION = process.env.REACT_APP_VERSION || require('../package.json').version;
const GIT_COMMIT = process.env.REACT_APP_GIT_COMMIT || 'dev';
const GIT_COMMIT_FULL = process.env.REACT_APP_GIT_COMMIT_FULL || 'development';
const GIT_BRANCH = process.env.REACT_APP_GIT_BRANCH || 'local';
const BUILD_TIME = process.env.REACT_APP_BUILD_TIME || 'dev-build';

// 在 LPK 部署时使用相对路径，由 routes 自动代理到后端
// 在开发环境通过 env-config.js 配置完整地址
const BACKEND_API_URL = window._env_?.BACKEND_API_URL || '';

// Debug logger utility
const debugLog = (category, ...args) => {
  const timestamp = new Date().toISOString();
  const message = `[${timestamp}] [${category}]`;
  console.log(message, ...args);
  return { timestamp, category, message: args.map(arg =>
    typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
  ).join(' ') };
};

function AppContent() {
  const { message } = AntApp.useApp();
  const [loading, setLoading] = useState(false);
  const [queryingArch, setQueryingArch] = useState(false);
  const [architectures, setArchitectures] = useState([]);
  const [syncLogs, setSyncLogs] = useState([]);
  const [syncStatus, setSyncStatus] = useState(null);
  const [logsModalVisible, setLogsModalVisible] = useState(false);
  const [inspectModalVisible, setInspectModalVisible] = useState(false);
  const [inspectLogs, setInspectLogs] = useState([]);
  const [inspectStatus, setInspectStatus] = useState('querying');
  const [debugLogs, setDebugLogs] = useState([]);
  const [debugModalVisible, setDebugModalVisible] = useState(false);
  const [authChecking, setAuthChecking] = useState(true);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [userInfo, setUserInfo] = useState(null);
  const [configList, setConfigList] = useState([]);
  const [selectedConfig, setSelectedConfig] = useState('');
  const [saveConfigModalVisible, setSaveConfigModalVisible] = useState(false);
  const [deleteConfigModalVisible, setDeleteConfigModalVisible] = useState(false);
  const [configNameInput, setConfigNameInput] = useState('');
  const [debugEnabled, setDebugEnabled] = useState(false);
  const [logsModalMaximized, setLogsModalMaximized] = useState(false);
  const [form] = Form.useForm();
  const eventSourceRef = useRef(null);
  const statusIntervalRef = useRef(null);
  const logsEndRef = useRef(null);
  const inspectLogsEndRef = useRef(null);
  const debugLogsEndRef = useRef(null);

  // Enhanced debug logging function
  const addDebugLog = (category, ...args) => {
    const log = debugLog(category, ...args);
    setDebugLogs(prev => [...prev, log]);
  };

  // Global error handler
  useEffect(() => {
    const handleError = (event) => {
      addDebugLog('ERROR', 'Global error:', event.error?.message || event.message);
    };
    const handleUnhandledRejection = (event) => {
      addDebugLog('ERROR', 'Unhandled rejection:', event.reason);
    };

    window.addEventListener('error', handleError);
    window.addEventListener('unhandledrejection', handleUnhandledRejection);

    addDebugLog('INIT', 'App initialized', {
      version: APP_VERSION,
      gitCommit: GIT_COMMIT,
      gitBranch: GIT_BRANCH,
      buildTime: BUILD_TIME,
      userAgent: navigator.userAgent,
      backendUrl: BACKEND_API_URL
    });

    return () => {
      window.removeEventListener('error', handleError);
      window.removeEventListener('unhandledrejection', handleUnhandledRejection);
    };
  }, []);

  // Check authentication status on mount
  useEffect(() => {
    addDebugLog('AUTH', 'Checking authentication status');
    fetch(`${BACKEND_API_URL}/api/v1/auth/userinfo`, {
      credentials: 'include'
    })
      .then(res => res.json())
      .then(data => {
        addDebugLog('AUTH', 'User info:', data);
        setOidcEnabled(data.oidc_enabled || false);
        if (data.authenticated) {
          setIsAuthenticated(true);
          setUserInfo(data);
          addDebugLog('AUTH', 'User is authenticated');
        } else {
          setIsAuthenticated(false);
          addDebugLog('AUTH', `User is not authenticated (OIDC enabled: ${data.oidc_enabled})`);
        }
        setAuthChecking(false);
      })
      .catch(err => {
        addDebugLog('ERROR', 'Failed to check auth:', err);
        setIsAuthenticated(false);
        setOidcEnabled(false);
        setAuthChecking(false);
      });
  }, []);

  // Load config list
  const loadConfigList = useCallback(async () => {
    try {
      addDebugLog('CONFIG', 'Loading config list');
      const response = await fetch(`${BACKEND_API_URL}/api/v1/configs`, {
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        setConfigList(data.configs || []);
        addDebugLog('CONFIG', 'Config list loaded:', data.configs);
      } else {
        const error = await response.json();
        addDebugLog('ERROR', 'Failed to load config list:', error);
      }
    } catch (error) {
      addDebugLog('ERROR', 'Load config list exception:', error.message);
    }
  }, []);

  // Load config by name
  const loadConfigByName = useCallback(async (name, showSuccessMessage = true) => {
    try {
      addDebugLog('CONFIG', 'Loading config:', name);
      const response = await fetch(`${BACKEND_API_URL}/api/v1/config/${encodeURIComponent(name)}`, {
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        addDebugLog('CONFIG', 'Config loaded (passwords masked):', { ...data, sourcePassword: data.sourcePassword ? '***' : '', destPassword: data.destPassword ? '***' : '' });

        // Always reset all fields first to clear previous values
        const formValues = {
          sourceImage: '',
          destImage: '',
          sourceUsername: '',
          destUsername: '',
          sourcePassword: '',
          destPassword: '',
          srcTlsVerify: true,
          destTlsVerify: true,
          retryTimes: 3,
        };

        if (data) {
          if (data.sourceRegistry) formValues.sourceImage = data.sourceRegistry;
          if (data.destRegistry) formValues.destImage = data.destRegistry;

          // Decode base64 encoded credentials received from backend
          try {
            if (data.sourceUsername) formValues.sourceUsername = atob(data.sourceUsername);
            if (data.destUsername) formValues.destUsername = atob(data.destUsername);
            if (data.sourcePassword) formValues.sourcePassword = atob(data.sourcePassword);
            if (data.destPassword) formValues.destPassword = atob(data.destPassword);
          } catch (err) {
            addDebugLog('ERROR', 'Failed to decode base64 credentials:', err);
          }

          if (data.srcTLSVerify !== undefined) formValues.srcTlsVerify = data.srcTLSVerify;
          if (data.destTLSVerify !== undefined) formValues.destTlsVerify = data.destTLSVerify;
          if (data.retryTimes !== undefined) formValues.retryTimes = data.retryTimes;
        }

        form.setFieldsValue(formValues);

        if (showSuccessMessage) {
          message.success(`已加载配置: ${name}`, 3);
        }
      } else {
        const error = await response.json();
        message.error(`加载配置失败: ${error.error || '未知错误'}`, 5);
        addDebugLog('ERROR', 'Failed to load config:', error);
      }
    } catch (error) {
      message.error(`加载配置失败: ${error.message}`, 5);
      addDebugLog('ERROR', 'Load config exception:', error.message);
    }
  }, [form, message]);

  // Load config list and last used config after auth check completes
  useEffect(() => {
    if (authChecking) {
      return;
    }
    // If OIDC is enabled and user is not authenticated, don't load config
    if (oidcEnabled && !isAuthenticated) {
      return;
    }

    // Load config list
    loadConfigList();

    // First load env defaults
    addDebugLog('FETCH', 'Fetching env defaults from:', `${BACKEND_API_URL}/api/v1/env/defaults`);
    fetch(`${BACKEND_API_URL}/api/v1/env/defaults`, {
      credentials: 'include'
    })
      .then(res => {
        addDebugLog('FETCH', 'Env defaults response status:', res.status);
        return res.json();
      })
      .then(data => {
        addDebugLog('FETCH', 'Env defaults data:', data);
        if (data.sourceRegistry || data.destRegistry) {
          // Preserve existing field values when setting env defaults
          const currentValues = form.getFieldsValue();
          form.setFieldsValue({
            ...currentValues,
            sourceImage: data.sourceRegistry || '',
            destImage: data.destRegistry || '',
          });
          addDebugLog('FETCH', 'Form fields updated with env defaults');
        }
      })
      .catch(err => {
        const errMsg = `Failed to fetch environment defaults: ${err.message}`;
        console.error(errMsg, err);
        addDebugLog('ERROR', errMsg, err);
      });

    // Then load last used config name
    addDebugLog('FETCH', 'Fetching last used config from:', `${BACKEND_API_URL}/api/v1/config/last-used`);
    fetch(`${BACKEND_API_URL}/api/v1/config/last-used`, {
      credentials: 'include'
    })
      .then(res => {
        addDebugLog('FETCH', 'Last used config response status:', res.status);
        return res.json();
      })
      .then(data => {
        addDebugLog('FETCH', 'Last used config name:', data.name);
        if (data.name) {
          setSelectedConfig(data.name);
          // Load the last used config (no success message on initial load)
          loadConfigByName(data.name, false);
        }
      })
      .catch(err => {
        const errMsg = `Failed to fetch last used config: ${err.message}`;
        console.error(errMsg, err);
        addDebugLog('ERROR', errMsg, err);
      });
  }, [form, isAuthenticated, oidcEnabled, authChecking, loadConfigList, loadConfigByName]);

  useEffect(() => {
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [syncLogs]);

  useEffect(() => {
    if (inspectLogsEndRef.current) {
      inspectLogsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [inspectLogs]);

  useEffect(() => {
    if (debugLogsEndRef.current) {
      debugLogsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [debugLogs]);

  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, []);

  const queryArchitectures = async () => {
    const sourceImage = form.getFieldValue('sourceImage');
    const sourceUsername = form.getFieldValue('sourceUsername');
    const sourcePassword = form.getFieldValue('sourcePassword');
    const srcTlsVerify = form.getFieldValue('srcTlsVerify');

    addDebugLog('QUERY_ARCH', 'Starting architecture query', { sourceImage, hasUsername: !!sourceUsername, srcTlsVerify });

    if (!sourceImage) {
      message.warning('请先输入源镜像地址');
      addDebugLog('QUERY_ARCH', 'Validation failed: no source image');
      return;
    }

    // Reset state and prepare logs
    setInspectLogs([]);
    setInspectStatus('querying');
    setQueryingArch(true);

    // Add initial log
    setInspectLogs([
      `正在查询镜像架构: ${sourceImage}`,
      sourceUsername ? '使用认证凭据...' : '使用匿名访问...',
      srcTlsVerify ? '启用 TLS 验证' : '跳过 TLS 验证',
      '执行查询中...'
    ]);

    try {
      const url = `${BACKEND_API_URL}/api/v1/inspect`;
      const body = {
        image: sourceImage,
        username: sourceUsername,
        password: sourcePassword,
        tlsVerify: srcTlsVerify,
      };
      addDebugLog('QUERY_ARCH', 'Sending request to:', url, 'Body:', { ...body, password: body.password ? '***' : '' });

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(body),
      });

      addDebugLog('QUERY_ARCH', 'Response received:', { status: response.status, ok: response.ok });

      if (response.ok) {
        const data = await response.json();
        addDebugLog('QUERY_ARCH', 'Response data:', data);
        setArchitectures(data.architectures || []);
        setInspectLogs(prev => [
          ...prev,
          '',
          `✓ 查询成功！`,
          `找到 ${data.architectures.length} 个架构:`,
          ...data.architectures.map(arch => `  - ${arch}`)
        ]);
        setInspectStatus('success');
        message.success(`查询成功，找到 ${data.architectures.length} 个架构`);
        addDebugLog('QUERY_ARCH', 'Query successful');
      } else {
        const error = await response.json();
        addDebugLog('QUERY_ARCH', 'Query failed:', error);
        setInspectLogs(prev => [
          ...prev,
          '',
          `✗ 查询失败: ${error.error || '未知错误'}`
        ]);
        setInspectStatus('error');
        message.error(`查询失败: ${error.error || '未知错误'}`);
      }
    } catch (error) {
      addDebugLog('ERROR', 'Query arch exception:', error.message, error);
      setInspectLogs(prev => [
        ...prev,
        '',
        `✗ 请求失败: ${error.message}`
      ]);
      setInspectStatus('error');
      message.error(`请求失败: ${error.message}`);
    } finally {
      setQueryingArch(false);
      addDebugLog('QUERY_ARCH', 'Query completed');
    }
  };

  const startLogStream = (taskId) => {
    addDebugLog('LOG_STREAM', 'Starting log stream for task:', taskId);

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      addDebugLog('LOG_STREAM', 'Closed previous EventSource');
    }
    if (statusIntervalRef.current) {
      clearInterval(statusIntervalRef.current);
      addDebugLog('LOG_STREAM', 'Cleared previous status interval');
    }

    setSyncLogs([]);
    const url = `${BACKEND_API_URL}/api/v1/sync/${taskId}/logs`;
    addDebugLog('LOG_STREAM', 'Creating EventSource:', url);

    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      addDebugLog('LOG_STREAM', 'EventSource connection opened');
    };

    // Start polling status immediately to update status in real-time
    statusIntervalRef.current = setInterval(async () => {
      try {
        const statusUrl = `${BACKEND_API_URL}/api/v1/sync/${taskId}`;
        const response = await fetch(statusUrl, { credentials: 'include' });
        if (response.ok) {
          const data = await response.json();
          setSyncStatus(data.status);
          addDebugLog('LOG_STREAM', 'Status update:', data.status);
          if (data.status === 'completed' || data.status === 'failed') {
            addDebugLog('LOG_STREAM', 'Task finished, stopping polling');
            clearInterval(statusIntervalRef.current);
            statusIntervalRef.current = null;
            if (eventSourceRef.current) {
              eventSourceRef.current.close();
            }
          }
        }
      } catch (error) {
        console.error('Failed to fetch sync status:', error);
        addDebugLog('ERROR', 'Status polling failed:', error.message);
      }
    }, 1000);

    eventSource.onmessage = (event) => {
      addDebugLog('LOG_STREAM', 'Received log message');
      setSyncLogs(prev => [...prev, event.data]);
    };

    eventSource.onerror = (error) => {
      console.log('EventSource error:', error);
      addDebugLog('ERROR', 'EventSource error:', error);
      eventSource.close();
    };
  };

  const onFinish = async (values) => {
    addDebugLog('SYNC', 'Starting sync task with values:', { ...values, sourcePassword: '***', destPassword: '***' });
    setLoading(true);
    try {
      const url = `${BACKEND_API_URL}/api/v1/sync`;
      addDebugLog('SYNC', 'Sending sync request to:', url);

      // Ensure correct data types for backend
      const payload = {
        sourceImage: values.sourceImage || '',
        destImage: values.destImage || '',
        architecture: values.architecture || 'all',
        sourceUsername: values.sourceUsername || '',
        sourcePassword: values.sourcePassword || '',
        destUsername: values.destUsername || '',
        destPassword: values.destPassword || '',
        srcTlsVerify: values.srcTlsVerify !== undefined ? values.srcTlsVerify : true,
        destTlsVerify: values.destTlsVerify !== undefined ? values.destTlsVerify : true,
        retryTimes: typeof values.retryTimes === 'number' ? parseInt(values.retryTimes, 10) : 3,
      };

      addDebugLog('SYNC', 'Payload prepared:', { ...payload, sourcePassword: '***', destPassword: '***' });

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(payload),
      });

      addDebugLog('SYNC', 'Sync response received:', { status: response.status, ok: response.ok });

      if (response.ok) {
        const data = await response.json();
        addDebugLog('SYNC', 'Sync task created:', data);
        message.success('镜像同步任务已启动！');
        setSyncStatus('running');
        setLogsModalVisible(true);
        startLogStream(data.id);
      } else {
        const error = await response.text();
        addDebugLog('ERROR', 'Sync failed:', { status: response.status, error });
        message.error('启动同步任务失败');
      }
    } catch (error) {
      addDebugLog('ERROR', 'Sync exception:', error.message, error);
      message.error('请求失败: ' + error.message);
    } finally {
      setLoading(false);
      addDebugLog('SYNC', 'Sync request completed');
    }
  };

  const handleCloseModal = () => {
    setLogsModalVisible(false);
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }
    if (statusIntervalRef.current) {
      clearInterval(statusIntervalRef.current);
      statusIntervalRef.current = null;
    }
  };

  const handleCloseInspectModal = () => {
    setInspectModalVisible(false);
  };

  // Handle select config from dropdown
  const handleSelectConfig = async (name) => {
    if (!name) return;
    setSelectedConfig(name);
    addDebugLog('CONFIG', 'User selected config:', name);
    await loadConfigByName(name);
  };

  // Show save config modal
  const handleShowSaveConfigModal = () => {
    setConfigNameInput(selectedConfig || '');
    setSaveConfigModalVisible(true);
  };

  // Save configuration with name
  const handleSaveConfigWithName = async () => {
    const name = configNameInput.trim();
    if (!name) {
      message.error('请输入配置名称');
      return;
    }

    // Validate config name
    if (!/^[a-zA-Z0-9._-]+$/.test(name)) {
      message.error('配置名称只能包含字母、数字、点、横线和下划线');
      return;
    }

    try {
      const values = form.getFieldsValue();
      addDebugLog('CONFIG', 'Saving config:', name, { ...values, sourcePassword: '***', destPassword: '***' });

      // Encode credentials with base64 for secure transmission
      const config = {
        sourceRegistry: values.sourceImage || '',
        destRegistry: values.destImage || '',
        sourceUsername: values.sourceUsername ? btoa(values.sourceUsername) : '',
        destUsername: values.destUsername ? btoa(values.destUsername) : '',
        sourcePassword: values.sourcePassword ? btoa(values.sourcePassword) : '',
        destPassword: values.destPassword ? btoa(values.destPassword) : '',
        srcTLSVerify: values.srcTlsVerify !== undefined ? values.srcTlsVerify : true,
        destTLSVerify: values.destTlsVerify !== undefined ? values.destTlsVerify : true,
        retryTimes: typeof values.retryTimes === 'number' ? parseInt(values.retryTimes, 10) : 3,
      };

      const response = await fetch(`${BACKEND_API_URL}/api/v1/config/${encodeURIComponent(name)}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(config),
      });

      if (response.ok) {
        message.success(`配置已保存: ${name}`, 5);
        addDebugLog('CONFIG', 'Config saved successfully:', name);
        setSaveConfigModalVisible(false);
        setSelectedConfig(name);
        // Reload config list
        await loadConfigList();
      } else {
        const error = await response.json();
        message.error(`保存失败: ${error.error || '未知错误'}`, 5);
        addDebugLog('ERROR', 'Failed to save config:', error);
      }
    } catch (error) {
      message.error(`保存失败: ${error.message}`, 5);
      addDebugLog('ERROR', 'Save config exception:', error.message);
    }
  };

  // Show delete config modal
  const handleShowDeleteConfigModal = () => {
    if (!selectedConfig) {
      message.warning('请先选择要删除的配置');
      addDebugLog('CONFIG', 'Delete config failed: no config selected');
      return;
    }

    addDebugLog('CONFIG', 'Delete config button clicked, selectedConfig:', selectedConfig);
    addDebugLog('CONFIG', 'Showing delete confirmation modal for:', selectedConfig);
    setDeleteConfigModalVisible(true);
  };

  // Delete selected config
  const handleDeleteConfig = async () => {
    const configToDelete = selectedConfig;
    addDebugLog('CONFIG', 'User confirmed deletion, deleting config:', configToDelete);

    try {
      const response = await fetch(`${BACKEND_API_URL}/api/v1/config/${encodeURIComponent(configToDelete)}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      addDebugLog('CONFIG', 'Delete response status:', response.status);

      if (response.ok) {
        await response.json();
        message.success(`已删除配置: ${configToDelete}`, 5);
        addDebugLog('CONFIG', 'Config deleted successfully:', configToDelete);
        setSelectedConfig('');
        setDeleteConfigModalVisible(false);
        // Reload config list
        await loadConfigList();
      } else {
        const error = await response.json();
        message.error(`删除失败: ${error.error || '未知错误'}`, 5);
        addDebugLog('ERROR', 'Failed to delete config:', error);
      }
    } catch (error) {
      message.error(`删除失败: ${error.message}`, 5);
      addDebugLog('ERROR', 'Delete config exception:', error.message);
    }
  };

  // Handle login
  const handleLogin = () => {
    window.location.href = `${BACKEND_API_URL}/api/v1/auth/login`;
  };

  // Handle logout
  const handleLogout = async () => {
    try {
      await fetch(`${BACKEND_API_URL}/api/v1/auth/logout`, {
        method: 'POST',
        credentials: 'include'
      });
      setIsAuthenticated(false);
      setUserInfo(null);
      message.success('已退出登录');
    } catch (err) {
      message.error('退出登录失败');
    }
  };

  // Copy to clipboard
  const handleCopyToClipboard = (fieldName) => {
    const value = form.getFieldValue(fieldName);
    if (!value) {
      message.warning('输入框为空，无法复制');
      addDebugLog('COPY', `Copy failed: ${fieldName} is empty`);
      return;
    }

    navigator.clipboard.writeText(value)
      .then(() => {
        message.success('已复制到剪贴板');
        addDebugLog('COPY', `Copied ${fieldName}:`, value);
      })
      .catch(err => {
        message.error('复制失败');
        addDebugLog('ERROR', `Copy failed for ${fieldName}:`, err.message);
      });
  };

  // Copy sync logs to clipboard
  const handleCopySyncLogs = () => {
    if (syncLogs.length === 0) {
      message.warning('日志为空，无法复制');
      addDebugLog('COPY', 'Copy sync logs failed: logs are empty');
      return;
    }

    const logsText = syncLogs.join('\n');
    navigator.clipboard.writeText(logsText)
      .then(() => {
        message.success('日志已复制到剪贴板');
        addDebugLog('COPY', `Copied sync logs: ${syncLogs.length} lines`);
      })
      .catch(err => {
        message.error('复制日志失败');
        addDebugLog('ERROR', 'Copy sync logs failed:', err.message);
      });
  };

  // Show loading while checking auth
  if (authChecking) {
    return (
      <div className="App">
        <div className="container">
          <Card style={{ textAlign: 'center', padding: '50px' }}>
            <Title level={3}>正在检查登录状态...</Title>
          </Card>
        </div>
      </div>
    );
  }

  // Show login page only if OIDC is enabled and user is not authenticated
  if (oidcEnabled && !isAuthenticated) {
    return (
      <div className="App">
        <div className="container">
          <Title level={2}>Image Sync - 容器镜像同步工具</Title>
          <Card style={{ textAlign: 'center', padding: '50px' }}>
            <Title level={3}>为了确保您的配置信息安全，请登录后再使用！</Title>
            <Text type="secondary" style={{ display: 'block', marginBottom: '24px', fontSize: '16px' }}>
              登录后您可以安全地保存和管理镜像仓库配置
            </Text>
            <Button type="primary" size="large" onClick={handleLogin}>
              登录
            </Button>
          </Card>
          <div className="footer">
            <Text type="secondary">
              Image Sync · v{APP_VERSION}
              {GIT_COMMIT !== 'dev' && GIT_COMMIT_FULL !== 'development' && (
                <>
                  {' · '}
                  <a
                    href={`https://github.com/lazycatapps/image-sync/commit/${GIT_COMMIT_FULL}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    title={`Commit: ${GIT_COMMIT_FULL}`}
                  >
                    {GIT_COMMIT}
                  </a>
                </>
              )}
              {' · '}
              Copyright © {new Date().getFullYear()} Lazycat Apps
              {' · '}
              <a href="https://github.com/lazycatapps/image-sync" target="_blank" rel="noopener noreferrer">
                GitHub
              </a>
            </Text>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="App">
      <div className="container">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <Title level={2} style={{ margin: 0 }}>Image Sync - 容器镜像同步工具</Title>
          {oidcEnabled && (
            <Space>
              {userInfo && (
                <Text type="secondary">
                  {userInfo.email}
                  {userInfo.is_admin && <span style={{ color: '#1890ff', marginLeft: '8px' }}>(管理员)</span>}
                </Text>
              )}
              <Button onClick={handleLogout}>退出登录</Button>
            </Space>
          )}
        </div>
        <Card>
          <Form
            form={form}
            layout="vertical"
            onFinish={onFinish}
            autoComplete="off"
          >
            <div className="source-section">
              <Title level={4}>源镜像信息</Title>
              <Form.Item
                label="源镜像地址"
                name="sourceImage"
                rules={[{ required: true, message: '请输入源镜像地址' }]}
              >
                <Input
                  placeholder="例如: docker.io/library/nginx:latest"
                  suffix={
                    <CopyOutlined
                      onClick={() => handleCopyToClipboard('sourceImage')}
                      style={{ color: '#1890ff', cursor: 'pointer' }}
                      title="复制地址"
                    />
                  }
                />
              </Form.Item>

              <Space direction="horizontal" style={{ width: '100%', alignItems: 'flex-start', flexWrap: 'wrap' }} size="large">
                <Form.Item
                  label="源仓库用户名"
                  name="sourceUsername"
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Input placeholder="选填（私有仓库需要）" />
                </Form.Item>

                <Form.Item
                  label="源仓库密码"
                  name="sourcePassword"
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Input.Password placeholder="选填（私有仓库需要）" />
                </Form.Item>

                <Form.Item
                  label=" "
                  name="srcTlsVerify"
                  valuePropName="checked"
                  initialValue={true}
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Checkbox>启用 TLS 证书验证</Checkbox>
                </Form.Item>
              </Space>

              <div>
                <Space direction="horizontal" style={{ width: '100%', display: 'flex', alignItems: 'flex-end', flexWrap: 'wrap' }} size="middle">
                  <Form.Item
                    label="镜像架构"
                    name="architecture"
                    style={{ flex: '1 1 200px', minWidth: '0', marginBottom: 0 }}
                    initialValue="all"
                  >
                    <Select placeholder="选择镜像架构">
                      <Select.Option value="all">全部架构（推荐）</Select.Option>
                      {architectures.length === 0 ? (
                        <>
                          <Select.Option value="linux/amd64">linux/amd64</Select.Option>
                          <Select.Option value="linux/arm64">linux/arm64</Select.Option>
                          <Select.Option value="linux/arm/v7">linux/arm/v7</Select.Option>
                          <Select.Option value="linux/386">linux/386</Select.Option>
                        </>
                      ) : (
                        architectures.map(arch => (
                          <Select.Option key={arch} value={arch}>{arch}</Select.Option>
                        ))
                      )}
                    </Select>
                  </Form.Item>

                  <Button
                    onClick={queryArchitectures}
                    loading={queryingArch}
                    style={{ marginBottom: '0px', flexShrink: 0 }}
                  >
                    查询可用架构
                  </Button>

                  {inspectLogs.length > 0 && (
                    <InfoCircleOutlined
                      onClick={() => setInspectModalVisible(true)}
                      style={{
                        fontSize: '18px',
                        color: '#1890ff',
                        cursor: 'pointer',
                        marginBottom: '0px',
                        marginLeft: '8px'
                      }}
                      title="查看详情"
                    />
                  )}
                </Space>

                {architectures.length > 0 && (
                  <div style={{ marginTop: '8px' }}>
                    <Space size={[0, 8]} wrap>
                      {architectures.map(arch => (
                        <Tag key={arch} color="blue">{arch}</Tag>
                      ))}
                    </Space>
                  </div>
                )}
              </div>
            </div>

            <div className="dest-section">
              <Title level={4}>目标镜像信息</Title>
              <Form.Item
                label="目标镜像地址"
                name="destImage"
                rules={[{ required: true, message: '请输入目标镜像地址' }]}
              >
                <Input
                  placeholder="例如: registry.example.com/nginx:latest"
                  suffix={
                    <CopyOutlined
                      onClick={() => handleCopyToClipboard('destImage')}
                      style={{ color: '#1890ff', cursor: 'pointer' }}
                      title="复制地址"
                    />
                  }
                />
              </Form.Item>

              <Space direction="horizontal" style={{ width: '100%', alignItems: 'flex-start', flexWrap: 'wrap' }} size="large">
                <Form.Item
                  label="目标仓库用户名"
                  name="destUsername"
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Input placeholder="选填（私有仓库需要）" />
                </Form.Item>

                <Form.Item
                  label="目标仓库密码"
                  name="destPassword"
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Input.Password placeholder="选填（私有仓库需要）" />
                </Form.Item>

                <Form.Item
                  label=" "
                  name="destTlsVerify"
                  valuePropName="checked"
                  initialValue={true}
                  style={{ flex: '1 1 200px', minWidth: '0' }}
                >
                  <Checkbox>启用 TLS 证书验证</Checkbox>
                </Form.Item>
              </Space>
            </div>

            <Collapse
              style={{ marginTop: '20px', marginBottom: '20px' }}
              items={[
                {
                  key: '1',
                  label: '高级选项',
                  children: (
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Form.Item
                        label="网络重试次数"
                        name="retryTimes"
                        initialValue={3}
                        style={{ maxWidth: '200px', marginBottom: '16px' }}
                        tooltip="当遇到网络超时等错误时，Skopeo 自动重试的次数（建议 1-10）"
                      >
                        <InputNumber min={0} max={100} placeholder="默认 3 次" style={{ width: '100%' }} />
                      </Form.Item>
                      <Alert
                        message="配置管理说明"
                        description="您可以保存多份配置，并通过下拉框快速切换。配置信息（包括密码）会以加密方式保存在服务器端，仅限您本人访问。"
                        type="info"
                        showIcon
                      />
                      <Space wrap style={{ width: '100%' }}>
                        <Text>选择配置：</Text>
                        <Select
                          placeholder="选择已保存的配置"
                          style={{ width: 250 }}
                          value={selectedConfig || undefined}
                          onChange={handleSelectConfig}
                          showSearch
                        >
                          {configList.map(name => (
                            <Select.Option key={name} value={name}>
                              {name}
                            </Select.Option>
                          ))}
                        </Select>
                        <Button type="primary" onClick={handleShowSaveConfigModal}>保存当前配置</Button>
                        <Button
                          danger
                          onClick={handleShowDeleteConfigModal}
                          disabled={!selectedConfig}
                        >
                          删除选中配置
                        </Button>
                      </Space>
                      <div style={{ marginTop: '16px', paddingTop: '16px', borderTop: '1px solid #f0f0f0' }}>
                        <Checkbox
                          checked={debugEnabled}
                          onChange={(e) => setDebugEnabled(e.target.checked)}
                        >
                          启用调试模式（显示调试按钮）
                        </Checkbox>
                      </div>
                    </Space>
                  ),
                },
              ]}
            />

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} size="large" block>
                开始同步
              </Button>
            </Form.Item>
          </Form>
        </Card>
        <div className="footer">
          <Text type="secondary">
            Image Sync · v{APP_VERSION}
            {GIT_COMMIT !== 'dev' && GIT_COMMIT_FULL !== 'development' && (
              <>
                {' · '}
                <a
                  href={`https://github.com/lazycatapps/image-sync/commit/${GIT_COMMIT_FULL}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  title={`Commit: ${GIT_COMMIT_FULL}`}
                >
                  {GIT_COMMIT}
                </a>
              </>
            )}
            {' · '}
            Copyright © {new Date().getFullYear()} Lazycat Apps
            {' · '}
            <a href="https://github.com/lazycatapps/image-sync" target="_blank" rel="noopener noreferrer">
              GitHub
            </a>
          </Text>
        </div>

        <Modal
          title="架构查询"
          open={inspectModalVisible}
          onCancel={handleCloseInspectModal}
          footer={[
            <Button key="close" onClick={handleCloseInspectModal}>
              关闭
            </Button>
          ]}
          width={800}
        >
          {inspectStatus && (
            <Alert
              message={inspectStatus === 'querying' ? '正在查询...' : inspectStatus === 'success' ? '查询成功' : '查询失败'}
              type={inspectStatus === 'success' ? 'success' : inspectStatus === 'error' ? 'error' : 'info'}
              style={{ marginBottom: '16px' }}
            />
          )}
          <div style={{
            background: '#000',
            color: '#0f0',
            padding: '16px',
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '12px',
            maxHeight: '500px',
            overflowY: 'auto'
          }}>
            {inspectLogs.map((log, index) => (
              <div key={index} style={{ marginBottom: '4px' }}>
                {log}
              </div>
            ))}
            <div ref={inspectLogsEndRef} />
          </div>
        </Modal>

        <Modal
          title={
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>同步日志</span>
              <div>
                <Button
                  type="text"
                  icon={<CopyOutlined />}
                  onClick={handleCopySyncLogs}
                  style={{ marginRight: '8px' }}
                >
                  复制
                </Button>
                <Button
                  type="text"
                  icon={logsModalMaximized ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
                  onClick={() => setLogsModalMaximized(!logsModalMaximized)}
                  style={{ marginRight: '24px' }}
                >
                  {logsModalMaximized ? '恢复' : '最大化'}
                </Button>
              </div>
            </div>
          }
          open={logsModalVisible}
          onCancel={handleCloseModal}
          footer={[
            <Button key="close" onClick={handleCloseModal}>
              关闭
            </Button>
          ]}
          width={logsModalMaximized ? '96vw' : 800}
          style={logsModalMaximized ? { top: 20, maxWidth: 'none', paddingBottom: 20 } : {}}
        >
          {syncStatus && (
            <Alert
              message={`任务状态: ${syncStatus}`}
              type={syncStatus === 'completed' ? 'success' : syncStatus === 'failed' ? 'error' : 'info'}
              style={{ marginBottom: '16px' }}
            />
          )}
          <div style={{
            background: '#000',
            color: '#0f0',
            padding: '16px',
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '12px',
            maxHeight: logsModalMaximized ? 'calc(96vh - 200px)' : '500px',
            overflowY: 'auto'
          }}>
            {syncLogs.map((log, index) => (
              <div key={index} style={{ marginBottom: '4px' }}>
                {log}
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        </Modal>

        <Modal
          title="调试日志"
          open={debugModalVisible}
          onCancel={() => setDebugModalVisible(false)}
          footer={[
            <Button key="clear" onClick={() => setDebugLogs([])}>
              清空日志
            </Button>,
            <Button key="close" onClick={() => setDebugModalVisible(false)}>
              关闭
            </Button>
          ]}
          width={900}
        >
          <Alert
            message="调试信息"
            description={`总计 ${debugLogs.length} 条日志记录`}
            type="info"
            style={{ marginBottom: '16px' }}
          />
          <div style={{
            background: '#1e1e1e',
            color: '#d4d4d4',
            padding: '16px',
            borderRadius: '4px',
            fontFamily: 'Consolas, Monaco, "Courier New", monospace',
            fontSize: '12px',
            maxHeight: '600px',
            overflowY: 'auto'
          }}>
            {debugLogs.map((log, index) => (
              <div key={index} style={{ marginBottom: '8px', borderBottom: '1px solid #333', paddingBottom: '4px' }}>
                <div style={{ color: '#569cd6' }}>
                  [{log.timestamp}] <span style={{ color: '#4ec9b0' }}>[{log.category}]</span>
                </div>
                <div style={{ color: '#ce9178', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {log.message}
                </div>
              </div>
            ))}
            <div ref={debugLogsEndRef} />
          </div>
        </Modal>

        <Modal
          title="保存配置"
          open={saveConfigModalVisible}
          onOk={handleSaveConfigWithName}
          onCancel={() => setSaveConfigModalVisible(false)}
          okText="保存"
          cancelText="取消"
        >
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              message="配置名称规则"
              description="只能包含字母、数字、点、横线和下划线，例如：default、prod-env、my.config"
              type="info"
              showIcon
              style={{ marginBottom: '16px' }}
            />
            <Text>配置名称：</Text>
            <Input
              placeholder="输入配置名称"
              value={configNameInput}
              onChange={(e) => setConfigNameInput(e.target.value)}
              onPressEnter={handleSaveConfigWithName}
            />
            {configList.includes(configNameInput.trim()) && (
              <Alert
                message="配置名称已存在，保存将会覆盖原配置"
                type="warning"
                showIcon
              />
            )}
          </Space>
        </Modal>

        <Modal
          title="确认删除"
          open={deleteConfigModalVisible}
          onOk={handleDeleteConfig}
          onCancel={() => {
            addDebugLog('CONFIG', 'User cancelled deletion');
            setDeleteConfigModalVisible(false);
          }}
          okText="删除"
          okType="danger"
          cancelText="取消"
        >
          <Alert
            message="警告"
            description={`确定要删除配置 "${selectedConfig}" 吗？此操作不可恢复。`}
            type="warning"
            showIcon
          />
        </Modal>

        {debugEnabled && (
          <FloatButton
            icon={<BugOutlined />}
            type="primary"
            style={{ right: 24, bottom: 24 }}
            onClick={() => setDebugModalVisible(true)}
            badge={{ count: debugLogs.length, overflowCount: 99 }}
            tooltip="查看调试日志"
          />
        )}
      </div>
    </div>
  );
}

function App() {
  return (
    <AntApp>
      <AppContent />
    </AntApp>
  );
}

export default App;
