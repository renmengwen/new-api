package service

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

type ExcelFileSpec struct {
	FileNamePrefix string
	SheetName      string
	Headers        []string
	Rows           [][]string
}

func BuildExcelFile(spec ExcelFileSpec) (string, []byte, error) {
	file := excelize.NewFile()
	sheetName := spec.SheetName
	defaultSheetName := file.GetSheetName(0)

	if err := file.SetSheetName(defaultSheetName, sheetName); err != nil {
		return "", nil, err
	}

	for index, header := range spec.Headers {
		cellName, err := excelize.CoordinatesToCellName(index+1, 1)
		if err != nil {
			return "", nil, err
		}
		if err := file.SetCellValue(sheetName, cellName, header); err != nil {
			return "", nil, err
		}
	}

	for rowIndex, row := range spec.Rows {
		for colIndex, value := range row {
			cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err != nil {
				return "", nil, err
			}
			if err := file.SetCellValue(sheetName, cellName, value); err != nil {
				return "", nil, err
			}
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return "", nil, err
	}

	fileName := fmt.Sprintf("%s_%s.xlsx", spec.FileNamePrefix, time.Now().Format("2006-01-02_15-04-05"))
	return fileName, buffer.Bytes(), nil
}
