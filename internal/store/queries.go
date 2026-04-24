package store

// SQL queries are centralized here so the impl file can focus on control
// flow. Columns match the order used on scan and on insert bind sites; keep
// them aligned when editing.

const (
	qSelectPRByID = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs WHERE id = ?`

	qSelectPRsAll = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs ORDER BY last_seen_at DESC, id ASC`

	qSelectPRsPending = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs WHERE review_pending = 1
		ORDER BY last_seen_at DESC, id ASC`

	qSelectPRsHistory = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs WHERE review_pending = 0
		ORDER BY last_seen_at DESC, id ASC`

	qInsertPR = `INSERT INTO prs (
		id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, first_seen_at, last_seen_at,
		last_notified_at, profile_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// qUpdatePRMutable rewrites the fields that may legitimately change
	// between polls. first_seen_at is intentionally absent — it stays frozen
	// at the initial sighting.
	qUpdatePRMutable = `UPDATE prs
		SET title = ?, author = ?, url = ?, repo = ?, is_draft = ?,
			review_pending = 1, last_seen_at = ?,
			state = CASE WHEN state = '' THEN 'OPEN' ELSE state END
		WHERE id = ?`

	// qMarkNotPendingAll clears review_pending for every PR still flagged
	// pending. Used when the poll result is empty.
	qMarkNotPendingAll = `UPDATE prs SET review_pending = 0 WHERE review_pending = 1`

	qUpdatePRStatus = `UPDATE prs
		SET additions = ?, deletions = ?, is_draft = ?, state = ?
		WHERE id = ?`

	// qDeleteRetention drops non-OPEN history older than the cutoff. state=''
	// is preserved to match the legacy JSON semantics (pre-enrichment records
	// never expire solely due to age).
	qDeleteRetention = `DELETE FROM prs
		WHERE state NOT IN ('OPEN', '') AND last_seen_at < ?`

	// qClearHistory wipes finalized rows (merged / closed) from the history.
	// Rows still at state='OPEN' but review_pending=0 survive on purpose:
	// the PR is alive on GitHub and a future re-request needs the prior
	// record to be detected as a transition instead of a brand-new sighting
	// (SPEC re-request detection). state='' legacy rows are also spared for
	// the same reason.
	qClearHistory = `DELETE FROM prs WHERE review_pending = 0 AND state NOT IN ('OPEN', '')`

	qGetMeta = `SELECT value FROM meta WHERE key = ?`

	qSetMeta = `INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`

	qCountPRs        = `SELECT COUNT(*) FROM prs`
	qCountPRsPending = `SELECT COUNT(*) FROM prs WHERE review_pending = 1`
	qCountPRsHistory = `SELECT COUNT(*) FROM prs WHERE review_pending = 0`
)

// Meta keys persisted by the store.
const (
	metaLastPollAt     = "last_poll_at"
	metaJSONMigratedAt = "json_migrated_at"
)
