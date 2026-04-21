import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, Tag, Typography } from '@douyinfe/semi-ui';
import { IconChevronDown, IconChevronRight } from '@douyinfe/semi-icons';

import { buildAiModelDocTree, expandAiModelDocGroups } from './catalog';

const { Text } = Typography;

const methodColorMap = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  PATCH: 'yellow',
  DELETE: 'red',
};

const DocsSidebar = ({ activeDocId, onSelectDoc }) => {
  const groups = useMemo(() => buildAiModelDocTree(), []);
  const [expandedGroups, setExpandedGroups] = useState(() =>
    groups.map((group) => group.key),
  );

  useEffect(() => {
    setExpandedGroups((current) => expandAiModelDocGroups(current, activeDocId));
  }, [activeDocId]);

  const toggleGroup = (groupKey) => {
    setExpandedGroups((current) =>
      current.includes(groupKey)
        ? current.filter((key) => key !== groupKey)
        : [...current, groupKey],
    );
  };

  return (
    <Card className='h-full rounded-2xl'>
      <div className='space-y-3'>
        {groups.map((group) => {
          const expanded = expandedGroups.includes(group.key);

          return (
            <section key={group.key} className='space-y-2'>
              <Button
                block
                theme='borderless'
                type='tertiary'
                icon={expanded ? <IconChevronDown /> : <IconChevronRight />}
                onClick={() => toggleGroup(group.key)}
              >
                <span className='flex w-full items-center justify-between gap-2 text-left'>
                  <span>{group.title}</span>
                  <Text type='secondary'>{group.items.length}</Text>
                </span>
              </Button>

              {expanded && (
                <div className='space-y-2 pl-2'>
                  {group.items.map((doc) => {
                    const methodColor = methodColorMap[doc.method] || 'grey';
                    const selected = doc.id === activeDocId;

                    return (
                      <button
                        key={doc.id}
                        type='button'
                        onClick={() => onSelectDoc(doc.id)}
                        className={[
                          'flex w-full items-start gap-3 rounded-xl border px-3 py-2 text-left transition-colors',
                          selected
                            ? 'border-[var(--semi-color-primary)] bg-[var(--semi-color-primary-light-default)]'
                            : 'border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] hover:bg-[var(--semi-color-fill-1)]',
                        ].join(' ')}
                      >
                        <Tag color={methodColor} size='small'>
                          {doc.method}
                        </Tag>
                        <div className='min-w-0 flex-1'>
                          <div className='truncate text-sm font-medium'>
                            {doc.title}
                          </div>
                          <div className='truncate text-xs text-[var(--semi-color-text-2)]'>
                            {doc.path}
                          </div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </section>
          );
        })}
      </div>
    </Card>
  );
};

export default DocsSidebar;
