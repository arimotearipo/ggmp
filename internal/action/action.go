package action

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/arimotearipo/ggmp/internal/database"
	"github.com/arimotearipo/ggmp/internal/encryption"
	"github.com/arimotearipo/ggmp/internal/types"
	"github.com/arimotearipo/ggmp/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type session struct {
	username  string
	masterKey []byte
	id        int
}

type Action struct {
	db   *database.Database
	sess *session
}

func NewAction(db *database.Database) *Action {
	return &Action{db: db, sess: nil}
}

// === LOGINS ====
func (a *Action) AddPassword(uri, username, password string) error {
	encryptedPassword, err := encryption.Encrypt(password, a.sess.masterKey)
	if err != nil {
		return err
	}

	encryptedUsername, err := encryption.Encrypt(username, a.sess.masterKey)
	if err != nil {
		return err
	}

	encryptedURI, err := encryption.Encrypt(uri, a.sess.masterKey)
	if err != nil {
		return err
	}

	err = a.db.AddPassword(a.sess.id, encryptedURI, encryptedUsername, encryptedPassword)
	if err != nil {
		return err
	}

	return nil
}

func (a *Action) GetPassword(selectedUri types.URI) (username string, password string, err error) {
	username, encryptedPassword, err := a.db.GetPassword(selectedUri.Id, a.sess.id)
	if err != nil {
		return "", "", err
	}

	password, err = encryption.Decrypt(encryptedPassword, a.sess.masterKey)
	if err != nil {
		return "", "", err
	}

	username, err = encryption.Decrypt(username, a.sess.masterKey)
	if err != nil {
		return "", "", err
	}

	utils.ClipboardWriteAndErase(password, 15)

	return username, password, nil
}

func (a *Action) ListURIs() ([]types.URI, error) {
	uris, err := a.db.ListURIs(a.sess.id)
	if err != nil {
		return nil, err
	}

	var decryptedUris []types.URI

	for _, uri := range uris {
		uri.Uri, err = encryption.Decrypt(uri.Uri, a.sess.masterKey)
		if err != nil {
			return nil, err
		}
		decryptedUris = append(decryptedUris, uri)
	}

	return decryptedUris, nil
}

func (a *Action) DeletePassword(uriId int) error {
	err := a.db.DeleteAccount(uriId, a.sess.id)
	if err != nil {
		return err
	}

	return nil
}

func (a *Action) UpdatePassword(uriId int, username, password string) error {
	encryptedPassword, err := encryption.Encrypt(password, a.sess.masterKey)
	if err != nil {
		return err
	}

	err = a.db.UpdatePassword(uriId, username, encryptedPassword)
	if err != nil {
		return err
	}

	return nil
}

// === MASTER ACCOUNT ===

// Will prompt the user for username and password and
// proceeds to compare the hash and password
func (a *Action) Login(username, password string) error {
	id, hashedPassword, salt, err := a.db.GetMasterAccount(username)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return err
	}

	masterKey := encryption.DeriveKey([]byte(password), salt)

	a.sess = &session{username, masterKey, id}

	return nil
}

func (a *Action) Logout() {
	a.sess.username = ""
	a.sess.masterKey = nil
	a.sess.id = -1
	a.sess = nil
}

func (a *Action) Register(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	salt, _ := encryption.GenerateSalt()

	err = a.db.AddMasterAccount(username, string(hashedPassword), salt)
	if err != nil {
		return err
	}

	return nil
}

func (a *Action) DeleteMasterAccount(username, password string) error {
	if err := a.Login(username, password); err != nil {
		return err
	}

	err := a.db.DeleteMasterAccount(username)
	if err != nil {
		return err
	}

	return nil
}

func (a *Action) ListMasterAccounts() ([]string, error) {
	accounts, err := a.db.ListMasterAccounts()
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (a *Action) UpdateMasterPassword(newMasterPassword string) error {
	salt, _ := encryption.GenerateSalt()

	// derive new master key and saved old master in case we need to rollback transaction
	newMasterKey := encryption.DeriveKey([]byte(newMasterPassword), salt)
	savedOldMasterKey := make([]byte, len(a.sess.masterKey))
	copy(savedOldMasterKey, a.sess.masterKey)

	err := a.db.BeginTx()
	if err != nil {
		return err
	}
	defer a.db.RollbackTx(func() {
		a.sess.masterKey = savedOldMasterKey
	})

	// get all the uris associated to the logged in master account
	uris, err := a.db.ListURIs(a.sess.id)
	if err != nil {
		return err
	}

	for _, uri := range uris {
		// get the decrypted password and username
		// GetPassword() will perform decryption on saved passwords therefore need to use previous master key
		a.sess.masterKey = savedOldMasterKey
		username, decryptedPassword, err := a.GetPassword(uri)
		if err != nil {
			return err
		}

		// call update. UpdatePassword() will re-encrypt the password with new master key
		// UpdatePassword() will perform encryption on new master key therefore need to use new master key
		a.sess.masterKey = newMasterKey
		err = a.UpdatePassword(uri.Id, username, decryptedPassword)
		if err != nil {
			return err
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newMasterPassword), 12)
	if err != nil {
		return err
	}

	err = a.db.ChangeMasterPassword(a.sess.id, string(hashedPassword), salt)
	if err != nil {
		return err
	}

	err = a.db.CommitTx()
	if err != nil {
		return err
	}

	a.sess.masterKey = newMasterKey
	return nil
}

const (
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	numbers   = "0123456789"
	special   = "!@#$%^&*()"
)

func (a *Action) GeneratePassword(c types.PasswordGeneratorConfig) (string, error) {
	sum := c.LowercaseLength + c.UppercaseLength + c.NumericLength + c.SpecialLength

	if sum > c.TotalLength {
		e := fmt.Sprintf("total length should not be less than %d", sum)
		return "", errors.New(e)
	}

	var characterPool strings.Builder
	var result strings.Builder

	// Generate minimum required characters
	if c.UppercaseLength > 0 {
		for i := 0; i < c.UppercaseLength; i++ {
			result.WriteByte(uppercase[rand.Intn(len(uppercase))])
		}
		characterPool.WriteString(uppercase)
	}

	if c.LowercaseLength > 0 {
		for i := 0; i < c.LowercaseLength; i++ {
			result.WriteByte(lowercase[rand.Intn(len(lowercase))])
		}
		characterPool.WriteString(lowercase)
	}

	if c.SpecialLength > 0 {
		for i := 0; i < c.SpecialLength; i++ {
			result.WriteByte(special[rand.Intn(len(special))])
		}
		characterPool.WriteString(special)
	}

	if c.NumericLength > 0 {
		for i := 0; i < c.NumericLength; i++ {
			result.WriteByte(numbers[rand.Intn(len(numbers))])
		}
		characterPool.WriteString(numbers)
	}

	// at this point, if result, user has set all config values to 0
	if result.Len() == 0 {
		return "", errors.New("no characters generated")
	}

	// Fill the remaining length with random characters
	for i := result.Len(); i < c.TotalLength; i++ {
		result.WriteByte(characterPool.String()[rand.Intn(characterPool.Len())])
	}

	// Shuffle the string
	resultRunes := []rune(result.String())
	rand.Shuffle(len(resultRunes), func(i, j int) {
		resultRunes[i], resultRunes[j] = resultRunes[j], resultRunes[i]
	})

	generatedPassword := string(resultRunes)

	utils.ClipboardWriteAndErase(generatedPassword, 15)

	return generatedPassword, nil
}
