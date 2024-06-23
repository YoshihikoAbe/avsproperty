package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/YoshihikoAbe/avsproperty"
)

func main() {
	var unicode bool

	flag.BoolVar(&unicode, "u", false, "Set output encoding to UTF-8")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILENAME \n\nProperty format conversion tool\n\nList of available options:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	filename := flag.Arg(0)
	if filename == "" {
		flag.Usage()
		os.Exit(1)
	}

	prop := &avsproperty.Property{}
	if err := prop.ReadFile(filename); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if prop.Settings.Format == avsproperty.FormatBinary {
		prop.Settings.Format = avsproperty.FormatPrettyXML
	} else {
		prop.Settings.Format = avsproperty.FormatBinary
	}
	if unicode {
		prop.Settings.Encoding = avsproperty.EncodingUTF8
	}

	if err := prop.Write(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
