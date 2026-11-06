/**
 * Focus Query utilities for client-side boolean search token extraction and query helpers.
 */

export const QUERY_OPERATOR_CHIPS = [
  { label: "+ AND", insert: " AND " },
  { label: "+ OR", insert: " OR " },
  { label: "+ NOT", insert: " NOT " },
  { label: '+ "phrase"', insert: ' ""' },
  { label: "+ ( )", insert: " ()" },
] as const;

/**
 * Extracts non-negated search terms from a boolean focus query string.
 * This is used for keyword badging on article cards and reader views.
 */
export function extractPositiveTerms(query: string): string[] {
  if (!query || !query.trim()) return [];

  const terms: string[] = [];
  // Matches quoted phrases, operators/delimiters, or standalone words
  const regex = /"([^"]+)"|(\b(?:AND|OR|NOT)\b|&&|\|\||[()!,-])|([^\s()!,-]+)/gi;

  let match: RegExpExecArray | null;
  let nextNegated = false;

  while ((match = regex.exec(query)) !== null) {
    const phrase = match[1];
    const op = match[2];
    const word = match[3];

    if (op) {
      const upper = op.toUpperCase();
      if (upper === "NOT" || upper === "!" || upper === "-") {
        nextNegated = true;
      }
      continue;
    }

    if (phrase !== undefined) {
      if (!nextNegated) {
        const trimmed = phrase.trim();
        if (trimmed) terms.push(trimmed);
      }
      nextNegated = false;
      continue;
    }

    if (word !== undefined) {
      let cleanWord = word.trim();
      // Strip field prefix if present (e.g. title:nvidia -> nvidia)
      if (cleanWord.includes(":")) {
        const parts = cleanWord.split(":");
        cleanWord = parts.slice(1).join(":");
      }
      if (!nextNegated && cleanWord) {
        terms.push(cleanWord);
      }
      nextNegated = false;
    }
  }

  // Deduplicate case-insensitively while preserving original casing
  const seen = new Set<string>();
  const unique: string[] = [];
  for (const t of terms) {
    const lower = t.toLowerCase();
    if (!seen.has(lower)) {
      seen.add(lower);
      unique.push(t);
    }
  }
  return unique;
}
