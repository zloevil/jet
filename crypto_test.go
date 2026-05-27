package jet

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type cryptoTestSuite struct {
	Suite
}

func (s *cryptoTestSuite) SetupSuite() {
	s.Suite.Init(func() CLogger { return L(InitLogger(&LogConfig{Level: TraceLevel})) })
}

func TestCryptoSuite(t *testing.T) {
	suite.Run(t, new(cryptoTestSuite))
}

func (s *cryptoTestSuite) Test_Encrypt_Decrypt() {
	key := "this_must_be_of_32_byte_length!!"
	val := "some-text"
	encStr, err := EncryptString(s.Ctx, key, val)
	s.NoError(err)
	s.NotEmpty(encStr)
	decStr, err := DecryptString(s.Ctx, key, encStr)
	s.NoError(err)
	s.NotEmpty(decStr)
	s.Equal(decStr, val)
}
