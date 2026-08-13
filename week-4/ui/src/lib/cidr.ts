/** Basic IPv4 CIDR validation: shape, octet range (0-255), and prefix range (0-32). */
export function isValidCidr(cidr: string): boolean {
  const match = cidr.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/)
  if (!match) return false
  const octets = match.slice(1, 5).map(Number)
  const prefix = Number(match[5])
  return octets.every((o) => o >= 0 && o <= 255) && prefix >= 0 && prefix <= 32
}
