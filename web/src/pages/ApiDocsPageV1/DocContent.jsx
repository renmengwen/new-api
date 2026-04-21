import React from 'react';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';

const { Title, Text, Paragraph } = Typography;

const methodColorMap = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  PATCH: 'yellow',
  DELETE: 'red',
};

const Section = ({ title, children }) => (
  <section className='space-y-2'>
    <Title heading={5} className='m-0'>
      {title}
    </Title>
    {children}
  </section>
);

const DocContent = ({ doc }) => {
  if (!doc) {
    return (
      <Card className='rounded-2xl'>
        <div className='py-12 text-center'>
          <Title heading={4}>暂无可用文档</Title>
          <Text type='secondary'>请选择左侧接口查看详情</Text>
        </div>
      </Card>
    );
  }

  const methodColor = methodColorMap[doc.method] || 'grey';

  return (
    <Card className='rounded-2xl'>
      <div className='space-y-6'>
        <div className='space-y-3'>
          <div className='flex flex-wrap items-center gap-3'>
            <Tag color={methodColor}>{doc.method}</Tag>
            <Title heading={3} className='m-0'>
              {doc.title}
            </Title>
          </div>
          <Paragraph className='m-0' type='secondary'>
            {doc.summary}
          </Paragraph>
        </div>

        <Section title='接口概览'>
          <Text>{doc.description}</Text>
        </Section>

        <Section title='请求路径'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <Text className='font-mono'>{doc.method} {doc.path}</Text>
          </Card>
        </Section>

        <Section title='鉴权方式'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <Text>
              {doc.auth?.example || 'Authorization: Bearer sk-xxxxxxxx'}
            </Text>
          </Card>
        </Section>

        <Section title='请求示例'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <pre className='m-0 whitespace-pre-wrap break-all text-sm leading-6'>
              {doc.requestExample || '暂无请求示例'}
            </pre>
          </Card>
        </Section>

        <Section title='响应示例'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <pre className='m-0 whitespace-pre-wrap break-all text-sm leading-6'>
              {doc.responseExample || '暂无响应示例'}
            </pre>
          </Card>
        </Section>
      </div>
    </Card>
  );
};

export default DocContent;
