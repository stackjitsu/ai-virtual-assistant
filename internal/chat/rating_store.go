package chat

import (
	"context"
	"time"
)

// One statement decides eligibility, deduplication, replacement and the row
// returned to the caller. Both adapters execute exactly this statement; a
// detached read cannot substitute for the serialized write decision.
// Fixed-width UTC nanoseconds make lexical comparison safe across processes.
const saveRatingSQL = `INSERT INTO ratings(conversation_id,rating,comment,rated_at)
 SELECT ?,?,?,? WHERE EXISTS (
 SELECT 1 FROM turns WHERE conversation_id=? AND status IN ('answered','needs_clarification'))
 ON CONFLICT(conversation_id) DO UPDATE SET
 rating=excluded.rating, comment=excluded.comment,
 rated_at=CASE WHEN ratings.rating=excluded.rating AND ratings.comment=excluded.comment THEN ratings.rated_at
 WHEN excluded.rated_at>ratings.rated_at THEN excluded.rated_at
 WHEN substr(ratings.rated_at,21,9)='999999999' THEN strftime('%Y-%m-%dT%H:%M:%S',ratings.rated_at,'+1 second')||'.000000000Z'
 ELSE substr(ratings.rated_at,1,20)||printf('%09d',CAST(substr(ratings.rated_at,21,9) AS INTEGER)+1)||'Z' END
 RETURNING conversation_id,rating,comment,rated_at`

func ratingTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func decodeRatingRow(v []any) (Rating, error) {
	if len(v) != 4 {
		return Rating{}, ErrUnavailable
	}
	id, a := textCell(v[0])
	n, b := v[1].(int64)
	comment, c := textCell(v[2])
	when, d := timeCell(v[3])
	if !a || !b || !c || !d || n < 1 || n > 5 {
		return Rating{}, ErrUnavailable
	}
	out := Rating{id, int(n), comment, when}
	if validateRating(out) != nil {
		return Rating{}, ErrUnavailable
	}
	return out, nil
}

// SaveRating is atomic on SQLite and Turso. A failed remote response may
// conceal a committed update; never infer a successful replacement from a
// later read that could belong to another concurrent operation.
func (s *SQLStore) SaveRating(ctx context.Context, proposed Rating) (Rating, error) {
	if validateRating(proposed) != nil {
		return Rating{}, ErrInvalid
	}
	r, err := s.query(ctx, saveRatingSQL, proposed.ConversationID, proposed.Value, proposed.Comment, ratingTimestamp(proposed.RatedAt), proposed.ConversationID)
	if err != nil {
		return Rating{}, ErrUnavailable
	}
	if len(r.rows) == 1 {
		stored, err := decodeRatingRow(r.rows[0])
		if err != nil || stored.ConversationID != proposed.ConversationID {
			return Rating{}, ErrUnavailable
		}
		return stored, nil
	}
	if len(r.rows) != 0 {
		return Rating{}, ErrUnavailable
	}
	// A zero-row conditional INSERT is definitive for this statement: there
	// was no eligible completed turn at the atomic decision point.
	return Rating{}, ErrRatingRequiresTurn
}
