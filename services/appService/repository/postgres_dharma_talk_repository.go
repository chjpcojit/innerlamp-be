package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"chjpcoj.com.innerlamp/pkg/database"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// dharmaTalkColumns / dharmaTalkVisible / dharmaTalkOrder are shared by every
// read so list and detail scan identically, hide the same rows, and paging
// order can never diverge.
const (
	dharmaTalkColumns = `id, sort_order, title, speaker, speaker_initials, category, season, published_date,
		duration_minutes, summary, key_points, quote, quote_source, tags, language, media_type,
		youtube_id, audio_url, chapters, follow_up_meditation, needs_review, review_note,
		source_name, source_url, license`

	// A talk under review is stored but never served.
	dharmaTalkVisible = `needs_review = false`

	// id is the final tiebreaker so equal sort_order rows cannot swap between pages.
	dharmaTalkOrder = `ORDER BY sort_order ASC, id ASC`
)

// chapterJSON / followUpJSON are the shapes stored in the JSONB columns.
type chapterJSON struct {
	StartSeconds int    `json:"start_seconds"`
	Title        string `json:"title"`
}

type followUpJSON struct {
	Title        string `json:"title"`
	Minutes      int    `json:"minutes"`
	MeditationID string `json:"meditation_id"`
}

type dharmaTalkRepository struct {
	db database.Database
}

// NewDharmaTalkRepository returns the Postgres-backed domain.DharmaTalkRepository.
func NewDharmaTalkRepository(db database.Database) domain.DharmaTalkRepository {
	return &dharmaTalkRepository{db: db}
}

// List returns one window of the ordered, displayable list plus the number of
// displayable rows. The count and the page are two statements, not one snapshot.
func (r *dharmaTalkRepository) List(ctx context.Context, offset, limit int) ([]domain.DharmaTalk, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM dharma_talks WHERE `+dharmaTalkVisible).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count visible dharma talks: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+dharmaTalkColumns+` FROM dharma_talks WHERE `+dharmaTalkVisible+` `+dharmaTalkOrder+` OFFSET $1 LIMIT $2`,
		offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query dharma talks: %w", err)
	}
	items, err := scanDharmaTalks(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindByID returns the displayable talk or domain.ErrDharmaTalkNotFound.
func (r *dharmaTalkRepository) FindByID(ctx context.Context, id string) (*domain.DharmaTalk, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+dharmaTalkColumns+` FROM dharma_talks WHERE id = $1 AND `+dharmaTalkVisible, id)
	t, err := scanDharmaTalk(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDharmaTalkNotFound
		}
		return nil, fmt.Errorf("find dharma talk: %w", err)
	}
	return t, nil
}

// Count returns the number of stored talks, hidden ones included — the seed's
// "is the table empty" check must not treat an all-under-review table as empty.
func (r *dharmaTalkRepository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM dharma_talks`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count dharma talks: %w", err)
	}
	return n, nil
}

// InsertMissing inserts talks whose id is absent, in one transaction: all rows
// or none, so a bad row in the seed cannot leave the catalogue half-seeded.
// ON CONFLICT DO NOTHING keeps concurrent seeders and existing rows safe.
func (r *dharmaTalkRepository) InsertMissing(ctx context.Context, talks []domain.DharmaTalk) (int, error) {
	const query = `
		INSERT INTO dharma_talks (id, sort_order, title, speaker, speaker_initials, category, season,
			published_date, duration_minutes, summary, key_points, quote, quote_source, tags, language,
			media_type, youtube_id, audio_url, chapters, follow_up_meditation, needs_review, review_note,
			source_name, source_url, license)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
		ON CONFLICT (id) DO NOTHING
	`
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin insert dharma talks: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	inserted := 0
	for _, t := range talks {
		keyPoints, tags := t.KeyPoints, t.Tags
		if keyPoints == nil {
			keyPoints = []string{}
		}
		if tags == nil {
			tags = []string{}
		}
		chapters := make([]chapterJSON, 0, len(t.Chapters))
		for _, c := range t.Chapters {
			chapters = append(chapters, chapterJSON{StartSeconds: c.StartSeconds, Title: c.Title})
		}
		chaptersRaw, err := json.Marshal(chapters)
		if err != nil {
			return 0, fmt.Errorf("encode chapters of dharma talk %q: %w", t.ID, err)
		}

		// NULL, not '' / 0 / zero date, when the source has none.
		var season, published, duration, youtubeID, audioURL, followUp any
		if t.Season != "" {
			season = t.Season
		}
		if t.PublishedDate != nil {
			published = *t.PublishedDate
		}
		if t.DurationMinutes > 0 {
			duration = t.DurationMinutes
		}
		if t.YouTubeID != "" {
			youtubeID = t.YouTubeID
		}
		if t.AudioURL != "" {
			audioURL = t.AudioURL
		}
		if f := t.FollowUpMeditation; f != nil {
			raw, err := json.Marshal(followUpJSON{Title: f.Title, Minutes: f.Minutes, MeditationID: f.MeditationID})
			if err != nil {
				return 0, fmt.Errorf("encode follow-up meditation of dharma talk %q: %w", t.ID, err)
			}
			followUp = raw
		}

		res, err := tx.ExecContext(ctx, query,
			t.ID, t.SortOrder, t.Title, t.Speaker, t.SpeakerInitials, t.Category, season,
			published, duration, t.Summary, pq.Array(keyPoints), t.Quote, t.QuoteSource, pq.Array(tags), t.Language,
			string(t.MediaType), youtubeID, audioURL, chaptersRaw, followUp, t.NeedsReview, t.ReviewNote,
			t.SourceName, t.SourceURL, t.License,
		)
		if err != nil {
			return 0, fmt.Errorf("insert dharma talk %q: %w", t.ID, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("rows affected for dharma talk %q: %w", t.ID, err)
		}
		inserted += int(n)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit insert dharma talks: %w", err)
	}
	return inserted, nil
}

func scanDharmaTalk(row rowScanner) (*domain.DharmaTalk, error) {
	var t domain.DharmaTalk
	var (
		season, youtubeID, audioURL sql.NullString
		published                   sql.NullTime
		duration                    sql.NullInt64
		mediaType                   string
		chaptersRaw, followUpRaw    []byte
	)
	if err := row.Scan(
		&t.ID, &t.SortOrder, &t.Title, &t.Speaker, &t.SpeakerInitials, &t.Category, &season, &published,
		&duration, &t.Summary, pq.Array(&t.KeyPoints), &t.Quote, &t.QuoteSource, pq.Array(&t.Tags), &t.Language,
		&mediaType, &youtubeID, &audioURL, &chaptersRaw, &followUpRaw, &t.NeedsReview, &t.ReviewNote,
		&t.SourceName, &t.SourceURL, &t.License,
	); err != nil {
		return nil, err
	}
	t.MediaType = domain.MediaType(mediaType)
	t.Season = season.String
	t.YouTubeID = youtubeID.String
	t.AudioURL = audioURL.String
	t.DurationMinutes = int(duration.Int64)
	if published.Valid {
		d := published.Time
		t.PublishedDate = &d
	}

	var chapters []chapterJSON
	if err := json.Unmarshal(chaptersRaw, &chapters); err != nil {
		return nil, fmt.Errorf("decode chapters of dharma talk %q: %w", t.ID, err)
	}
	for _, c := range chapters {
		t.Chapters = append(t.Chapters, domain.DharmaTalkChapter{StartSeconds: c.StartSeconds, Title: c.Title})
	}
	if followUpRaw != nil {
		var f followUpJSON
		if err := json.Unmarshal(followUpRaw, &f); err != nil {
			return nil, fmt.Errorf("decode follow-up meditation of dharma talk %q: %w", t.ID, err)
		}
		t.FollowUpMeditation = &domain.FollowUpMeditation{Title: f.Title, Minutes: f.Minutes, MeditationID: f.MeditationID}
	}
	return &t, nil
}

func scanDharmaTalks(rows *sql.Rows) ([]domain.DharmaTalk, error) {
	defer func() { _ = rows.Close() }()
	items := []domain.DharmaTalk{}
	for rows.Next() {
		t, err := scanDharmaTalk(rows)
		if err != nil {
			return nil, fmt.Errorf("scan dharma talk: %w", err)
		}
		items = append(items, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dharma talks: %w", err)
	}
	return items, nil
}
