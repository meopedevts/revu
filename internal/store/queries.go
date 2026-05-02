package store

// SQL queries are centralized here so the impl file can focus on control
// flow. Columns match the order used on scan and on insert bind sites; keep
// them aligned when editing.

const (
	qSelectPRByID = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs WHERE id = ?`

	qSelectPRsAll = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs ORDER BY last_seen_at DESC, id ASC`

	// qSelectPRsPending / qSelectPRsHistory encode the REV-16 rule (refined):
	// classification is driven by state alone. Once the PR is MERGED or
	// CLOSED there is nothing left for me to do, so it leaves pending even
	// if I never submitted a review — the common trigger is a co-reviewer
	// merging before I got to it. review_state is still persisted so the
	// badge communicates what happened, but it no longer gates the tab.
	qSelectPRsPending = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs
		WHERE state NOT IN ('MERGED', 'CLOSED')
		ORDER BY last_seen_at DESC, id ASC`

	qSelectPRsHistory = `SELECT id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at
		FROM prs
		WHERE state IN ('MERGED', 'CLOSED')
		ORDER BY last_seen_at DESC, id ASC`

	qInsertPR = `INSERT INTO prs (
		id, number, repo, title, author, url, state, is_draft,
		additions, deletions, review_pending, review_state, first_seen_at, last_seen_at,
		last_notified_at, profile_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// qUpdatePRMutable rewrites the fields that may legitimately change
	// between polls. first_seen_at is intentionally absent — it stays frozen
	// at the initial sighting. When the PR re-enters the search after having
	// dropped out (review_pending was 0), reset review_state to PENDING: GitHub
	// only surfaces a PR under review-requested when a fresh review is
	// expected, so whatever state lingered from the previous round is stale.
	qUpdatePRMutable = `UPDATE prs
		SET title = ?, author = ?, url = ?, repo = ?, is_draft = ?,
			review_pending = 1,
			review_state = CASE WHEN review_pending = 0 THEN 'PENDING' ELSE review_state END,
			last_seen_at = ?,
			state = CASE WHEN state = '' THEN 'OPEN' ELSE state END
		WHERE id = ?`

	// qMarkNotified persiste o instante da última notificação enviada,
	// usado pelo poller pra throttle de re-requests dentro da janela de
	// cooldown.
	qMarkNotified = `UPDATE prs SET last_notified_at = ? WHERE id = ?`

	// qBackfillNotifiedAt populates last_notified_at for legacy rows where
	// it was never written. Runs once on Load (gated by metaNotifyBackfillDone)
	// so existing PRs don't burst-notify on the upgrade after REV-43.
	qBackfillNotifiedAt = `UPDATE prs SET last_notified_at = first_seen_at
		WHERE last_notified_at IS NULL OR last_notified_at = ''`

	qUpdatePRStatus = `UPDATE prs
		SET additions = ?, deletions = ?, is_draft = ?, state = ?, review_state = ?
		WHERE id = ?`

	// qDeleteRetention drops non-OPEN history older than the cutoff. state=''
	// is preserved to match the legacy JSON semantics (pre-enrichment records
	// never expire solely due to age).
	qDeleteRetention = `DELETE FROM prs
		WHERE state NOT IN ('OPEN', '') AND last_seen_at < ?`

	// qClearHistory wipes rows that satisfy the REV-16 history rule: PR is
	// finalized (MERGED or CLOSED). Rows with state='' legacy / 'OPEN' stay
	// so re-request detection keeps a prior sighting to transition off.
	qClearHistory = `DELETE FROM prs WHERE state IN ('MERGED', 'CLOSED')`

	qGetMeta = `SELECT value FROM meta WHERE key = ?`

	qSetMeta = `INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
)

// Meta keys persisted by the store.
const (
	metaLastPollAt         = "last_poll_at"
	metaJSONMigratedAt     = "json_migrated_at"
	metaNotifyBackfillDone = "notify_backfill_done"
	// metaTrayAcknowledgedAt registra o instante em que o user reconheceu
	// (abriu a janela) o estado de attention do tray. PRs com last_seen_at
	// posterior a esse instante voltam a sinalizar attention.
	metaTrayAcknowledgedAt = "tray_acknowledged_at"
)
