package profiles

const (
	profileCols = `id, name, auth_method, keyring_ref, github_username,
		is_active, created_at, last_validated_at`

	qSelectAll = `SELECT ` + profileCols + ` FROM profiles
		ORDER BY created_at ASC, id ASC`

	qSelectByID = `SELECT ` + profileCols + ` FROM profiles WHERE id = ?`

	qSelectActive = `SELECT ` + profileCols + ` FROM profiles WHERE is_active = 1 LIMIT 1`

	qInsert = `INSERT INTO profiles (
		id, name, auth_method, keyring_ref, github_username,
		is_active, created_at, last_validated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	qUpdateFields = `UPDATE profiles SET
		name = ?, auth_method = ?, keyring_ref = ?,
		github_username = ?, last_validated_at = ?
		WHERE id = ?`

	qDelete = `DELETE FROM profiles WHERE id = ?`

	qClearActive = `UPDATE profiles SET is_active = 0 WHERE is_active = 1`

	qSetActive = `UPDATE profiles SET is_active = 1 WHERE id = ?`

	qCount = `SELECT COUNT(*) FROM profiles`
)
