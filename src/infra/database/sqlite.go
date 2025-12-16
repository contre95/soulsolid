package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/metrics"
	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SqliteLibrary is a SQLite implementation of the Library interface.
type SqliteLibrary struct {
	db *sql.DB
}

// NewSqliteLibrary creates a new SqliteLibrary.
func NewSqliteLibrary(path string) (*SqliteLibrary, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &SqliteLibrary{db: db}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS artists (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			sort_name TEXT
		);
		
		CREATE TABLE IF NOT EXISTS albums (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			type TEXT,
			release_date TEXT,
			release_group_id TEXT,
			label TEXT,
			catalog_number TEXT,
			country TEXT,
			status TEXT,
			barcode TEXT,
			added_date TEXT,
			modified_date TEXT
		);
		
		CREATE TABLE IF NOT EXISTS tracks (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			title_version TEXT,
			duration INTEGER,
			track_number INTEGER,
			disc_number INTEGER,
			isrc TEXT,
			chromaprint_fingerprint TEXT,
			bitrate INTEGER,
			format TEXT,
			sample_rate INTEGER,
			bit_depth INTEGER,
			channels INTEGER,
			explicit_content BOOLEAN DEFAULT FALSE,
			preview_url TEXT,
			composer TEXT,
			genre TEXT,
			year INTEGER,
			original_year INTEGER,
			lyrics TEXT,
			explicit_lyrics BOOLEAN DEFAULT FALSE,
			bpm REAL,
			gain REAL,
			source TEXT,
			source_url TEXT,
			added_date TEXT,
			modified_date TEXT
		);
		
		CREATE TABLE IF NOT EXISTS track_artists (
			track_id TEXT,
			artist_id TEXT,
			role TEXT,
			PRIMARY KEY (track_id, artist_id, role),
			FOREIGN KEY (track_id) REFERENCES tracks(id),
			FOREIGN KEY (artist_id) REFERENCES artists(id)
		);
		
		CREATE TABLE IF NOT EXISTS album_artists (
			album_id TEXT,
			artist_id TEXT,
			role TEXT,
			PRIMARY KEY (album_id, artist_id, role),
			FOREIGN KEY (album_id) REFERENCES albums(id),
			FOREIGN KEY (artist_id) REFERENCES artists(id)
		);
		
		CREATE TABLE IF NOT EXISTS track_albums (
			track_id TEXT PRIMARY KEY,
			album_id TEXT,
			FOREIGN KEY (track_id) REFERENCES tracks(id),
			FOREIGN KEY (album_id) REFERENCES albums(id)
		);
		
		CREATE TABLE IF NOT EXISTS track_attributes (
			id INTEGER PRIMARY KEY,
			track_id TEXT,
			key TEXT NOT NULL,
			value TEXT,
			UNIQUE(track_id, key) ON CONFLICT REPLACE,
			FOREIGN KEY (track_id) REFERENCES tracks(id)
		);
		
		CREATE TABLE IF NOT EXISTS album_attributes (
			id INTEGER PRIMARY KEY,
			album_id TEXT,
			key TEXT NOT NULL,
			value TEXT,
			UNIQUE(album_id, key) ON CONFLICT REPLACE,
			FOREIGN KEY (album_id) REFERENCES albums(id)
		);
		
		CREATE TABLE IF NOT EXISTS artist_attributes (
			id INTEGER PRIMARY KEY,
			artist_id TEXT,
			key TEXT NOT NULL,
			value TEXT,
			UNIQUE(artist_id, key) ON CONFLICT REPLACE,
			FOREIGN KEY (artist_id) REFERENCES artists(id)
		);

		CREATE TABLE IF NOT EXISTS library_metrics (
			id INTEGER PRIMARY KEY,
			metric_type TEXT NOT NULL,
			metric_key TEXT,
			metric_value INTEGER,
			updated_at TEXT,
			UNIQUE(metric_type, metric_key)
		);

		CREATE INDEX IF NOT EXISTS idx_track_artists_track ON track_artists(track_id);
		CREATE INDEX IF NOT EXISTS idx_track_artists_artist ON track_artists(artist_id);
		CREATE INDEX IF NOT EXISTS idx_album_artists_album ON album_artists(album_id);
		CREATE INDEX IF NOT EXISTS idx_album_artists_artist ON album_artists(artist_id);
		CREATE INDEX IF NOT EXISTS idx_track_attributes_track ON track_attributes(track_id);
		CREATE INDEX IF NOT EXISTS idx_album_attributes_album ON album_attributes(album_id);
		CREATE INDEX IF NOT EXISTS idx_artist_attributes_artist ON artist_attributes(artist_id);
	`)
	if err != nil {
		return err
	}

	// Add new columns to existing tables if they don't exist
	_, err = db.Exec(`
		ALTER TABLE tracks ADD COLUMN source TEXT;
		ALTER TABLE tracks ADD COLUMN source_url TEXT;
	`)
	// Ignore errors if columns already exist
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}

	return nil
}

// AddTrack adds a track to the database.
func (d *SqliteLibrary) AddTrack(ctx context.Context, track *music.Track) error {
	// Validate track using domain validation
	if err := track.Validate(); err != nil {
		slog.Error("AddTrack: validation failed", "error", err, "trackID", track.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert track
	_, err = tx.ExecContext(ctx, `
    INSERT INTO tracks (id, path, title, title_version, duration, track_number, disc_number,
      isrc, chromaprint_fingerprint, bitrate, format, sample_rate, bit_depth, channels,
      explicit_content, preview_url, composer, genre, year, original_year, lyrics,
      explicit_lyrics, bpm, gain, source, source_url, added_date, modified_date)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  `, track.ID, track.Path, track.Title, track.TitleVersion, track.Metadata.Duration, track.Metadata.TrackNumber, track.Metadata.DiscNumber,
		track.ISRC, track.ChromaprintFingerprint, track.Bitrate, track.Format, track.SampleRate, track.BitDepth, track.Channels,
		track.ExplicitContent, track.PreviewURL, track.Metadata.Composer, track.Metadata.Genre, track.Metadata.Year,
		track.Metadata.OriginalYear, track.Metadata.Lyrics, track.Metadata.ExplicitLyrics, track.Metadata.BPM, track.Metadata.Gain,
		track.MetadataSource.Source, track.MetadataSource.MetadataSourceURL, track.AddedDate.Format(time.RFC3339), track.ModifiedDate.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Insert track-artist relationships
	for _, artistRole := range track.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_artists (track_id, artist_id, role)
			VALUES (?, ?, ?)
		`, track.ID, artistRole.Artist.ID, artistRole.Role)
		if err != nil {
			return err
		}
	}

	// Insert track-album relationship
	if track.Album != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_albums (track_id, album_id)
			VALUES (?, ?)
		`, track.ID, track.Album.ID)
		if err != nil {
			return err
		}
	}

	// Insert track attributes
	for key, value := range track.Attributes {
		_, err = tx.ExecContext(ctx, `
      INSERT INTO track_attributes (track_id, key, value)
      VALUES (?, ?, ?)
    `, track.ID, key, value)
		if err != nil {
			return err
		}
	}

	// Insert AcoustID as attribute if present
	if acoustID, exists := track.Attributes["acoustid"]; exists && acoustID != "" {
		_, err = tx.ExecContext(ctx, `
      INSERT INTO track_attributes (track_id, key, value)
      VALUES (?, ?, ?)
    `, track.ID, "acoustid", acoustID)
		if err != nil {
			return err
		}
	}

	// Insert MusicBrainz ID as attribute if present
	if track.Attributes != nil {
		if musicBrainzID, exists := track.Attributes["musicbrainz_id"]; exists && musicBrainzID != "" {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO track_attributes (track_id, key, value)
				VALUES (?, ?, ?)
			`, track.ID, "musicbrainz_id", musicBrainzID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// UpdateAlbum updates an album in the database.
func (d *SqliteLibrary) UpdateAlbum(ctx context.Context, album *music.Album) error {
	// Validate album using domain validation
	if err := album.Validate(); err != nil {
		slog.Error("UpdateAlbum: validation failed", "error", err, "albumID", album.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update album
	_, err = tx.ExecContext(ctx, `
		UPDATE albums
		SET title = ?, type = ?, release_date = ?, release_group_id = ?,
			label = ?, catalog_number = ?, country = ?, status = ?, barcode = ?, modified_date = ?
		WHERE id = ?
	`, album.Title, string(album.Type), album.ReleaseDate.Format(time.RFC3339),
		album.ReleaseGroupID, album.Label, album.CatalogNumber,
		album.Country, album.Status, album.Barcode, album.ModifiedDate.Format(time.RFC3339), album.ID)
	if err != nil {
		return err
	}

	// Delete existing album-artist relationships
	_, err = tx.ExecContext(ctx, `DELETE FROM album_artists WHERE album_id = ?`, album.ID)
	if err != nil {
		return err
	}

	// Insert new album-artist relationships
	for _, artistRole := range album.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO album_artists (album_id, artist_id, role)
			VALUES (?, ?, ?)
		`, album.ID, artistRole.Artist.ID, artistRole.Role)
		if err != nil {
			return err
		}
	}

	// Update album attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM album_attributes WHERE album_id = ?`, album.ID)
	if err != nil {
		return err
	}

	for key, value := range album.Attributes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO album_attributes (album_id, key, value)
			VALUES (?, ?, ?)
		`, album.ID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteAlbum deletes an album from the database and all its associated tracks.
func (d *SqliteLibrary) DeleteAlbum(ctx context.Context, id string) error {
	slog.Debug("DeleteAlbum called", "albumID", id)

	// First, find all tracks associated with this album
	rows, err := d.db.QueryContext(ctx, `
		SELECT t.id, t.path
		FROM tracks t
		INNER JOIN track_albums ta ON t.id = ta.track_id
		WHERE ta.album_id = ?
	`, id)
	if err != nil {
		return err
	}
	defer rows.Close()

	var trackIDs []string
	var trackPaths []string
	for rows.Next() {
		var trackID, trackPath string
		if err := rows.Scan(&trackID, &trackPath); err != nil {
			return err
		}
		trackIDs = append(trackIDs, trackID)
		trackPaths = append(trackPaths, trackPath)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all tracks associated with this album
	for _, trackID := range trackIDs {
		// Delete track attributes
		_, err = tx.ExecContext(ctx, `DELETE FROM track_attributes WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track artists
		_, err = tx.ExecContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track albums
		_, err = tx.ExecContext(ctx, `DELETE FROM track_albums WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track
		_, err = tx.ExecContext(ctx, `DELETE FROM tracks WHERE id = ?`, trackID)
		if err != nil {
			return err
		}
	}

	// Delete album attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM album_attributes WHERE album_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete album artists
	_, err = tx.ExecContext(ctx, `DELETE FROM album_artists WHERE album_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete album
	_, err = tx.ExecContext(ctx, `DELETE FROM albums WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// Commit the database transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Delete track files from filesystem
	for _, path := range trackPaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to delete track file from filesystem", "path", path, "error", err)
			// Don't return error here - database deletion succeeded, file deletion is secondary
		} else {
			slog.Debug("Successfully deleted track file from filesystem", "path", path)
		}
	}

	return nil
}

// GetTrack gets a track from the database.
func (d *SqliteLibrary) GetTrack(ctx context.Context, id string) (*music.Track, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get track basic info
	row := tx.QueryRowContext(ctx, `
    SELECT id, path, title, title_version, duration, track_number, disc_number,
      isrc, chromaprint_fingerprint, bitrate, format, sample_rate, bit_depth, channels, explicit_content,
      preview_url, composer, genre, year, original_year, lyrics, explicit_lyrics,
      bpm, gain, source, source_url, added_date, modified_date
    FROM tracks
    WHERE id = ?
  `, id)

	track := &music.Track{}
	var addedDateStr, modifiedDateStr string
	var sourceNull, sourceURLNull sql.NullString

	err = row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.ChromaprintFingerprint, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &sourceNull, &sourceURLNull, &addedDateStr, &modifiedDateStr)

	// Handle nullable source fields
	track.MetadataSource.Source = sourceNull.String
	track.MetadataSource.MetadataSourceURL = sourceURLNull.String
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	track.AddedDate, _ = time.Parse(time.RFC3339, addedDateStr)
	track.ModifiedDate, _ = time.Parse(time.RFC3339, modifiedDateStr)

	// Get track artists
	rows, err := d.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.sort_name, ta.role
		FROM track_artists ta
		JOIN artists a ON ta.artist_id = a.id
		WHERE ta.track_id = ?
	`, track.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var artist music.Artist
		var role string
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &role)
		if err != nil {
			return nil, err
		}

		// Load artist attributes
		artistAttrRows, err := d.db.QueryContext(ctx, `
			SELECT key, value FROM artist_attributes WHERE artist_id = ?
		`, artist.ID)
		if err != nil {
			return nil, err
		}

		artist.Attributes = make(map[string]string)
		for artistAttrRows.Next() {
			var key, value string
			if err := artistAttrRows.Scan(&key, &value); err != nil {
				artistAttrRows.Close()
				return nil, err
			}
			artist.Attributes[key] = value
		}
		artistAttrRows.Close()

		track.Artists = append(track.Artists, music.ArtistRole{Artist: &artist, Role: role})
	}

	// Get track album
	var albumID string
	err = tx.QueryRowContext(ctx, `SELECT album_id FROM track_albums WHERE track_id = ?`, id).Scan(&albumID)
	if err == nil {
		album, err := d.GetAlbum(ctx, albumID)
		if err != nil {
			return nil, err
		}
		track.Album = album
	}

	// Get track attributes
	attrRows, err := tx.QueryContext(ctx, `SELECT key, value FROM track_attributes WHERE track_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	track.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		err := attrRows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}
		track.Attributes[key] = value
	}

	return track, nil
}

// GetGenreDistribution returns the distribution of tracks by genre.
func (d *SqliteLibrary) GetGenreDistribution(ctx context.Context) (map[string]int, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(genre, 'Unknown') as genre, COUNT(*) as count
		FROM tracks
		GROUP BY genre
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var genre string
		var count int
		if err := rows.Scan(&genre, &count); err != nil {
			return nil, err
		}
		distribution[genre] = count
	}

	return distribution, rows.Err()
}

// GetMetadataCompleteness returns statistics about metadata completeness.
func (d *SqliteLibrary) GetMetadataCompleteness(ctx context.Context) (metrics.MetadataCompletenessStats, error) {
	var stats metrics.MetadataCompletenessStats

	// Count tracks with complete metadata (title, artist, album, genre, year)
	err := d.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tracks t
		WHERE t.title != ''
			AND EXISTS (SELECT 1 FROM track_artists ta WHERE ta.track_id = t.id)
			AND EXISTS (SELECT 1 FROM track_albums ta2 WHERE ta2.track_id = t.id)
			AND t.genre IS NOT NULL AND t.genre != ''
			AND t.year > 0
	`).Scan(&stats.Complete)
	if err != nil {
		return stats, err
	}

	// Count tracks missing genre
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE genre IS NULL OR genre = ''").Scan(&stats.MissingGenre)
	if err != nil {
		return stats, err
	}

	// Count tracks missing year
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE year IS NULL OR year = 0").Scan(&stats.MissingYear)
	if err != nil {
		return stats, err
	}

	// Count tracks missing lyrics
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE lyrics IS NULL OR lyrics = ''").Scan(&stats.MissingLyrics)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

// GetTotalTracks returns the total number of tracks in the library.
func (d *SqliteLibrary) GetTotalTracks(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks").Scan(&count)
	return count, err
}

// GetTotalArtists returns the total number of artists in the library.
func (d *SqliteLibrary) GetTotalArtists(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM artists WHERE name != '' AND name IS NOT NULL").Scan(&count)
	return count, err
}

// GetTotalAlbums returns the total number of albums in the library.
func (d *SqliteLibrary) GetTotalAlbums(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM albums").Scan(&count)
	return count, err
}

// GetFormatDistribution returns the distribution of tracks by audio format.
func (d *SqliteLibrary) GetFormatDistribution(ctx context.Context) (map[string]int, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT COALESCE(format, 'Unknown') as format, COUNT(*) as count
		FROM tracks
		GROUP BY format
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var format string
		var count int
		if err := rows.Scan(&format, &count); err != nil {
			return nil, err
		}
		distribution[format] = count
	}

	return distribution, rows.Err()
}

// GetYearDistribution returns the distribution of tracks by release year.
func (d *SqliteLibrary) GetYearDistribution(ctx context.Context) (map[string]int, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT year, COUNT(*) as count
		FROM tracks
		WHERE year > 0
		GROUP BY year
		ORDER BY year DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var year int
		var count int
		if err := rows.Scan(&year, &count); err != nil {
			return nil, err
		}
		distribution[fmt.Sprintf("%d", year)] = count
	}

	return distribution, rows.Err()
}

// GetLyricsStats returns statistics about lyrics presence.
func (d *SqliteLibrary) GetLyricsStats(ctx context.Context) (metrics.LyricsStats, error) {
	var stats metrics.LyricsStats

	// Count tracks with lyrics
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE lyrics IS NOT NULL AND lyrics != ''").Scan(&stats.WithLyrics)
	if err != nil {
		return stats, err
	}

	// Count tracks without lyrics
	err = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE lyrics IS NULL OR lyrics = ''").Scan(&stats.WithoutLyrics)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

// GetTracksWithISRC returns the number of tracks that have an ISRC.
func (d *SqliteLibrary) GetTracksWithISRC(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE isrc IS NOT NULL AND isrc != ''").Scan(&count)
	return count, err
}

// GetTracksWithValidBPM returns the number of tracks that have a BPM != 0.
func (d *SqliteLibrary) GetTracksWithValidBPM(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE bpm IS NOT NULL AND bpm != 0").Scan(&count)
	return count, err
}

// GetTracksWithValidYear returns the number of tracks that have a valid year (>1000 <3000).
func (d *SqliteLibrary) GetTracksWithValidYear(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE year > 1000 AND year < 3000").Scan(&count)
	return count, err
}

// GetTracksWithValidGenre returns the number of tracks that have a genre not Unknown and not empty.
func (d *SqliteLibrary) GetTracksWithValidGenre(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks WHERE genre IS NOT NULL AND genre != '' AND LOWER(genre) != 'unknown'").Scan(&count)
	return count, err
}

// StoreMetric stores a metric in the database.
func (d *SqliteLibrary) StoreMetric(ctx context.Context, metricType, key string, value int) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO library_metrics (metric_type, metric_key, metric_value, updated_at)
		VALUES (?, ?, ?, datetime('now'))
	`, metricType, key, value)
	return err
}

// GetStoredMetrics retrieves stored metrics of a specific type.
func (d *SqliteLibrary) GetStoredMetrics(ctx context.Context, metricType string) ([]metrics.StoredMetric, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT metric_key, metric_value
		FROM library_metrics
		WHERE metric_type = ?
		ORDER BY metric_value DESC
	`, metricType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var storedMetrics []metrics.StoredMetric
	for rows.Next() {
		var m metrics.StoredMetric
		m.Type = metricType
		if err := rows.Scan(&m.Key, &m.Value); err != nil {
			return nil, err
		}
		storedMetrics = append(storedMetrics, m)
	}

	return storedMetrics, rows.Err()
}

// ClearStoredMetrics removes all stored metrics.
func (d *SqliteLibrary) ClearStoredMetrics(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM library_metrics")
	return err
}

// UpdateTrack updates a track in the database.
func (d *SqliteLibrary) UpdateTrack(ctx context.Context, track *music.Track) error {
	// Validate track using domain validation
	if err := track.Validate(); err != nil {
		slog.Error("UpdateTrack: validation failed", "error", err, "trackID", track.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update track
	_, err = tx.ExecContext(ctx, `
    UPDATE tracks
    SET path = ?, title = ?, title_version = ?, duration = ?, track_number = ?, disc_number = ?,
      isrc = ?, bitrate = ?, format = ?, chromaprint_fingerprint = ?, sample_rate = ?, bit_depth = ?, channels = ?,
      explicit_content = ?, preview_url = ?, composer = ?, genre = ?, year = ?,
      original_year = ?, lyrics = ?, explicit_lyrics = ?, bpm = ?, gain = ?, source = ?, source_url = ?, modified_date = ?
    WHERE id = ?
  `, track.Path, track.Title, track.TitleVersion, track.Metadata.Duration, track.Metadata.TrackNumber, track.Metadata.DiscNumber,
		track.ISRC, track.Bitrate, track.Format, track.ChromaprintFingerprint, track.SampleRate, track.BitDepth, track.Channels,
		track.ExplicitContent, track.PreviewURL, track.Metadata.Composer, track.Metadata.Genre, track.Metadata.Year,
		track.Metadata.OriginalYear, track.Metadata.Lyrics, track.Metadata.ExplicitLyrics, track.Metadata.BPM, track.Metadata.Gain,
		track.MetadataSource.Source, track.MetadataSource.MetadataSourceURL, track.ModifiedDate.Format(time.RFC3339), track.ID)
	if err != nil {
		return err
	}

	// Delete existing track artists
	_, err = tx.ExecContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`, track.ID)
	if err != nil {
		return err
	}

	// Insert new track artists
	for _, artistRole := range track.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_artists (track_id, artist_id, role)
			VALUES (?, ?, ?)
		`, track.ID, artistRole.Artist.ID, artistRole.Role)
		if err != nil {
			return err
		}
	}

	// Update track-album relationship
	_, err = tx.ExecContext(ctx, `DELETE FROM track_albums WHERE track_id = ?`, track.ID)
	if err != nil {
		return err
	}

	if track.Album != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_albums (track_id, album_id)
			VALUES (?, ?)
		`, track.ID, track.Album.ID)
		if err != nil {
			return err
		}
	}

	// Update track attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM track_attributes WHERE track_id = ?`, track.ID)
	if err != nil {
		return err
	}

	for key, value := range track.Attributes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO track_attributes (track_id, key, value)
			VALUES (?, ?, ?)
		`, track.ID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteTrack deletes a track from the database and filesystem.
func (d *SqliteLibrary) DeleteTrack(ctx context.Context, id string) error {
	slog.Debug("DeleteTrack called", "trackID", id)

	// First get the track path before deleting from database
	var path string
	err := d.db.QueryRowContext(ctx, `SELECT path FROM tracks WHERE id = ?`, id).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("track not found: %s", id)
		}
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete track attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM track_attributes WHERE track_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete track artists
	_, err = tx.ExecContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete track albums
	_, err = tx.ExecContext(ctx, `DELETE FROM track_albums WHERE track_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete track
	_, err = tx.ExecContext(ctx, `DELETE FROM tracks WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// Commit the database transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Delete the file from filesystem
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to delete track file from filesystem", "path", path, "error", err)
		// Don't return error here - database deletion succeeded, file deletion is secondary
	} else {
		slog.Debug("Successfully deleted track file from filesystem", "path", path)
	}

	return nil
}

// AddAlbum adds an album to the database.
func (d *SqliteLibrary) AddAlbum(ctx context.Context, album *music.Album) error {
	// Validate album using domain validation
	if err := album.Validate(); err != nil {
		slog.Error("AddAlbum: validation failed", "error", err, "albumID", album.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert album
	_, err = tx.ExecContext(ctx, `
		INSERT INTO albums (id, title, type, release_date, release_group_id,
			label, catalog_number, country, status, barcode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, album.ID, album.Title, string(album.Type), album.ReleaseDate.Format(time.RFC3339),
		album.ReleaseGroupID, album.Label, album.CatalogNumber,
		album.Country, album.Status, album.Barcode)
	if err != nil {
		return err
	}

	// Insert album-artist relationships
	for _, artistRole := range album.Artists {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO album_artists (album_id, artist_id, role)
			VALUES (?, ?, ?)
		`, album.ID, artistRole.Artist.ID, artistRole.Role)
		if err != nil {
			return err
		}
	}

	// Insert album attributes
	for key, value := range album.Attributes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO album_attributes (album_id, key, value)
			VALUES (?, ?, ?)
		`, album.ID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAlbum gets an album from the database.
func (d *SqliteLibrary) GetAlbum(ctx context.Context, id string) (*music.Album, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get album basic info
	row := tx.QueryRowContext(ctx, `
		SELECT id, title, type, release_date, release_group_id,
			label, catalog_number, country, status, barcode
		FROM albums
		WHERE id = ?
	`, id)

	album := &music.Album{}
	var releaseDateStr string
	var albumType string

	err = row.Scan(&album.ID, &album.Title, &albumType, &releaseDateStr,
		&album.ReleaseGroupID, &album.Label, &album.CatalogNumber, &album.Country,
		&album.Status, &album.Barcode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	album.Type = music.AlbumType(albumType)
	album.ReleaseDate, _ = time.Parse(time.RFC3339, releaseDateStr)

	// Get album attributes
	attrRows, err := tx.QueryContext(ctx, `SELECT key, value FROM album_attributes WHERE album_id = ?`, id)
	if err != nil {
		return nil, err
	}

	album.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		if err := attrRows.Scan(&key, &value); err != nil {
			attrRows.Close()
			return nil, err
		}
		album.Attributes[key] = value
	}
	attrRows.Close()

	// Get album artists
	rows, err := tx.QueryContext(ctx, `
		SELECT a.id, a.name, a.sort_name, aa.role
		FROM album_artists aa
		JOIN artists a ON aa.artist_id = a.id
		WHERE aa.album_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var artist music.Artist
		var role string
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &role)
		if err != nil {
			return nil, err
		}

		// Load artist attributes
		artistAttrRows, err := tx.QueryContext(ctx, `
			SELECT key, value FROM artist_attributes WHERE artist_id = ?
		`, artist.ID)
		if err != nil {
			return nil, err
		}

		artist.Attributes = make(map[string]string)
		for artistAttrRows.Next() {
			var key, value string
			if err := artistAttrRows.Scan(&key, &value); err != nil {
				artistAttrRows.Close()
				return nil, err
			}
			artist.Attributes[key] = value
		}
		artistAttrRows.Close()

		album.Artists = append(album.Artists, music.ArtistRole{Artist: &artist, Role: role})
	}

	return album, tx.Commit()
}

// AddArtist adds an artist to the database.
func (d *SqliteLibrary) AddArtist(ctx context.Context, artist *music.Artist) error {
	// Validate artist using domain validation
	if err := artist.Validate(); err != nil {
		slog.Error("AddArtist: validation failed", "error", err, "artistID", artist.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("AddArtist: failed to begin transaction", "error", err, "artistID", artist.ID)
		return err
	}
	defer tx.Rollback()

	// Insert artist
	_, err = tx.ExecContext(ctx, `
		INSERT INTO artists (id, name, sort_name)
		VALUES (?, ?, ?)
	`, artist.ID, artist.Name, artist.SortName)
	if err != nil {
		slog.Error("AddArtist: failed to insert artist", "error", err, "artistID", artist.ID, "artistName", artist.Name)
		return err
	}

	// Insert artist attributes
	for key, value := range artist.Attributes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO artist_attributes (artist_id, key, value)
			VALUES (?, ?, ?)
		`, artist.ID, key, value)
		if err != nil {
			slog.Error("AddArtist: failed to insert artist attribute", "error", err, "artistID", artist.ID, "key", key)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("AddArtist: failed to commit transaction", "error", err, "artistID", artist.ID)
		return err
	}

	slog.Debug("AddArtist: successfully added artist", "artistID", artist.ID, "artistName", artist.Name)
	return nil
}

// UpdateArtist updates an artist in the database.
func (d *SqliteLibrary) UpdateArtist(ctx context.Context, artist *music.Artist) error {
	// Validate artist using domain validation
	if err := artist.Validate(); err != nil {
		slog.Error("UpdateArtist: validation failed", "error", err, "artistID", artist.ID)
		return err
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update artist
	_, err = tx.ExecContext(ctx, `
		UPDATE artists
		SET name = ?, sort_name = ?
		WHERE id = ?
	`, artist.Name, artist.SortName, artist.ID)
	if err != nil {
		return err
	}

	// Delete existing artist attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM artist_attributes WHERE artist_id = ?`, artist.ID)
	if err != nil {
		return err
	}

	// Insert new artist attributes
	for key, value := range artist.Attributes {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO artist_attributes (artist_id, key, value)
			VALUES (?, ?, ?)
		`, artist.ID, key, value)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteArtist deletes an artist from the database and all tracks associated with that artist.
func (d *SqliteLibrary) DeleteArtist(ctx context.Context, id string) error {
	slog.Debug("DeleteArtist called", "artistID", id)

	// First, find all tracks associated with this artist (either directly or through albums)
	rows, err := d.db.QueryContext(ctx, `
		SELECT DISTINCT t.id, t.path
		FROM tracks t
		LEFT JOIN track_artists ta ON t.id = ta.track_id
		LEFT JOIN track_albums tal ON t.id = tal.track_id
		LEFT JOIN album_artists aa ON tal.album_id = aa.album_id
		WHERE ta.artist_id = ? OR aa.artist_id = ?
	`, id, id)
	if err != nil {
		return err
	}
	defer rows.Close()

	var trackIDs []string
	var trackPaths []string
	for rows.Next() {
		var trackID, trackPath string
		if err := rows.Scan(&trackID, &trackPath); err != nil {
			return err
		}
		trackIDs = append(trackIDs, trackID)
		trackPaths = append(trackPaths, trackPath)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all tracks associated with this artist
	for _, trackID := range trackIDs {
		// Delete track attributes
		_, err = tx.ExecContext(ctx, `DELETE FROM track_attributes WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track artists
		_, err = tx.ExecContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track albums
		_, err = tx.ExecContext(ctx, `DELETE FROM track_albums WHERE track_id = ?`, trackID)
		if err != nil {
			return err
		}

		// Delete track
		_, err = tx.ExecContext(ctx, `DELETE FROM tracks WHERE id = ?`, trackID)
		if err != nil {
			return err
		}
	}

	// Delete artist attributes
	_, err = tx.ExecContext(ctx, `DELETE FROM artist_attributes WHERE artist_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete track artists
	_, err = tx.ExecContext(ctx, `DELETE FROM track_artists WHERE artist_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete album artists
	_, err = tx.ExecContext(ctx, `DELETE FROM album_artists WHERE artist_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete artist
	_, err = tx.ExecContext(ctx, `DELETE FROM artists WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// Commit the database transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Delete track files from filesystem
	for _, path := range trackPaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to delete track file from filesystem", "path", path, "error", err)
			// Don't return error here - database deletion succeeded, file deletion is secondary
		} else {
			slog.Debug("Successfully deleted track file from filesystem", "path", path)
		}
	}

	return nil
}

// GetArtist gets an artist from the database.
func (d *SqliteLibrary) GetArtist(ctx context.Context, id string) (*music.Artist, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get artist basic info
	row := tx.QueryRowContext(ctx, `
		SELECT id, name, sort_name
		FROM artists
		WHERE id = ?
	`, id)

	artist := &music.Artist{}

	err = row.Scan(&artist.ID, &artist.Name, &artist.SortName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Get artist attributes
	attrRows, err := tx.QueryContext(ctx, `SELECT key, value FROM artist_attributes WHERE artist_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	artist.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		err := attrRows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}
		artist.Attributes[key] = value
	}

	return artist, tx.Commit()
}

// GetArtists gets all artists from the database.
func (d *SqliteLibrary) GetArtists(ctx context.Context) ([]*music.Artist, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, name
		FROM artists
		WHERE name != '' AND name IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := []*music.Artist{}

	for rows.Next() {
		artist := &music.Artist{}
		err := rows.Scan(&artist.ID, &artist.Name)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}

	return artists, nil
}

// GetAlbums gets all albums from the database.
func (d *SqliteLibrary) GetAlbums(ctx context.Context) ([]*music.Album, error) {
	// Get all album IDs first
	rows, err := d.db.QueryContext(ctx, `SELECT id FROM albums`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := []*music.Album{}

	for rows.Next() {
		var albumID string
		err := rows.Scan(&albumID)
		if err != nil {
			return nil, err
		}

		album, err := d.GetAlbum(ctx, albumID)
		if err != nil {
			return nil, err
		}
		// Skip albums that weren't found (shouldn't happen in a consistent database)
		if album == nil {
			continue
		}

		albums = append(albums, album)
	}

	return albums, nil
}

// GetTracks gets all tracks from the database.
func (d *SqliteLibrary) GetTracks(ctx context.Context) ([]*music.Track, error) {
	// Get all track IDs first
	rows, err := d.db.QueryContext(ctx, `SELECT id FROM tracks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []*music.Track{}

	for rows.Next() {
		var trackID string
		err := rows.Scan(&trackID)
		if err != nil {
			return nil, err
		}

		track, err := d.GetTrack(ctx, trackID)
		if err != nil {
			return nil, err
		}
		// Skip tracks that weren't found (shouldn't happen in a consistent database)
		if track == nil {
			continue
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTracksPaginated gets paginated tracks from the database.
func (d *SqliteLibrary) GetTracksPaginated(ctx context.Context, limit, offset int) ([]*music.Track, error) {
	// Get paginated track IDs first
	rows, err := d.db.QueryContext(ctx, `SELECT id FROM tracks ORDER BY title LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []*music.Track{}

	for rows.Next() {
		var trackID string
		err := rows.Scan(&trackID)
		if err != nil {
			return nil, err
		}

		track, err := d.GetTrack(ctx, trackID)
		if err != nil {
			return nil, err
		}
		// Skip tracks that weren't found (shouldn't happen in a consistent database)
		if track == nil {
			continue
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTracksFilteredPaginated gets paginated tracks from the database with filtering.
func (d *SqliteLibrary) GetTracksFilteredPaginated(ctx context.Context, limit, offset int, titleFilter string, artistIDs, albumIDs []string) ([]*music.Track, error) {
	query := `SELECT DISTINCT t.id FROM tracks t`
	args := []interface{}{}
	conditions := []string{}

	// Add title filter
	if titleFilter != "" {
		conditions = append(conditions, "t.title LIKE ?")
		args = append(args, "%"+titleFilter+"%")
	}

	// Add artist filter
	if len(artistIDs) > 0 {
		placeholders := strings.Repeat("?,", len(artistIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM track_artists ta WHERE ta.track_id = t.id AND ta.artist_id IN ("+placeholders+"))")
		for _, id := range artistIDs {
			args = append(args, id)
		}
	}

	// Add album filter
	if len(albumIDs) > 0 {
		placeholders := strings.Repeat("?,", len(albumIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM track_albums ta WHERE ta.track_id = t.id AND ta.album_id IN ("+placeholders+"))")
		for _, id := range albumIDs {
			args = append(args, id)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY t.title LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []*music.Track{}

	for rows.Next() {
		var trackID string
		err := rows.Scan(&trackID)
		if err != nil {
			return nil, err
		}

		track, err := d.GetTrack(ctx, trackID)
		if err != nil {
			return nil, err
		}
		// Skip tracks that weren't found (shouldn't happen in a consistent database)
		if track == nil {
			continue
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

// GetTracksFilteredCount gets the filtered count of tracks in the database.
func (d *SqliteLibrary) GetTracksFilteredCount(ctx context.Context, titleFilter string, artistIDs, albumIDs []string) (int, error) {
	query := `SELECT COUNT(DISTINCT t.id) FROM tracks t`
	args := []interface{}{}
	conditions := []string{}

	// Add title filter
	if titleFilter != "" {
		conditions = append(conditions, "t.title LIKE ?")
		args = append(args, "%"+titleFilter+"%")
	}

	// Add artist filter
	if len(artistIDs) > 0 {
		placeholders := strings.Repeat("?,", len(artistIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM track_artists ta WHERE ta.track_id = t.id AND ta.artist_id IN ("+placeholders+"))")
		for _, id := range artistIDs {
			args = append(args, id)
		}
	}

	// Add album filter
	if len(albumIDs) > 0 {
		placeholders := strings.Repeat("?,", len(albumIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM track_albums ta WHERE ta.track_id = t.id AND ta.album_id IN ("+placeholders+"))")
		for _, id := range albumIDs {
			args = append(args, id)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTracksCount gets the total count of tracks in the database.
func (d *SqliteLibrary) GetTracksCount(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tracks`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetArtistsPaginated gets paginated artists from the database.
func (d *SqliteLibrary) GetArtistsPaginated(ctx context.Context, limit, offset int) ([]*music.Artist, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT id, name FROM artists WHERE name != '' AND name IS NOT NULL ORDER BY name LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := []*music.Artist{}

	for rows.Next() {
		artist := &music.Artist{}
		err := rows.Scan(&artist.ID, &artist.Name)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}

	return artists, nil
}

// GetArtistsFilteredPaginated gets paginated artists from the database with filtering.
func (d *SqliteLibrary) GetArtistsFilteredPaginated(ctx context.Context, limit, offset int, nameFilter string) ([]*music.Artist, error) {
	query := `SELECT id, name FROM artists WHERE name != '' AND name IS NOT NULL`
	args := []interface{}{}

	// Add name filter
	if nameFilter != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	query += " ORDER BY name LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := []*music.Artist{}

	for rows.Next() {
		artist := &music.Artist{}
		err := rows.Scan(&artist.ID, &artist.Name)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}

	return artists, nil
}

// GetArtistsFilteredCount gets the filtered count of artists in the database.
func (d *SqliteLibrary) GetArtistsFilteredCount(ctx context.Context, nameFilter string) (int, error) {
	query := `SELECT COUNT(*) FROM artists WHERE name != '' AND name IS NOT NULL`
	args := []interface{}{}

	// Add name filter
	if nameFilter != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+nameFilter+"%")
	}

	var count int
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetArtistsCount gets the total count of artists in the database.
func (d *SqliteLibrary) GetArtistsCount(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artists WHERE name != '' AND name IS NOT NULL`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAlbumsPaginated gets paginated albums from the database.
func (d *SqliteLibrary) GetAlbumsPaginated(ctx context.Context, limit, offset int) ([]*music.Album, error) {
	// Get paginated album IDs first
	rows, err := d.db.QueryContext(ctx, `SELECT id FROM albums ORDER BY title LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := []*music.Album{}

	for rows.Next() {
		var albumID string
		err := rows.Scan(&albumID)
		if err != nil {
			return nil, err
		}

		album, err := d.GetAlbum(ctx, albumID)
		if err != nil {
			return nil, err
		}
		// Skip albums that weren't found (shouldn't happen in a consistent database)
		if album == nil {
			continue
		}

		albums = append(albums, album)
	}

	return albums, nil
}

// GetAlbumsFilteredPaginated gets paginated albums from the database with filtering.
func (d *SqliteLibrary) GetAlbumsFilteredPaginated(ctx context.Context, limit, offset int, titleFilter string, artistIDs []string) ([]*music.Album, error) {
	query := `SELECT DISTINCT a.id FROM albums a`
	args := []interface{}{}
	conditions := []string{}

	// Add title filter
	if titleFilter != "" {
		conditions = append(conditions, "a.title LIKE ?")
		args = append(args, "%"+titleFilter+"%")
	}

	// Add artist filter
	if len(artistIDs) > 0 {
		placeholders := strings.Repeat("?,", len(artistIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM album_artists aa WHERE aa.album_id = a.id AND aa.artist_id IN ("+placeholders+"))")
		for _, id := range artistIDs {
			args = append(args, id)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY a.title LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := []*music.Album{}

	for rows.Next() {
		var albumID string
		err := rows.Scan(&albumID)
		if err != nil {
			return nil, err
		}

		album, err := d.GetAlbum(ctx, albumID)
		if err != nil {
			return nil, err
		}
		// Skip albums that weren't found (shouldn't happen in a consistent database)
		if album == nil {
			continue
		}

		albums = append(albums, album)
	}

	return albums, nil
}

// GetAlbumsFilteredCount gets the filtered count of albums in the database.
func (d *SqliteLibrary) GetAlbumsFilteredCount(ctx context.Context, titleFilter string, artistIDs []string) (int, error) {
	query := `SELECT COUNT(DISTINCT a.id) FROM albums a`
	args := []interface{}{}
	conditions := []string{}

	// Add title filter
	if titleFilter != "" {
		conditions = append(conditions, "a.title LIKE ?")
		args = append(args, "%"+titleFilter+"%")
	}

	// Add artist filter
	if len(artistIDs) > 0 {
		placeholders := strings.Repeat("?,", len(artistIDs))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		conditions = append(conditions, "EXISTS (SELECT 1 FROM album_artists aa WHERE aa.album_id = a.id AND aa.artist_id IN ("+placeholders+"))")
		for _, id := range artistIDs {
			args = append(args, id)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAlbumsCount gets the total count of albums in the database.
func (d *SqliteLibrary) GetAlbumsCount(ctx context.Context) (int, error) {
	var count int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM albums`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *SqliteLibrary) GetArtistByName(ctx context.Context, name string) (*music.Artist, error) {
	row := d.db.QueryRowContext(ctx, `SELECT id, name, sort_name FROM artists WHERE name = ? AND name != '' AND name IS NOT NULL`, name)
	artist := &music.Artist{}
	err := row.Scan(&artist.ID, &artist.Name, &artist.SortName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Get artist attributes
	attrRows, err := d.db.QueryContext(ctx, `SELECT key, value FROM artist_attributes WHERE artist_id = ?`, artist.ID)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	artist.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		if err := attrRows.Scan(&key, &value); err != nil {
			attrRows.Close()
			return nil, err
		}
		artist.Attributes[key] = value
	}
	attrRows.Close()

	return artist, nil
}

func (d *SqliteLibrary) GetAlbumByArtistAndName(ctx context.Context, artistID, name string) (*music.Album, error) {
	var albumID string
	err := d.db.QueryRowContext(ctx, `
		SELECT a.id FROM albums a
		JOIN album_artists aa ON a.id = aa.album_id
		WHERE aa.artist_id = ? AND a.title = ?
	`, artistID, name).Scan(&albumID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return d.GetAlbum(ctx, albumID)
}

func (d *SqliteLibrary) FindOrCreateArtist(ctx context.Context, artistName string) (*music.Artist, error) {
	// Validate artist name before proceeding
	if strings.TrimSpace(artistName) == "" {
		return nil, fmt.Errorf("artist name cannot be empty")
	}

	artist, err := d.GetArtistByName(ctx, artistName)
	if err != nil {
		return nil, err
	}
	if artist != nil {
		return artist, nil
	}

	newArtist := &music.Artist{
		ID:   uuid.New().String(),
		Name: artistName,
	}
	if err := d.AddArtist(ctx, newArtist); err != nil {
		return nil, err
	}
	return newArtist, nil
}

func (d *SqliteLibrary) FindOrCreateAlbum(ctx context.Context, artist *music.Artist, albumTitle string, year int) (*music.Album, error) {
	album, err := d.GetAlbumByArtistAndName(ctx, artist.ID, albumTitle)
	if err != nil {
		return nil, err
	}
	if album != nil {
		return album, nil
	}

	newAlbum := &music.Album{
		ID:    uuid.New().String(),
		Title: albumTitle,
		Artists: []music.ArtistRole{
			{Artist: artist, Role: "main"},
		},
		ReleaseDate: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := d.AddAlbum(ctx, newAlbum); err != nil {
		return nil, err
	}
	return newAlbum, nil
}

// FindTrackByMetadata finds a track by matching title, artist name, and album title
func (d *SqliteLibrary) FindTrackByMetadata(ctx context.Context, title, artistName, albumTitle string) (*music.Track, error) {
	// Query to find tracks with matching metadata
	row := d.db.QueryRowContext(ctx, `
		SELECT t.id, t.path, t.title, t.title_version, t.duration, t.track_number, t.disc_number,
			   t.isrc, t.bitrate, t.format, t.sample_rate, t.bit_depth, t.channels,
			   t.explicit_content, t.preview_url, t.composer, t.genre, t.year,
			   t.original_year, t.lyrics, t.explicit_lyrics, t.bpm, t.gain, t.source, t.source_url, t.added_date, t.modified_date
		FROM tracks t
		JOIN track_albums ta ON t.id = ta.track_id
		JOIN albums a ON ta.album_id = a.id
		JOIN album_artists aa ON a.id = aa.album_id
		JOIN artists art ON aa.artist_id = art.id
		WHERE LOWER(t.title) = LOWER(?)
		AND LOWER(art.name) = LOWER(?)
		AND LOWER(a.title) = LOWER(?)
		LIMIT 1
	`, title, artistName, albumTitle)

	track := &music.Track{}
	var addedDateStr, modifiedDateStr string
	var sourceNull, sourceURLNull sql.NullString
	err := row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &sourceNull, &sourceURLNull, &addedDateStr, &modifiedDateStr)

	// Handle nullable source fields
	track.MetadataSource.Source = sourceNull.String
	track.MetadataSource.MetadataSourceURL = sourceURLNull.String
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No track found
		}
		return nil, err
	}

	track.AddedDate, _ = time.Parse(time.RFC3339, addedDateStr)
	track.ModifiedDate, _ = time.Parse(time.RFC3339, modifiedDateStr)

	// Get track attributes
	var attrRows *sql.Rows
	attrRows, err = d.db.QueryContext(ctx, `SELECT key, value FROM track_attributes WHERE track_id = ?`, track.ID)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	track.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		if err := attrRows.Scan(&key, &value); err != nil {
			attrRows.Close()
			return nil, err
		}
		track.Attributes[key] = value
	}
	attrRows.Close()

	// Get track artists
	rows, err := d.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.sort_name, ta.role
		FROM track_artists ta
		JOIN artists a ON ta.artist_id = a.id
		WHERE ta.track_id = ?
	`, track.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var artist music.Artist
		var role string
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SortName, &role)
		if err != nil {
			return nil, err
		}
		track.Artists = append(track.Artists, music.ArtistRole{Artist: &artist, Role: role})
	}

	// Get track album
	var albumID string
	err = d.db.QueryRowContext(ctx, `SELECT album_id FROM track_albums WHERE track_id = ?`, track.ID).Scan(&albumID)
	if err == nil {
		album, err := d.GetAlbum(ctx, albumID)
		if err != nil {
			return nil, err
		}
		track.Album = album
	}

	// Get track attributes
	attrRows, err = d.db.QueryContext(ctx, `SELECT key, value FROM track_attributes WHERE track_id = ?`, track.ID)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	track.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		err := attrRows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}
		track.Attributes[key] = value
	}

	return track, nil
}

// FindTrackByPath finds a track by its file path
func (d *SqliteLibrary) FindTrackByPath(ctx context.Context, path string) (*music.Track, error) {
	row := d.db.QueryRowContext(ctx, `
		SELECT t.id, t.path, t.title, t.title_version, t.duration, t.track_number, t.disc_number,
			   t.isrc, t.bitrate, t.format, t.sample_rate, t.bit_depth, t.channels,
			   t.explicit_content, t.preview_url, t.composer, t.genre, t.year,
			   t.original_year, t.lyrics, t.explicit_lyrics, t.bpm, t.gain, t.source, t.source_url, t.added_date, t.modified_date
		FROM tracks t
		WHERE t.path = ?
		LIMIT 1
	`, path)

	track := &music.Track{}
	var addedDateStr, modifiedDateStr string
	var sourceNull, sourceURLNull sql.NullString
	err := row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &sourceNull, &sourceURLNull, &addedDateStr, &modifiedDateStr)

	// Handle nullable source fields
	track.MetadataSource.Source = sourceNull.String
	track.MetadataSource.MetadataSourceURL = sourceURLNull.String
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No track found
		}
		return nil, err
	}

	track.AddedDate, _ = time.Parse(time.RFC3339, addedDateStr)
	track.ModifiedDate, _ = time.Parse(time.RFC3339, modifiedDateStr)

	// Get track attributes
	var attrRows *sql.Rows
	attrRows, err = d.db.QueryContext(ctx, `SELECT key, value FROM track_attributes WHERE track_id = ?`, track.ID)
	if err != nil {
		return nil, err
	}
	defer attrRows.Close()

	track.Attributes = make(map[string]string)
	for attrRows.Next() {
		var key, value string
		err := attrRows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}
		track.Attributes[key] = value
	}

	return track, nil
}
