const BASE_URL = import.meta.env.VITE_API_URL ?? (import.meta.env.PROD ? "" : "http://localhost:8140");

export interface Release {
  productId: string;
  url: string;
  title: string;
  originalTitle?: string;
  releaseDate: string;
  releaseYear: number;
  productionYear: number;
  studio: string;
  runtime: string;
  rating: string;
  description: string;
  genres: string[];
  imageId?: string;
}

export async function fetchReleases(page: number): Promise<Release[]> {
  const resp = await fetch(`${BASE_URL}/api/releases?page=${page}`);
  if (!resp.ok) {
    throw new Error(`Failed to fetch releases: ${resp.status}`);
  }
  const data = await resp.json();
  return (data ?? []) as Release[];
}

export function coverImageUrl(id: string): string {
  return `${BASE_URL}/api/cover-image/${id}`;
}
