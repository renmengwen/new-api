export const isTaskResultPreviewUrl = (value) => {
  if (typeof value !== 'string') {
    return false;
  }
  const url = value.trim();
  if (!url) {
    return false;
  }
  return (
    /^https?:\/\//i.test(url) ||
    url.startsWith('/v1/images/generations/') ||
    url.startsWith('/v1/videos/')
  );
};
