//go:build integration

package excel

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
)

type excelTestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
}

func (s *excelTestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func TestExcelSuite(t *testing.T) {
	suite.Run(t, new(excelTestSuite))
}

func (s *excelTestSuite) Test_ReadFile() {
	reader, err := getFileReader()
	s.NoError(err)
	defer func() {
		reader.Close()
	}()

	// get xls content
	content, err := ReadFromReader(reader)
	s.NoError(err)
	// verify rows count
	s.Len(content, 2)
	// verify header
	hdr := content[0]
	// check proper len
	s.Len(hdr, 3)
	// check cells
	s.Equal("Hdr 1", hdr[0])
	s.Equal("Hdr 2", hdr[1])
	s.Equal("Hdr 3", hdr[2])
	// verify row
	row := content[1]
	// check proper len
	s.Len(row, 3)
	// check cells
	s.Equal("Cell 1 1", row[0])
	s.Equal("Cell 1 2", row[1])
	s.Equal("Cell 1 3", row[2])
}

func getFileReader() (io.ReadCloser, error) {
	// path to test file to read
	filePath, err := getFilePath()
	if err != nil {
		return nil, err
	}

	// read test file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewBuffer(content)), nil
}

func getFilePath() (string, error) {
	// file is expected to be found in the WD
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return path.Join(workDir, "test.xlsx"), nil
}
