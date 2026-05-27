package excel

import (
	"io"
	"slices"

	ex "github.com/xuri/excelize/v2"
)

func ReadFromReader(file io.ReadCloser) ([][]string, error) {
	// open excel file as a stream
	xls, err := ex.OpenReader(file)
	if err != nil {
		return nil, ErrExcelOpenFile(err)
	}
	defer xls.Close()

	// prepare map of sheets and find sheet name to read rows from
	sheets := xls.GetSheetMap()
	sheetIndex := xls.GetActiveSheetIndex()

	if sheetIndex == 0 {
		// no active sheet selected, let's read the first sheet available
		// find sheet with the min key
		keys := make([]int, 0, len(sheets))
		for k := range sheets {
			keys = append(keys, k)
		}
		sheetIndex = slices.Min(keys)
	}
	// find out name of the sheet to read rows from
	sheetName, ok := sheets[sheetIndex]
	if !ok {
		return nil, ErrExcelOpenSheet()
	}

	// read rows from the sheet
	rows, err := xls.GetRows(sheetName)
	if err != nil {
		return nil, ErrExcelReadRows(err)
	}

	return rows, nil
}
