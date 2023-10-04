package qoi

import "embed"

//go:embed all:data/*
var data embed.FS

var testFiles = []string{
	"testcard",
	"dice",
	"kodim10",
	"kodim23",
	"qoi_logo",
	"testcard_rgba",
	"wikipedia_008",
}
