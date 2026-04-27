import React, { useMemo, useState } from 'react';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import {
  BarChart3,
  Building2,
  CheckCircle2,
  ExternalLink,
  MessageCircle,
  Network,
  QrCode,
  Route,
  ShieldCheck,
  Sparkles,
} from 'lucide-react';
import { translateAboutPageConfig } from './aboutPageConfig';
import './about.css';

const capabilityIconMap = {
  network: Network,
  route: Route,
  shield: ShieldCheck,
  chart: BarChart3,
};

const contactIconMap = {
  wechat: MessageCircle,
  work_wechat: Building2,
};

const hasText = (value) => typeof value === 'string' && value.trim() !== '';

const isExternalUrl = (url) => /^https?:\/\//i.test(url);

const renderIcon = (iconName, iconMap, fallbackIcon = Sparkles) => {
  const Icon = iconMap[iconName] || fallbackIcon;

  return <Icon aria-hidden='true' size={24} strokeWidth={1.8} />;
};

const AboutAction = ({ href, children, variant = 'primary' }) => {
  if (!hasText(href) || !hasText(children)) {
    return null;
  }

  const external = isExternalUrl(href);

  return (
    <a
      className={`about-action about-action-${variant}`}
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
    >
      <span>{children}</span>
      <ExternalLink aria-hidden='true' size={16} strokeWidth={2} />
    </a>
  );
};

const SafeLink = ({ href, children, className }) => {
  if (!hasText(href) || !hasText(children)) {
    return null;
  }

  const external = isExternalUrl(href);

  return (
    <a
      className={className}
      href={href}
      target={external ? '_blank' : undefined}
      rel={external ? 'noopener noreferrer' : undefined}
    >
      <span>{children}</span>
      <ExternalLink aria-hidden='true' size={14} strokeWidth={2} />
    </a>
  );
};

const QrImage = ({ contact }) => {
  const { t } = useTranslation();
  const [hasImageError, setHasImageError] = useState(false);
  const currentImageUrl =
    hasText(contact.imageUrl) && !hasImageError ? contact.imageUrl : '';

  const handleImageError = () => {
    setHasImageError(true);
  };

  if (!currentImageUrl) {
    return (
      <div className='about-qr-placeholder'>
        <QrCode aria-hidden='true' size={36} strokeWidth={1.7} />
        <span>{t('二维码暂未配置')}</span>
      </div>
    );
  }

  return (
    <img
      className='about-qr-image'
      src={currentImageUrl}
      alt={contact.title || t('联系二维码')}
      loading='lazy'
      onError={handleImageError}
    />
  );
};

const getContactTitle = (contact, t) => {
  if (hasText(contact.title)) {
    return contact.title;
  }

  return contact.type === 'work_wechat' ? t('企业微信') : t('微信客服');
};

const AboutStructuredPage = ({ config, protectedAttribution = null }) => {
  const { t } = useTranslation();
  const displayConfig = useMemo(
    () => translateAboutPageConfig(config, t),
    [config, t],
  );
  const hero = displayConfig?.hero || {};
  const overview = displayConfig?.overview || {};
  const group = displayConfig?.group || {};
  const capabilities = Array.isArray(displayConfig?.capabilities)
    ? displayConfig.capabilities
    : [];
  const contacts = Array.isArray(displayConfig?.contacts)
    ? displayConfig.contacts
    : [];
  const customContent = hasText(displayConfig?.customContent)
    ? displayConfig.customContent
    : '';
  const customContentHtml = useMemo(
    () => (customContent ? marked.parse(customContent) : ''),
    [customContent],
  );
  const heroHeadingProps = hasText(hero.title)
    ? { 'aria-labelledby': 'about-hero-title' }
    : {};
  const groupHeadingProps = hasText(group.title)
    ? { 'aria-labelledby': 'about-group-title' }
    : {};

  return (
    <main className='about-page'>
      <div className='about-page-shell'>
        <section className='about-hero' {...heroHeadingProps}>
          <div className='about-hero-copy'>
            {hasText(hero.eyebrow) && (
              <p className='about-eyebrow'>{hero.eyebrow}</p>
            )}
            {hasText(hero.title) && <h1 id='about-hero-title'>{hero.title}</h1>}
            {hasText(hero.subtitle) && (
              <p className='about-hero-subtitle'>{hero.subtitle}</p>
            )}
            <div className='about-actions' aria-label={t('页面操作')}>
              <AboutAction href={hero.primaryActionUrl}>
                {hero.primaryActionText}
              </AboutAction>
              <AboutAction href={hero.secondaryActionUrl} variant='secondary'>
                {hero.secondaryActionText}
              </AboutAction>
            </div>
          </div>

          <aside className='about-overview-panel' aria-label={t('业务概览')}>
            <div className='about-overview-heading'>
              <div>
                <span className='about-section-kicker'>{t('业务概览')}</span>
                {hasText(overview.title) && <h2>{overview.title}</h2>}
              </div>
              {hasText(overview.status) && (
                <span className='about-status'>
                  <CheckCircle2 aria-hidden='true' size={15} />
                  {overview.status}
                </span>
              )}
            </div>
            {hasText(overview.description) && (
              <p className='about-overview-description'>
                {overview.description}
              </p>
            )}
            {Array.isArray(overview.metrics) && overview.metrics.length > 0 && (
              <div className='about-metrics'>
                {overview.metrics.map((metric, index) => (
                  <div
                    className='about-metric'
                    key={`${metric.label}-${index}`}
                  >
                    <strong>{metric.value}</strong>
                    <span>{metric.label}</span>
                  </div>
                ))}
              </div>
            )}
            {Array.isArray(overview.channels) &&
              overview.channels.length > 0 && (
                <div className='about-channels'>
                  {overview.channels.map((channel, index) => (
                    <div
                      className='about-channel-row'
                      key={`${channel.name}-${index}`}
                    >
                      <div className='about-channel-label'>
                        <span>{channel.name}</span>
                        <span>{channel.status}</span>
                      </div>
                      <div
                        className='about-channel-track'
                        role='progressbar'
                        aria-valuemin='0'
                        aria-valuemax='100'
                        aria-valuenow={channel.value}
                        aria-label={channel.name}
                      >
                        <span style={{ width: `${channel.value}%` }} />
                      </div>
                    </div>
                  ))}
                </div>
              )}
          </aside>
        </section>

        {capabilities.length > 0 && (
          <section
            className='about-section'
            aria-labelledby='about-capability-title'
          >
            <div className='about-section-heading'>
              <span className='about-section-kicker'>{t('核心能力')}</span>
              <h2 id='about-capability-title'>{t('核心能力')}</h2>
            </div>
            <div className='about-capability-grid'>
              {capabilities.map((capability, index) => (
                <article
                  className='about-card'
                  key={`${capability.title}-${index}`}
                >
                  <div className='about-card-icon'>
                    {renderIcon(capability.icon, capabilityIconMap)}
                  </div>
                  <h3>{capability.title}</h3>
                  <p>{capability.description}</p>
                </article>
              ))}
            </div>
          </section>
        )}

        <section className='about-group-section' {...groupHeadingProps}>
          <div className='about-group-copy'>
            {hasText(group.status) && (
              <span className='about-section-kicker about-group-status'>
                {group.status}
              </span>
            )}
            {hasText(group.title) && (
              <h2 id='about-group-title'>{group.title}</h2>
            )}
            {hasText(group.description) && <p>{group.description}</p>}
          </div>
          {Array.isArray(group.bullets) && group.bullets.length > 0 && (
            <ul className='about-group-list'>
              {group.bullets.filter(hasText).map((item, index) => (
                <li key={`${item}-${index}`}>
                  <CheckCircle2 aria-hidden='true' size={18} />
                  <span>{item}</span>
                </li>
              ))}
            </ul>
          )}
          {hasText(group.websiteUrl) && (
            <AboutAction href={group.websiteUrl} variant='secondary'>
              {hasText(group.websiteLabel) ? group.websiteLabel : t('访问网站')}
            </AboutAction>
          )}
        </section>

        {contacts.length > 0 && (
          <section
            className='about-section'
            aria-labelledby='about-contact-title'
          >
            <div className='about-section-heading'>
              <span className='about-section-kicker'>{t('联系渠道')}</span>
              <h2 id='about-contact-title'>{t('联系方式')}</h2>
            </div>
            <div className='about-contact-grid'>
              {contacts.map((contact, index) => {
                const title = getContactTitle(contact, t);

                return (
                  <article
                    className='about-qr-card'
                    key={`${contact.type}-${contact.imageUrl}-${contact.fallbackUrl}-${index}`}
                  >
                    <div className='about-contact-icon'>
                      {renderIcon(contact.type, contactIconMap, MessageCircle)}
                    </div>
                    <div className='about-contact-copy'>
                      <h3>{title}</h3>
                      {hasText(contact.description) && (
                        <p>{contact.description}</p>
                      )}
                      <SafeLink
                        className='about-contact-fallback'
                        href={contact.fallbackUrl}
                      >
                        {t('备用联系链接')}
                      </SafeLink>
                    </div>
                    <div className='about-qr-frame'>
                      <QrImage contact={{ ...contact, title }} />
                    </div>
                  </article>
                );
              })}
            </div>
          </section>
        )}

        {customContentHtml && (
          <section
            className='about-custom-content'
            dangerouslySetInnerHTML={{ __html: customContentHtml }}
          />
        )}

        {protectedAttribution && (
          <footer className='about-attribution'>{protectedAttribution}</footer>
        )}
      </div>
    </main>
  );
};

export default AboutStructuredPage;
