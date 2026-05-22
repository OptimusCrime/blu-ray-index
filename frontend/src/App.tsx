import { useState, useEffect, useCallback } from "react";
import { fetchReleases } from "./api";
import type { Release } from "./api";
import { ReleaseCard } from "./components/ReleaseCard";
import "./App.css";

export default function App() {
  const [releases, setReleases] = useState<Release[]>([]);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);

  const loadPage = useCallback(async (pageNum: number) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchReleases(pageNum);
      if (pageNum === 0) {
        setReleases(data);
      } else {
        setReleases((prev) => [...prev, ...data]);
      }
      if (data.length === 0) {
        setHasMore(false);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPage(0);
  }, [loadPage]);

  const loadMore = () => {
    const nextPage = page + 1;
    setPage(nextPage);
    loadPage(nextPage);
  };

  return (
    <div className="app">
      <header className="app-header">
        <h1>Blu-ray Releases</h1>
      </header>

      <main className="releases-container">
        {error && (
          <div className="error-banner">
            <p>{error}</p>
            <button onClick={() => loadPage(page)}>Retry</button>
          </div>
        )}

        {releases.length === 0 && !loading && !error && (
          <p className="empty-state">No releases found.</p>
        )}

        <div className="releases-list">
          {releases.map((release) => (
            <ReleaseCard key={release.productId} release={release} />
          ))}
        </div>

        {loading && (
          <div className="loading-indicator">
            <p>Loading releases…</p>
          </div>
        )}

        {!loading && hasMore && releases.length > 0 && (
          <div className="load-more-container">
            <button className="load-more-btn" onClick={loadMore}>
              Load more
            </button>
          </div>
        )}

        {!hasMore && releases.length > 0 && (
          <p className="end-of-list">No more releases.</p>
        )}
      </main>
    </div>
  );
}
