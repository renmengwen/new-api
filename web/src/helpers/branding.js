const DEFAULT_LOGO_PATH = '/logo.png';

function getAssetVersion() {
  return import.meta.env?.VITE_REACT_APP_VERSION || '';
}

function isVersionableLogoUrl(url) {
  return !/^(?:[a-z][a-z\d+\-.]*:|\/\/)/i.test(url);
}

export function appendAssetVersion(url, assetVersion = getAssetVersion()) {
  if (typeof url !== 'string') {
    return url;
  }

  const trimmedUrl = url.trim();
  if (!trimmedUrl || !assetVersion || !isVersionableLogoUrl(trimmedUrl)) {
    return trimmedUrl;
  }

  const hashIndex = trimmedUrl.indexOf('#');
  const hash = hashIndex >= 0 ? trimmedUrl.slice(hashIndex) : '';
  const urlWithoutHash =
    hashIndex >= 0 ? trimmedUrl.slice(0, hashIndex) : trimmedUrl;

  const queryIndex = urlWithoutHash.indexOf('?');
  const pathname =
    queryIndex >= 0 ? urlWithoutHash.slice(0, queryIndex) : urlWithoutHash;
  const query = queryIndex >= 0 ? urlWithoutHash.slice(queryIndex + 1) : '';
  const params = new URLSearchParams(query);

  if (!params.has('v')) {
    params.set('v', assetVersion);
  }

  const nextQuery = params.toString();
  return `${pathname}${nextQuery ? `?${nextQuery}` : ''}${hash}`;
}

export function resolveBrandingState({
  storedSystemName,
  storedLogo,
  statusData,
  assetVersion = getAssetVersion(),
} = {}) {
  const systemName = statusData?.system_name || storedSystemName;
  const rawLogo = statusData?.logo || storedLogo || DEFAULT_LOGO_PATH;

  return {
    systemName,
    logo: appendAssetVersion(rawLogo, assetVersion) || rawLogo,
  };
}

export function applyDocumentBranding(doc, branding = {}) {
  if (!doc) {
    return;
  }

  const { systemName, logo } = branding;

  if (systemName) {
    doc.title = systemName;
  }

  if (!logo) {
    return;
  }

  let linkElement = doc.querySelector?.("link[rel~='icon']");
  if (!linkElement && doc.createElement && doc.head?.appendChild) {
    linkElement = doc.createElement('link');
    linkElement.rel = 'icon';
    doc.head.appendChild(linkElement);
  }

  if (linkElement) {
    linkElement.href = logo;
  }
}
