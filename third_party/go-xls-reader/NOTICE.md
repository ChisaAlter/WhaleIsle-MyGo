# go-xls-reader production source

Source: https://github.com/ronohara/go-xls-reader/tree/v0.1.1
Version: v0.1.1, tree 8062b13d1d18161c6266007fb772f12893815bd3.
License: Apache-2.0; the upstream LICENSE is included unchanged.

Only production Go sources are included. Tests, sample workbooks and the test-only github.com/tealeg/xlsx dependency are omitted.

WhaleIsle adaptations: BOOLERR decoding through the typed-cell API; BIFF5 code pages 936 (GBK), 950 (Big5), 54936 (GB18030); workbook-order worksheet metadata without parsing every sheet; ignore hyperlink metadata so it does not replace stored cell text; stop workbook globals at EOF and report unsupported FILEPASS encryption. Formula values remain the stored cache; macros, formulas and external links are never executed.
