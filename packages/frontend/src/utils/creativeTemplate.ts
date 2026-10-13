import type { CreativeVariable } from '@/types/schemas'
// Replace original tokens once. A literal value containing another token must
// not be recursively expanded or treated as an expression.
export function previewCreativeTemplate(
  body: string,
  variables: CreativeVariable[],
  values: Record<string, string>,
) {
  const resolved = Object.fromEntries(variables.map((v) => [v.name, values[v.name] ?? v.default]))
  return body.replace(/\{([^{}]+)\}/g, (token, name: string) => resolved[name] ?? token)
}
