export type SearchableItem = {
  name: string;
  sku: string | null;
  category: string;
};

function normalize(value: string) {
  return value.trim().toLowerCase();
}

export function matchesItemSearch(item: SearchableItem, query: string) {
  const term = normalize(query);
  if (!term) return true;

  return (
    item.name.toLowerCase().includes(term) ||
    item.category.toLowerCase().includes(term) ||
    (item.sku ? item.sku.toLowerCase().includes(term) : false)
  );
}

function matchScore(item: SearchableItem, term: string) {
  const name = item.name.toLowerCase();
  const sku = item.sku ? item.sku.toLowerCase() : "";
  const category = item.category.toLowerCase();

  if (sku && sku === term) return 100;
  if (name === term) return 90;
  if (name.startsWith(term)) return 80;
  if (sku && sku.startsWith(term)) return 70;
  if (name.includes(term)) return 60;
  if (sku && sku.includes(term)) return 50;
  if (category.startsWith(term)) return 40;
  if (category.includes(term)) return 30;
  return 0;
}

/**
 * Picks the most relevant item for a search term: exact SKU, exact name,
 * name prefix, SKU prefix, then the first partial match.
 */
export function findBestItemMatch<T extends SearchableItem>(
  items: T[],
  query: string,
): T | null {
  const term = normalize(query);
  if (!term) return null;

  let best: T | null = null;
  let bestScore = 0;

  for (const item of items) {
    const score = matchScore(item, term);
    if (score > bestScore) {
      best = item;
      bestScore = score;
    }
  }

  return best;
}
