export const formatFailValues = (failValue) => {
  if (Array.isArray(failValue)) {
    return failValue.length > 0 ? failValue.join(', ') : '-';
  }

  if (failValue === undefined || failValue === null || failValue === '') {
    return '-';
  }

  return String(failValue);
};

export const formatFailWhen = (failWhen) => {
  if (!failWhen) {
    return '-';
  }

  return String(failWhen);
};
