/** Matches the backend's postgresIdentifierRe: a safe, unquoted Postgres identifier. */
export function isValidIdentifier(value: string): boolean {
  return /^[a-zA-Z_][a-zA-Z0-9_]{0,62}$/.test(value)
}
