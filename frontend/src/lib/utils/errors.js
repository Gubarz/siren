export function errorMessage(error, prefix = '') {
  const message = error?.message ?? String(error);
  return `${prefix}${message}`;
}
