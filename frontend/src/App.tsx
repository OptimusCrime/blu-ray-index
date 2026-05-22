import { useReleases } from "./hooks/useReleases";
import { ReleaseCard } from "./components/ReleaseCard";
import "./App.css";

export default function App() {
  const { releases, loading, error, hasMore, loadMore, retry } = useReleases();

  return (
    <div className="app">
      <header className="app-header">
        <h1>Blu-ray Releases</h1>
      </header>

      <main className="releases-container">
        {error && (
          <div className="error-banner">
            <p>{error}</p>
            <button onClick={retry}>Retry</button>
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
