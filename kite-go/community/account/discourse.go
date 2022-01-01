package account

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"net/url"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/community"
)

// ForumLogin describes the primary SSO data structure and can be used to describe SSO errors in detail
type ForumLogin struct {
	Sso string `json:"sso"`
	Sig string `json:"sig"`
}

func (e ForumLogin) Error() string {
	return fmt.Sprintf("account/discourse: invalid signature, payload: %s, %s", e.Sso, e.Sig)
}

type discourseManager struct {
	ssoSecret []byte
}

func newDiscourseManager(ssoSecret string) *discourseManager {
	return &discourseManager{
		ssoSecret: []byte(ssoSecret),
	}
}

func (d *discourseManager) SingleSignOn(submission ForumLogin, user *community.User) (ForumLogin, error) {
	valid, err := d.verify(submission)
	if err != nil {
		return ForumLogin{}, err
	}
	if !valid {
		return ForumLogin{}, submission
	}

	nonce, err := d.retrieveNonce(submission)
	if err != nil {
		return ForumLogin{}, err
	}

	payload := url.Values{}
	payload.Set("nonce", nonce)
	payload.Set("email", user.Email)
	payload.Set("external_id", strconv.FormatInt(user.ID, 10))
	if !user.EmailVerified {
		// TODO: capture email activation on our end instead of with discourse
		payload.Set("require_activation", "true")
	}
	if user.IsInternal {
		payload.Set("moderator", "true")
	}

	return d.createForumLogin(payload), nil
}

func (d *discourseManager) newSignature() hash.Hash {
	return hmac.New(sha256.New, d.ssoSecret)
}

func (d *discourseManager) sign(sso string) string {
	signature := d.newSignature()
	signature.Write([]byte(sso))
	return hex.EncodeToString(signature.Sum(nil))
}

func (d *discourseManager) verify(submission ForumLogin) (bool, error) {
	sso := submission.Sso
	sig := submission.Sig

	decoded, err := hex.DecodeString(sig)
	if err != nil {
		return false, err
	}

	expected, err := hex.DecodeString(d.sign(sso))
	if err != nil {
		return false, err
	}

	return hmac.Equal(decoded, expected), nil
}

func (d *discourseManager) retrieveNonce(submission ForumLogin) (string, error) {
	sso := submission.Sso

	unescaped, err := url.PathUnescape(sso)
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(unescaped)
	if err != nil {
		return "", err
	}

	values, err := url.ParseQuery(string(decoded))
	if err != nil {
		return "", err
	}

	return values.Get("nonce"), nil
}

func (d *discourseManager) createSso(values url.Values) string {
	str := values.Encode()

	encoded := base64.StdEncoding.EncodeToString([]byte(str))

	return url.PathEscape(encoded)
}

func (d *discourseManager) createForumLogin(values url.Values) ForumLogin {
	sso := d.createSso(values)
	sig := d.sign(sso)
	return ForumLogin{
		Sso: sso,
		Sig: sig,
	}
}
