import React from 'react';
import { Card, Tag, Typography } from '@douyinfe/semi-ui';
import { marked } from 'marked';

import { getAiModelDocDisplayState } from './catalog';
import seedanceVideoTaskApis from '../../../../docs/seedance-video-task-apis.md?raw';

const { Title, Text, Paragraph } = Typography;

const methodColorMap = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  PATCH: 'yellow',
  DELETE: 'red',
};

const markdownContentByDocId = {
  'videos-seedance': seedanceVideoTaskApis,
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
  const displayState = getAiModelDocDisplayState(doc);

  if (displayState.kind === 'empty') {
    return (
      <Card className='rounded-2xl'>
        <div className='py-12 text-center'>
          <Title heading={4}>{displayState.title}</Title>
          <Text type='secondary'>{displayState.message}</Text>
        </div>
      </Card>
    );
  }

  if (displayState.kind === 'placeholder') {
    const methodColor = methodColorMap[displayState.method] || 'grey';

    return (
      <Card className='rounded-2xl'>
        <div className='space-y-4 py-8'>
          <div className='flex flex-wrap items-center gap-3'>
            <Tag color={methodColor}>{displayState.method}</Tag>
            <Title heading={3} className='m-0'>
              {displayState.title}
            </Title>
          </div>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <div className='space-y-2'>
              <Text type='secondary'>占位文档</Text>
              <Paragraph className='m-0'>{displayState.message}</Paragraph>
              <Text className='font-mono'>{displayState.path}</Text>
            </div>
          </Card>
        </div>
      </Card>
    );
  }

  if (displayState.kind === 'markdown') {
    const methodColor = methodColorMap[displayState.method] || 'grey';
    const markdownContent = markdownContentByDocId[displayState.id] || '';

    return (
      <Card className='rounded-2xl'>
        <div className='space-y-6'>
          <div className='space-y-3'>
            <div className='flex flex-wrap items-center gap-3'>
              <Tag color={methodColor}>{displayState.method}</Tag>
              <Title heading={3} className='m-0'>
                {displayState.title}
              </Title>
            </div>
            <Paragraph className='m-0' type='secondary'>
              {displayState.summary}
            </Paragraph>
            <Text className='font-mono'>{displayState.path}</Text>
          </div>

          <div
            className='max-w-none text-sm leading-7 text-[var(--semi-color-text-0)] [&_code]:rounded [&_code]:bg-[var(--semi-color-fill-1)] [&_code]:px-1 [&_h1]:mt-0 [&_h1]:text-2xl [&_h2]:mt-8 [&_h2]:text-xl [&_h3]:mt-6 [&_h3]:text-lg [&_pre]:overflow-x-auto [&_pre]:rounded-xl [&_pre]:bg-[var(--semi-color-fill-0)] [&_pre]:p-4 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-[var(--semi-color-border)] [&_td]:px-3 [&_td]:py-2 [&_th]:border [&_th]:border-[var(--semi-color-border)] [&_th]:bg-[var(--semi-color-fill-0)] [&_th]:px-3 [&_th]:py-2'
            dangerouslySetInnerHTML={{ __html: marked.parse(markdownContent) }}
          />
        </div>
      </Card>
    );
  }

  const methodColor = methodColorMap[displayState.method] || 'grey';

  return (
    <Card className='rounded-2xl'>
      <div className='space-y-6'>
        <div className='space-y-3'>
          <div className='flex flex-wrap items-center gap-3'>
            <Tag color={methodColor}>{displayState.method}</Tag>
            <Title heading={3} className='m-0'>
              {displayState.title}
            </Title>
          </div>
          <Paragraph className='m-0' type='secondary'>
            {displayState.summary}
          </Paragraph>
        </div>

        <Section title='接口概览'>
          <Text>{displayState.description}</Text>
        </Section>

        <Section title='请求路径'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <Text className='font-mono'>
              {displayState.method} {displayState.path}
            </Text>
          </Card>
        </Section>

        <Section title='鉴权方式'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <Text>{displayState.authExample}</Text>
          </Card>
        </Section>

        <Section title='请求示例'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <pre className='m-0 whitespace-pre-wrap break-all text-sm leading-6'>
              {displayState.requestExample}
            </pre>
          </Card>
        </Section>

        <Section title='响应示例'>
          <Card className='rounded-xl bg-[var(--semi-color-fill-0)]'>
            <pre className='m-0 whitespace-pre-wrap break-all text-sm leading-6'>
              {displayState.responseExample}
            </pre>
          </Card>
        </Section>
      </div>
    </Card>
  );
};

export default DocContent;
