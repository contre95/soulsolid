package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"soulsolid/src/music"
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
			data BLOB,
			composer TEXT,
			genre TEXT,
			year INTEGER,
			original_year INTEGER,
			lyrics TEXT,
			explicit_lyrics BOOLEAN DEFAULT FALSE,
			bpm REAL,
			gain REAL,
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
		
		CREATE INDEX IF NOT EXISTS idx_track_artists_track ON track_artists(track_id);
		CREATE INDEX IF NOT EXISTS idx_track_artists_artist ON track_artists(artist_id);
		CREATE INDEX IF NOT EXISTS idx_album_artists_album ON album_artists(album_id);
		CREATE INDEX IF NOT EXISTS idx_album_artists_artist ON album_artists(artist_id);
		CREATE INDEX IF NOT EXISTS idx_track_attributes_track ON track_attributes(track_id);
		CREATE INDEX IF NOT EXISTS idx_album_attributes_album ON album_attributes(album_id);
		CREATE INDEX IF NOT EXISTS idx_artist_attributes_artist ON artist_attributes(artist_id);
	`)
	return err
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
			explicit_content, preview_url, data, composer, genre, year, original_year, lyrics,
			explicit_lyrics, bpm, gain, added_date, modified_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, track.ID, track.Path, track.Title, track.TitleVersion, track.Metadata.Duration, track.Metadata.TrackNumber, track.Metadata.DiscNumber,
		track.ISRC, track.ChromaprintFingerprint, track.Bitrate, track.Format, track.SampleRate, track.BitDepth, track.Channels,
		track.ExplicitContent, track.PreviewURL, track.Data, track.Metadata.Composer, track.Metadata.Genre, track.Metadata.Year,
		track.Metadata.OriginalYear, track.Metadata.Lyrics, track.Metadata.ExplicitLyrics, track.Metadata.BPM, track.Metadata.Gain,
		track.AddedDate.Format(time.RFC3339), track.ModifiedDate.Format(time.RFC3339))
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
			isrc, bitrate, format, sample_rate, bit_depth, channels, explicit_content,
			preview_url, data, composer, genre, year, original_year, lyrics, explicit_lyrics,
			bpm, gain, added_date, modified_date
		FROM tracks
		WHERE id = ?
	`, id)

	track := &music.Track{}
	var addedDateStr, modifiedDateStr string

	err = row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL, &track.Data,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &addedDateStr, &modifiedDateStr)
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

	return track, tx.Commit()
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
			isrc = ?, bitrate = ?, format = ?, sample_rate = ?, bit_depth = ?, channels = ?,
			explicit_content = ?, preview_url = ?, data = ?, composer = ?, genre = ?, year = ?,
			original_year = ?, lyrics = ?, explicit_lyrics = ?, bpm = ?, gain = ?, modified_date = ?
		WHERE id = ?
	`, track.Path, track.Title, track.TitleVersion, track.Metadata.Duration, track.Metadata.TrackNumber, track.Metadata.DiscNumber,
		track.ISRC, track.Bitrate, track.Format, track.SampleRate, track.BitDepth, track.Channels,
		track.ExplicitContent, track.PreviewURL, track.Data, track.Metadata.Composer, track.Metadata.Genre, track.Metadata.Year,
		track.Metadata.OriginalYear, track.Metadata.Lyrics, track.Metadata.ExplicitLyrics, track.Metadata.BPM, track.Metadata.Gain,
		track.ModifiedDate.Format(time.RFC3339), track.ID)
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
			   t.explicit_content, t.preview_url, t.data, t.composer, t.genre, t.year,
			   t.original_year, t.lyrics, t.explicit_lyrics, t.bpm, t.gain, t.added_date, t.modified_date
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
	err := row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL, &track.Data,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &addedDateStr, &modifiedDateStr)
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

// UpdateTrackFingerprint updates the fingerprint for a specific track
func (d *SqliteLibrary) UpdateTrackFingerprint(ctx context.Context, trackID, fingerprint string) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE tracks SET chromaprint_fingerprint = ? WHERE id = ?
	`, fingerprint, trackID)
	return err
}

// FindTrackByPath finds a track by its file path
func (d *SqliteLibrary) FindTrackByPath(ctx context.Context, path string) (*music.Track, error) {
	// Query to find track with matching path
	row := d.db.QueryRowContext(ctx, `
		SELECT t.id, t.path, t.title, t.title_version, t.duration, t.track_number, t.disc_number,
			   t.isrc, t.bitrate, t.format, t.sample_rate, t.bit_depth, t.channels,
			   t.explicit_content, t.preview_url, t.data, t.composer, t.genre, t.year,
			   t.original_year, t.lyrics, t.explicit_lyrics, t.bpm, t.gain, t.added_date, t.modified_date
		FROM tracks t
		WHERE t.path = ?
		LIMIT 1
	`, path)

	track := &music.Track{}
	var addedDateStr, modifiedDateStr string
	err := row.Scan(&track.ID, &track.Path, &track.Title, &track.TitleVersion, &track.Metadata.Duration,
		&track.Metadata.TrackNumber, &track.Metadata.DiscNumber,
		&track.ISRC, &track.Bitrate, &track.Format, &track.SampleRate, &track.BitDepth,
		&track.Channels, &track.ExplicitContent, &track.PreviewURL, &track.Data,
		&track.Metadata.Composer, &track.Metadata.Genre, &track.Metadata.Year,
		&track.Metadata.OriginalYear, &track.Metadata.Lyrics, &track.Metadata.ExplicitLyrics,
		&track.Metadata.BPM, &track.Metadata.Gain, &addedDateStr, &modifiedDateStr)
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
