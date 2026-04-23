package service

import "github.com/xuri/excelize/v2"

type AsyncExportPage struct {
	Rows [][]string
	Done bool
}

type AsyncExportWriterSpec struct {
	FilePath  string
	SheetName string
	Headers   []string
	PageSize  int
	FetchPage func(page int, pageSize int) (AsyncExportPage, error)
}

func WriteAsyncExportXLSX(spec AsyncExportWriterSpec) (int64, error) {
	file := excelize.NewFile()
	defer file.Close()

	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, spec.SheetName); err != nil {
		return 0, err
	}
	streamWriter, err := file.NewStreamWriter(spec.SheetName)
	if err != nil {
		return 0, err
	}

	headerRow := make([]any, len(spec.Headers))
	for i, header := range spec.Headers {
		headerRow[i] = header
	}
	if err := streamWriter.SetRow("A1", headerRow); err != nil {
		return 0, err
	}

	rowIndex := 2
	pageSize := spec.PageSize
	if pageSize <= 0 {
		pageSize = 1000
	}

	for page := 1; ; page++ {
		pageData, err := spec.FetchPage(page, pageSize)
		if err != nil {
			return 0, err
		}
		for _, row := range pageData.Rows {
			cellName, err := excelize.CoordinatesToCellName(1, rowIndex)
			if err != nil {
				return 0, err
			}
			rowValues := make([]any, len(row))
			for i, value := range row {
				rowValues[i] = value
			}
			if err := streamWriter.SetRow(cellName, rowValues); err != nil {
				return 0, err
			}
			rowIndex++
		}
		if pageData.Done {
			break
		}
	}

	if err := streamWriter.Flush(); err != nil {
		return 0, err
	}
	if err := file.SaveAs(spec.FilePath); err != nil {
		return 0, err
	}
	return int64(rowIndex - 2), nil
}
