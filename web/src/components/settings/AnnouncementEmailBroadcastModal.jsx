import React, { useEffect, useRef, useState } from 'react';
import { Button, Form, Modal, Space, Typography } from '@douyinfe/semi-ui';
import { Send } from 'lucide-react';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const defaultStats = { sent_count: 0, skipped_count: 0, failed_count: 0 };

const targetOptions = (t) => [
  { label: t('代理商'), value: 'agent' },
  { label: t('普通用户'), value: 'end_user' },
  { label: t('全量用户'), value: 'all' },
];

const AnnouncementEmailBroadcastModal = ({
  visible,
  source,
  defaultTitle,
  defaultContent,
  onClose,
}) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState(null);

  useEffect(() => {
    if (!visible) {
      return;
    }
    setStats(null);
    formApiRef.current?.setValues({
      target: 'all',
      title: defaultTitle || '',
      content: defaultContent || '',
    });
  }, [visible, defaultTitle, defaultContent]);

  const handleSend = async () => {
    const values = formApiRef.current?.getValues() || {};
    if (!values.target || !values.title?.trim() || !values.content?.trim()) {
      showError(t('请填写完整的邮件信息'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post('/api/notice/email-broadcast', {
        source,
        target: values.target,
        title: values.title.trim(),
        content: marked.parse(values.content || ''),
      });
      const { success, message, data } = res.data;
      const nextStats = data || defaultStats;
      setStats(nextStats);
      if (success) {
        showSuccess(
          `${t('邮件发送完成')}：${t('已发送')} ${nextStats.sent_count}，${t('跳过')} ${nextStats.skipped_count}，${t('失败')} ${nextStats.failed_count}`,
        );
        onClose?.();
      } else {
        showError(message || t('邮件发送失败'));
      }
    } catch (err) {
      showError(err.message || t('邮件发送失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t('邮件发送确认')}
      visible={visible}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose}>{t('取消')}</Button>
          <Button
            type='primary'
            icon={<Send size={14} />}
            loading={loading}
            onClick={handleSend}
          >
            {t('发送邮件')}
          </Button>
        </Space>
      }
      width={720}
    >
      <Form
        layout='vertical'
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        initValues={{
          target: 'all',
          title: defaultTitle || '',
          content: defaultContent || '',
        }}
      >
        <Form.Select
          field='target'
          label={t('接收用户')}
          optionList={targetOptions(t)}
          rules={[{ required: true, message: t('请选择接收用户') }]}
        />
        <Form.Input
          field='title'
          label={t('邮件标题')}
          rules={[{ required: true, message: t('请输入邮件标题') }]}
        />
        <Form.TextArea
          field='content'
          label={t('邮件正文')}
          autosize={{ minRows: 8, maxRows: 16 }}
          rules={[{ required: true, message: t('请输入邮件正文') }]}
        />
      </Form>
      {stats ? (
        <Text type='tertiary'>
          {`${t('已发送')} ${stats.sent_count}，${t('跳过')} ${stats.skipped_count}，${t('失败')} ${stats.failed_count}`}
        </Text>
      ) : null}
    </Modal>
  );
};

export default AnnouncementEmailBroadcastModal;
