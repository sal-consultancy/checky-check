const variablePattern = /\$\{([a-zA-Z_][a-zA-Z0-9_.-]*)\}/g;

export const resolveTemplateValue = (template, vars) => {
  if (typeof template !== 'string') {
    return template;
  }

  return template.replace(variablePattern, (match, key) => {
    if (Object.prototype.hasOwnProperty.call(vars || {}, key)) {
      return vars[key];
    }

    return match;
  });
};

export const formatFailValues = (failValue, vars = {}) => {
  if (Array.isArray(failValue)) {
    const values = failValue
      .map((value) => resolveTemplateValue(value, vars))
      .filter((value) => value !== undefined && value !== null && value !== '');
    return values.length > 0 ? values.join(', ') : '-';
  }

  if (failValue === undefined || failValue === null || failValue === '') {
    return '-';
  }

  const resolvedValue = resolveTemplateValue(failValue, vars);
  if (resolvedValue === undefined || resolvedValue === null || resolvedValue === '') {
    return '-';
  }

  return String(resolvedValue);
};

export const formatFailWhen = (failWhen) => {
  if (!failWhen) {
    return '-';
  }

  return String(failWhen);
};
