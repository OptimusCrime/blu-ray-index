import { useState, useEffect } from "react";
import { fetchReleases } from "../api";
import type { Release } from "../api";

interface UseReleasesResult {
  releases: Release[];
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  loadMore: () => void;
  retry: () => void;
}

export function useReleases(): UseReleasesResult {
  const [releases, setReleases] = useState<Release[]>([]);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);

  function load(targetPage: number, append: boolean) {
    setLoading(true);
    setError(null);
    fetchReleases(targetPage)
      .then((data) => {
        setReleases((prev) => (append ? [...prev, ...data] : data));
        if (data.length === 0) setHasMore(false);
      })
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : "Unknown error");
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    load(0, false);
  }, []);

  function loadMore() {
    const nextPage = page + 1;
    setPage(nextPage);
    load(nextPage, true);
  }

  function retry() {
    load(page, page > 0);
  }

  return { releases, loading, error, hasMore, loadMore, retry };
}
