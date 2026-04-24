import React, { useEffect, useState } from 'react';
import { Button, SideSheet, Typography } from '@douyinfe/semi-ui';
import { IconMenu } from '@douyinfe/semi-icons';
import { Navigate, useNavigate, useParams } from 'react-router-dom';

import { useIsMobile } from '../../hooks/common/useIsMobile';
import {
  createAiModelDocSelectionHandler,
  getAiModelDocById,
  resolveAiModelDocPageState,
} from './catalog';
import DocsSidebar from './DocsSidebar';
import DocContent from './DocContent';

const { Title, Text } = Typography;

const ApiDocsPageV1 = () => {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const { category, docId } = useParams();
  const [sidebarVisible, setSidebarVisible] = useState(false);
  const routeState = resolveAiModelDocPageState(category, docId);

  useEffect(() => {
    setSidebarVisible(false);
  }, [docId]);

  if (routeState.shouldRedirect) {
    return <Navigate to={routeState.redirectTo} replace />;
  }

  const doc = getAiModelDocById(routeState.docId);
  const handleSelectDoc = createAiModelDocSelectionHandler(navigate, () =>
    setSidebarVisible(false),
  );

  return (
    <div className='min-h-[calc(100vh-64px)] bg-[var(--semi-color-bg-0)] pt-16'>
      <div className='mx-auto flex w-full max-w-[1600px] flex-col gap-4 px-4 py-4 lg:px-6'>
        <div className='flex items-center justify-between gap-3 rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-4 py-3'>
          <div className='min-w-0'>
            <Title heading={4} className='m-0'>
              AI 模型接口文档
            </Title>
            <Text type='secondary'>按模型接口分组浏览本地文档</Text>
          </div>
          {isMobile && (
            <Button
              icon={<IconMenu />}
              theme='borderless'
              type='tertiary'
              onClick={() => setSidebarVisible(true)}
            >
              目录
            </Button>
          )}
        </div>

        <div className='flex items-start gap-4'>
          {!isMobile && (
            <aside className='sticky top-[80px] w-[320px] shrink-0'>
              <div className='max-h-[calc(100vh-96px)] overflow-y-auto pr-2 [scrollbar-color:var(--semi-color-fill-2)_transparent] [scrollbar-width:thin] [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-[var(--semi-color-fill-2)] [&::-webkit-scrollbar-track]:bg-transparent'>
                <DocsSidebar activeDocId={doc.id} onSelectDoc={handleSelectDoc} />
              </div>
            </aside>
          )}

          <main className='min-w-0 flex-1'>
            <DocContent doc={doc} />
          </main>
        </div>
      </div>

      {isMobile && (
        <SideSheet
          title='接口目录'
          visible={sidebarVisible}
          onCancel={() => setSidebarVisible(false)}
          width='100%'
          placement='left'
        >
          <DocsSidebar activeDocId={doc.id} onSelectDoc={handleSelectDoc} />
        </SideSheet>
      )}
    </div>
  );
};

export default ApiDocsPageV1;
