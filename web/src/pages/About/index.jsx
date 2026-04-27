/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { API, showError } from '../../helpers';
import { marked } from 'marked';
import { Empty } from '@douyinfe/semi-ui';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import AboutStructuredPage from './AboutStructuredPage';
import {
  isStructuredAboutEnabled,
  normalizeAboutPageConfig,
  parseAboutResponse,
} from './aboutPageConfig';

const ABOUT_CACHE_KEY = 'about';
const ABOUT_CONFIG_CACHE_KEY = 'about_page_config';

const readLocalStorage = (key) => {
  try {
    return localStorage.getItem(key) || '';
  } catch {
    return '';
  }
};

const writeLocalStorage = (key, value) => {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Ignore storage errors so the About page can still render fetched content.
  }
};

const removeLocalStorage = (key) => {
  try {
    localStorage.removeItem(key);
  } catch {
    // Ignore storage errors so the About page can still render fetched content.
  }
};

const loadCachedAboutConfig = () => {
  const cachedConfig = readLocalStorage(ABOUT_CONFIG_CACHE_KEY);

  return cachedConfig ? normalizeAboutPageConfig(cachedConfig) : null;
};

const About = () => {
  const { t } = useTranslation();
  const [about, setAbout] = useState(() => readLocalStorage(ABOUT_CACHE_KEY));
  const [legacyAbout, setLegacyAbout] = useState(() =>
    readLocalStorage(ABOUT_CACHE_KEY),
  );
  const [aboutConfig, setAboutConfig] = useState(loadCachedAboutConfig);
  const [aboutLoaded, setAboutLoaded] = useState(false);
  const currentYear = new Date().getFullYear();

  const displayAbout = async () => {
    setAbout(readLocalStorage(ABOUT_CACHE_KEY));
    try {
      const res = await API.get('/api/about');
      const { success, message } = res.data;
      if (success) {
        const { legacy, config } = parseAboutResponse(res.data);
        const normalizedConfig = normalizeAboutPageConfig(config);
        let aboutContent = legacy;
        if (legacy && !legacy.startsWith('https://')) {
          aboutContent = marked.parse(legacy);
        }
        setLegacyAbout(legacy);
        setAbout(aboutContent);
        setAboutConfig(normalizedConfig);
        writeLocalStorage(ABOUT_CACHE_KEY, aboutContent);
        writeLocalStorage(
          ABOUT_CONFIG_CACHE_KEY,
          JSON.stringify(normalizedConfig),
        );
      } else {
        showError(message);
        setAbout(t('加载关于内容失败...'));
        setLegacyAbout('');
        setAboutConfig(null);
        removeLocalStorage(ABOUT_CONFIG_CACHE_KEY);
      }
    } catch (error) {
      showError(error?.message || t('加载关于内容失败...'));
      setAbout(t('加载关于内容失败...'));
      setLegacyAbout('');
      setAboutConfig(null);
      removeLocalStorage(ABOUT_CONFIG_CACHE_KEY);
    } finally {
      setAboutLoaded(true);
    }
  };

  useEffect(() => {
    displayAbout().then();
  }, []);

  const emptyStyle = {
    padding: '24px',
  };

  const customDescription = (
    <div >

    </div>
  );

  const shouldRenderStructuredAbout = isStructuredAboutEnabled(
    aboutConfig,
    legacyAbout,
  );

  return (
    <div className='mt-[60px] px-2'>
      {shouldRenderStructuredAbout ? (
        <AboutStructuredPage
          config={aboutConfig}
        />
      ) : aboutLoaded && about === '' ? (
        <div className='flex justify-center items-center h-screen p-8'>
          <Empty
            image={
              <IllustrationConstruction style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationConstructionDark
                style={{ width: 150, height: 150 }}
              />
            }
            description={t('管理员暂时未设置任何关于内容')}
            style={emptyStyle}
          >
          </Empty>
        </div>
      ) : (
        <>
          {about.startsWith('https://') ? (
            <iframe
              src={about}
              style={{ width: '100%', height: '100vh', border: 'none' }}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: about }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default About;
