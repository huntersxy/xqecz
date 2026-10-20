import type { CreativeVariable } from '@/types/schemas'
export function previewCreativeTemplate(
  body: string,
  variables: CreativeVariable[],
  values: Record<string, string>,
) {
  const resolved = Object.fromEntries(variables.map((v) => [v.name, values[v.name] ?? v.default]))
  return body.replace(/\{([^{}]+)\}/g, (token, name: string) => resolved[name] ?? token)
}
