export function decodeBase64(input: string): string {
  try {
    return atob(input);
  } catch (e) {
    console.error('Error decoding Base64 string:', e);
    return '';
  }
}
