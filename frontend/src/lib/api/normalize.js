export function responseField(response, name, fallback) {
  const alternate = name.charAt(0).toLowerCase() + name.slice(1);
  return response?.[name] ?? response?.[alternate] ?? fallback;
}
