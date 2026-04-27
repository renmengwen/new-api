import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Divider,
  Form,
  Input,
  Row,
  Switch,
  TextArea,
} from '@douyinfe/semi-ui';
import { showError, showSuccess } from '../../helpers';
import {
  defaultAboutPageConfig,
  normalizeAboutPageConfig,
} from '../../pages/About/aboutPageConfig';

const fieldStyle = {
  marginBottom: 12,
};

const labelStyle = {
  display: 'block',
  marginBottom: 6,
  color: 'var(--semi-color-text-1)',
  fontWeight: 600,
};

const rowGroupStyle = {
  border: '1px solid var(--semi-color-border)',
  borderRadius: 6,
  padding: 12,
  marginBottom: 12,
};

const fullColProps = {
  xs: 24,
  sm: 24,
  lg: 24,
};

const quarterColProps = {
  xs: 24,
  sm: 12,
  lg: 6,
};

const thirdColProps = {
  xs: 24,
  sm: 12,
  lg: 8,
};

const halfColProps = {
  xs: 24,
  sm: 12,
  lg: 12,
};

const cloneConfig = (value) => JSON.parse(JSON.stringify(value));

const stripConfigMetadata = (value) => {
  if (Array.isArray(value)) {
    return value.map(stripConfigMetadata);
  }

  if (value && typeof value === 'object') {
    const cleaned = Object.keys(value).reduce((result, key) => {
      result[key] = stripConfigMetadata(value[key]);
      return result;
    }, {});

    Object.keys(cleaned).forEach((key) => {
      if (key.startsWith('__')) {
        delete cleaned[key];
      }
    });

    return cleaned;
  }

  return value;
};

const getPathValue = (source, path) =>
  path.reduce((current, key) => current?.[key], source);

const AboutPageSetting = ({
  inputs,
  loadingInput,
  updateOption,
  setInputs,
  setLoadingInput,
  t,
}) => {
  const translate = (text) => (typeof t === 'function' ? t(text) : text);
  const [config, setConfig] = useState(() =>
    normalizeAboutPageConfig(inputs.AboutPageConfig || defaultAboutPageConfig),
  );

  useEffect(() => {
    setConfig(
      normalizeAboutPageConfig(
        inputs.AboutPageConfig || defaultAboutPageConfig,
      ),
    );
  }, [inputs.AboutPageConfig]);

  const updateConfigValue = (path, value) => {
    setConfig((previousConfig) => {
      const nextConfig = cloneConfig(previousConfig);
      const lastKey = path[path.length - 1];
      const target = path
        .slice(0, -1)
        .reduce((current, key) => current[key], nextConfig);

      target[lastKey] = value;
      return nextConfig;
    });
  };

  const renderInput = (label, path, placeholder = label, props = {}) => (
    <div style={fieldStyle}>
      <label style={labelStyle}>{translate(label)}</label>
      <Input
        value={getPathValue(config, path) ?? ''}
        placeholder={translate(placeholder)}
        onChange={(value) => updateConfigValue(path, value)}
        {...props}
      />
    </div>
  );

  const renderTextArea = (label, path, placeholder = label, props = {}) => (
    <div style={fieldStyle}>
      <label style={labelStyle}>{translate(label)}</label>
      <TextArea
        value={getPathValue(config, path) ?? ''}
        placeholder={translate(placeholder)}
        autosize={{ minRows: 3, maxRows: 8 }}
        onChange={(value) => updateConfigValue(path, value)}
        {...props}
      />
    </div>
  );

  const renderLegacyTextArea = () => (
    <div style={fieldStyle}>
      <label style={labelStyle}>{translate('旧版关于内容')}</label>
      <TextArea
        value={inputs.About || ''}
        placeholder={translate(
          '填写旧版关于页面 Markdown 或 HTML 内容；结构化配置启用后仍可保留兼容内容',
        )}
        autosize={{ minRows: 6, maxRows: 12 }}
        style={{ fontFamily: 'JetBrains Mono, Consolas' }}
        onChange={(value) =>
          setInputs((previousInputs) => ({ ...previousInputs, About: value }))
        }
      />
    </div>
  );

  const handleSaveConfig = async () => {
    try {
      setLoadingInput((previousLoadingInput) => ({
        ...previousLoadingInput,
        AboutPageConfig: true,
      }));
      const cleanConfig = stripConfigMetadata(normalizeAboutPageConfig(config));
      const serializedConfig = JSON.stringify(cleanConfig);

      await updateOption('AboutPageConfig', serializedConfig);
      setInputs((previousInputs) => ({
        ...previousInputs,
        AboutPageConfig: serializedConfig,
      }));
      showSuccess(translate('关于页面配置已更新'));
    } catch (error) {
      console.error(translate('关于页面配置更新失败'), error);
      showError(translate('关于页面配置更新失败'));
    } finally {
      setLoadingInput((previousLoadingInput) => ({
        ...previousLoadingInput,
        AboutPageConfig: false,
      }));
    }
  };

  const handleSaveLegacyAbout = async () => {
    try {
      setLoadingInput((previousLoadingInput) => ({
        ...previousLoadingInput,
        About: true,
      }));
      await updateOption('About', inputs.About);
      setInputs((previousInputs) => ({
        ...previousInputs,
        About: inputs.About,
      }));
      showSuccess(translate('旧版关于内容已更新'));
    } catch (error) {
      console.error(translate('旧版关于内容更新失败'), error);
      showError(translate('旧版关于内容更新失败'));
    } finally {
      setLoadingInput((previousLoadingInput) => ({
        ...previousLoadingInput,
        About: false,
      }));
    }
  };

  const contactHeadings = ['微信客服', '企业微信客服'];

  return (
    <>
      <Divider margin='12px' />
      <Form.Section text={translate('关于页面配置/开关')}>
        <Banner
          fullMode={false}
          type='info'
          description={translate(
            '结构化配置用于新版关于页面；高级兼容内容可继续保存旧版 About 配置。',
          )}
          closeIcon={null}
          style={{ marginBottom: 12 }}
        />
        <Row>
          <Col {...fullColProps}>
            <div style={fieldStyle}>
              <label style={labelStyle}>
                {translate('启用结构化关于页面')}
              </label>
              <Switch
                checked={config.enabled === true}
                checkedText={translate('开')}
                uncheckedText={translate('关')}
                onChange={(value) => updateConfigValue(['enabled'], value)}
              />
            </div>
          </Col>
        </Row>
      </Form.Section>

      <Form.Section text={translate('首屏内容')}>
        <Row gutter={16}>
          <Col {...thirdColProps}>
            {renderInput('眉标', ['hero', 'eyebrow'])}
          </Col>
          <Col {...thirdColProps}>
            {renderInput('主标题', ['hero', 'title'])}
          </Col>
          <Col {...thirdColProps}>
            {renderInput('副标题', ['hero', 'subtitle'])}
          </Col>
        </Row>
        <Row gutter={16}>
          <Col {...quarterColProps}>
            {renderInput('主按钮文案', ['hero', 'primaryActionText'])}
          </Col>
          <Col {...quarterColProps}>
            {renderInput('主按钮链接', ['hero', 'primaryActionUrl'])}
          </Col>
          <Col {...quarterColProps}>
            {renderInput('次按钮文案', ['hero', 'secondaryActionText'])}
          </Col>
          <Col {...quarterColProps}>
            {renderInput('次按钮链接', ['hero', 'secondaryActionUrl'])}
          </Col>
        </Row>
      </Form.Section>

      <Form.Section text={translate('平台概览')}>
        <Row gutter={16}>
          <Col {...thirdColProps}>
            {renderInput('概览标题', ['overview', 'title'])}
          </Col>
          <Col {...thirdColProps}>
            {renderInput('运行状态', ['overview', 'status'])}
          </Col>
          <Col {...thirdColProps}>
            {renderTextArea('概览描述', ['overview', 'description'])}
          </Col>
        </Row>
        <Divider margin='12px' align='left'>
          {translate('指标')}
        </Divider>
        {config.overview.metrics.map((metric, index) => (
          <div key={`metric-${index}`} style={rowGroupStyle}>
            <Row gutter={16}>
              <Col {...halfColProps}>
                {renderInput('指标数值', [
                  'overview',
                  'metrics',
                  index,
                  'value',
                ])}
              </Col>
              <Col {...halfColProps}>
                {renderInput('指标标签', [
                  'overview',
                  'metrics',
                  index,
                  'label',
                ])}
              </Col>
            </Row>
          </div>
        ))}
        <Divider margin='12px' align='left'>
          {translate('渠道')}
        </Divider>
        {config.overview.channels.map((channel, index) => (
          <div key={`channel-${index}`} style={rowGroupStyle}>
            <Row gutter={16}>
              <Col {...thirdColProps}>
                {renderInput('渠道名称', [
                  'overview',
                  'channels',
                  index,
                  'name',
                ])}
              </Col>
              <Col {...thirdColProps}>
                {renderInput(
                  '渠道占比',
                  ['overview', 'channels', index, 'value'],
                  '0-100',
                  { type: 'number' },
                )}
              </Col>
              <Col {...thirdColProps}>
                {renderInput('渠道状态', [
                  'overview',
                  'channels',
                  index,
                  'status',
                ])}
              </Col>
            </Row>
          </div>
        ))}
      </Form.Section>

      <Form.Section text={translate('能力卡片')}>
        {config.capabilities.map((capability, index) => (
          <div key={`capability-${index}`} style={rowGroupStyle}>
            <Row gutter={16}>
              <Col {...quarterColProps}>
                {renderInput('图标标识', ['capabilities', index, 'icon'])}
              </Col>
              <Col {...quarterColProps}>
                {renderInput('卡片标题', ['capabilities', index, 'title'])}
              </Col>
              <Col {...halfColProps}>
                {renderTextArea('卡片描述', [
                  'capabilities',
                  index,
                  'description',
                ])}
              </Col>
            </Row>
          </div>
        ))}
      </Form.Section>

      <Form.Section text={translate('集团背书')}>
        <Row gutter={16}>
          <Col {...thirdColProps}>
            {renderInput('集团标题', ['group', 'title'])}
          </Col>
          <Col {...thirdColProps}>
            {renderInput('集团状态', ['group', 'status'])}
          </Col>
          <Col {...thirdColProps}>
            {renderTextArea('集团描述', ['group', 'description'])}
          </Col>
        </Row>
        {config.group.bullets.map((bullet, index) => (
          <div key={`group-bullet-${index}`} style={rowGroupStyle}>
            {renderInput('背书要点', ['group', 'bullets', index])}
          </div>
        ))}
        <Row gutter={16}>
          <Col {...halfColProps}>
            {renderInput('官网按钮文案', ['group', 'websiteLabel'])}
          </Col>
          <Col {...halfColProps}>
            {renderInput('官网链接', ['group', 'websiteUrl'])}
          </Col>
        </Row>
      </Form.Section>

      <Form.Section text={translate('客服二维码')}>
        {[0, 1].map((index) => (
          <div key={`contact-${index}`} style={rowGroupStyle}>
            <Divider margin='8px' align='left'>
              {translate(contactHeadings[index])}
            </Divider>
            <Row gutter={16}>
              <Col {...thirdColProps}>
                {renderInput('客服标题', ['contacts', index, 'title'])}
              </Col>
              <Col {...thirdColProps}>
                {renderInput('二维码图片地址', ['contacts', index, 'imageUrl'])}
              </Col>
              <Col {...thirdColProps}>
                {renderInput('备用链接', ['contacts', index, 'fallbackUrl'])}
              </Col>
            </Row>
            {renderTextArea('客服说明', ['contacts', index, 'description'])}
          </div>
        ))}
      </Form.Section>

      <Form.Section text={translate('高级兼容内容')}>
        {renderTextArea('自定义内容', ['customContent'])}
        <Button
          type='primary'
          onClick={handleSaveConfig}
          loading={loadingInput.AboutPageConfig}
          style={{ marginBottom: 12 }}
        >
          {translate('保存关于页面配置')}
        </Button>
        <Banner
          fullMode={false}
          type='warning'
          description={translate(
            '旧版 About 内容仅用于兼容旧页面或回退场景，保存按钮会单独写入 About 配置项。',
          )}
          closeIcon={null}
          style={{ marginBottom: 12 }}
        />
        {renderLegacyTextArea()}
        <Button onClick={handleSaveLegacyAbout} loading={loadingInput.About}>
          {translate('保存旧版关于内容')}
        </Button>
      </Form.Section>
    </>
  );
};

export default AboutPageSetting;
