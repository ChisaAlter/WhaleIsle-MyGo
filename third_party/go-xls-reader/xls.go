package xls

import (
	"errors"
	"io"
	"os"

	"github.com/extrame/ole2"
)

// Open one xls file
func Open(file string) (*WorkBook, error) {
	if fi, err := os.Open(file); err == nil {
		return OpenReader(fi)
	} else {
		return nil, err
	}
}

// Open one xls file and return the closer
func OpenWithCloser(file string) (*WorkBook, io.Closer, error) {
	if fi, err := os.Open(file); err == nil {
		wb, err := OpenReader(fi)
		return wb, fi, err
	} else {
		return nil, nil, err
	}
}

// Open xls file from reader
//
// There is no charset argument: the old one was inert. It was passed straight
// to ole2.Open, whose source mentions it exactly once — as the parameter name —
// so any value at all behaved identically. What actually decides how text is
// decoded is the workbook's own CODEPAGE record, read at workbook.go:87 and
// consumed by get_string.
func OpenReader(reader io.ReadSeeker) (wb *WorkBook, err error) {
	var ole *ole2.Ole
	if ole, err = ole2.Open(reader, ""); err == nil {
		var dir []*ole2.File
		if dir, err = ole.ListDir(); err == nil {
			var book *ole2.File
			var root *ole2.File
			for _, file := range dir {
				name := file.Name()
				if name == "Workbook" {
					if book == nil {
						book = file
					}
					//book = file
					// break
				}
				if name == "Book" {
					book = file
					// break
				}
				if name == "Root Entry" {
					root = file
				}
			}
			if book != nil {
				wb = newWorkBookFromOle2(ole.OpenFile(book, root))
				if wb.Encrypted {
					return nil, errors.New("Encrypted XLS workbooks are not supported; use Open external")
				}
				return
			}
		}
	}
	return
}
