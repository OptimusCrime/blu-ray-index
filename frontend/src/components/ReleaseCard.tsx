import { useState } from "react";
import type { Release } from "../api";
import { coverImageUrl } from "../api";

interface Props {
  release: Release;
}

export function ReleaseCard({ release }: Props) {
  const [imgError, setImgError] = useState(false);

  return (
    <article className="release-card">
      <a href={release.url} target="_blank" rel="noopener noreferrer" className="cover-link">
        {release.imageId && !imgError ? (
          <img
            src={coverImageUrl(release.imageId)}
            alt={release.title}
            className="cover-image"
            loading="lazy"
            onError={() => setImgError(true)}
          />
        ) : (
          <div className="cover-placeholder" />
        )}
      </a>

      <div className="release-info">
        <a href={release.url} target="_blank" rel="noopener noreferrer">
          <h2 className="release-title">{release.title}</h2>
        </a>
        {release.originalTitle && (
          <p className="original-title">{release.originalTitle}</p>
        )}

        <div className="release-meta">
          {release.studio && <span>{release.studio}</span>}
          {release.releaseYear > 0 && <span>{release.releaseYear}</span>}
          {release.runtime && <span>{release.runtime}</span>}
          {release.rating && <span>{release.rating}</span>}
          {release.releaseDate && (
            <span className="release-date">{release.releaseDate}</span>
          )}
        </div>

        {release.genres.length > 0 && (
          <div className="genres">
            {release.genres.map((g) => (
              <span key={g} className="genre-tag">
                {g}
              </span>
            ))}
          </div>
        )}

        {release.description && (
          <p className="description">{release.description}</p>
        )}
      </div>
    </article>
  );
}
