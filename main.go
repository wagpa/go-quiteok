package main

import (
	"errors"
	"flag"
	"fmt"
	"go_quiteok/qoi"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

type parameters struct {
	inPath  string
	outPath string
	decode  bool
}

func getInput() (parameters, error) {
	in := flag.String("in", "in.qoi", "the parameters file (qoi or png)")
	out := flag.String("out", "out.png", "the output file (png or qoi)")
	flag.Parse()

	code := flag.Arg(0)
	if code != "encode" && code != "decode" {
		fmt.Println(flag.Args())
		return parameters{}, fmt.Errorf("supported actions are `encode` and `decode`, not %s", code)
	}

	params := parameters{
		inPath:  *in,
		outPath: *out,
		decode:  code == "decode",
	}

	if params.decode {
		if filepath.Ext(params.inPath) != ".qoi" || filepath.Ext(params.outPath) != ".png" {
			return parameters{}, errors.New("`decode` expects the first param to be a qoi and the second a png file")
		}
	} else if filepath.Ext(params.outPath) != ".qoi" || filepath.Ext(params.inPath) != ".png" {
		return parameters{}, fmt.Errorf("`encode` expects the first param to be a png and the second a qoi file, not %s %s", filepath.Ext(params.inPath), filepath.Ext(params.outPath))
	}

	return params, nil
}

func main() {

	params, err := getInput()
	if err != nil {
		panic(err)
	}

	reader, oErr := os.Open(params.inPath)
	if oErr != nil {
		panic(oErr)
	}

	var decoded image.Image
	var dErr error = nil
	if params.decode {
		decoded, dErr = qoi.Decode(reader)
	} else {
		decoded, dErr = png.Decode(reader)
	}
	if dErr != nil {
		panic(dErr)
	}

	writer, ofErr := os.OpenFile(params.outPath, os.O_CREATE|os.O_WRONLY, 0644)
	if ofErr != nil {
		panic(ofErr)
	}

	var eErr error = nil
	if params.decode {
		eErr = png.Encode(writer, decoded)
	} else {
		eErr = qoi.Encode(writer, decoded)
	}
	if eErr != nil {
		panic(eErr)
	}
}
