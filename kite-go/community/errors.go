package community

// Use hard coded error codes instead of iota, to avoid issues
// when reorganizing, shuffling, adding new errors.
const (
	// User creation errors

	ErrCodeUserExists    = 1
	ErrCodeInvalidUser   = 2
	ErrCodePasswordShort = 3
	ErrCodePasswordLong  = 4

	// User login errors

	ErrCodeUserNotFound     = 5
	ErrCodeWrongPassword    = 6
	ErrCodeUserPasswordless = 9

	// Session validation errors

	ErrCodeInvalidSession = 7
	ErrCodeLogoutFailed   = 8

	// Community error page errors

	ErrCodeErrorPageNotFound        = 101
	ErrCodeInvalidErrorPageID       = 102
	ErrCodeInvalidErrorTemplateID   = 103
	ErrCodeInvalidErrorTemplateLang = 104
	ErrCodeBadQuillDelta            = 105

	// Email verification error

	ErrCodeNoEmailVerification = 106

	// Invite code errors

	ErrCodeInvalidInviteCode = 201
	ErrCodeUsedInviteCode    = 202
	ErrCodeInvalidSignup     = 203

	// Enterprise errors
	ErrCodePersonnelNotApproved = 301
)
