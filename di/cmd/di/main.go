// di is a compile-time dependency injection code generator.
//
// It scans Go packages for injector functions (functions containing a call to
// di.Build) and provider sets (di.NewSet), then writes a di_gen.go file that
// implements the injectors.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	di "github.com/jdkhome/gdk/di/internal/di"
)

func main() {
	headerFile := flag.String("header_file", "", "path to file to insert as a header in di_gen.go")
	prefixFileName := flag.String("output_file_prefix", "", "string to prepend to output file names")
	tags := flag.String("tags", "", "append build tags to the default dibuild")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("failed to get working directory: ", err)
	}

	opts := &di.GenerateOptions{
		PrefixOutputFile: *prefixFileName,
		Tags:             *tags,
	}
	if *headerFile != "" {
		data, err := os.ReadFile(*headerFile)
		if err != nil {
			log.Fatalf("failed to read header file %q: %v", *headerFile, err)
		}
		opts.Header = data
	}

	pkgs := flag.Args()
	if len(pkgs) == 0 {
		pkgs = []string{"."}
	}

	outs, errs := di.Generate(context.Background(), wd, os.Environ(), pkgs, opts)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Println(e)
		}
		log.Fatal("generate failed")
	}

	success := true
	for _, out := range outs {
		if len(out.Errs) > 0 {
			for _, e := range out.Errs {
				log.Println(e)
			}
			log.Printf("%s: generate failed\n", out.PkgPath)
			success = false
			continue
		}
		if len(out.Content) == 0 {
			continue
		}
		if err := out.Commit(); err != nil {
			log.Printf("%s: failed to write %s: %v\n", out.PkgPath, out.OutputPath, err)
			success = false
			continue
		}
		log.Printf("%s: wrote %s\n", out.PkgPath, out.OutputPath)
	}
	if !success {
		log.Fatal("at least one generate failure")
	}
}
