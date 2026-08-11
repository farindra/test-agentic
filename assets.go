// Package assets meng-embed hasil build Vue (web/dist) ke dalam binary Go,
// biar deploy cukup satu file/image — nggak perlu web server terpisah buat
// static file. File ini WAJIB di root modul: go:embed cuma boleh nunjuk
// subdirektori dari lokasi file sumbernya sendiri (nggak boleh "..").
package assets

import "embed"

//go:embed all:web/dist
var DistFS embed.FS
