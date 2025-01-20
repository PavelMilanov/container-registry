package secure

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func Hashed(data string) string {
	hashedData, err := bcrypt.GenerateFromPassword([]byte(data), bcrypt.DefaultCost)
	if err != nil {
		logrus.Debug(err)
	}
	return string(hashedData)
}

func ValidateHash(data string, hashedData []byte) error {
	if err := bcrypt.CompareHashAndPassword(hashedData, []byte(data)); err != nil {
		logrus.Debug(err)
		return err
	}
	return nil
}
