package main

import (
	"log"
	"os"

	"github.com/ski7777/csd-sll-mapper/internal/dwg"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf(
			"usage: %s drawing.dwg",
			os.Args[0],
		)
	}

	filename := os.Args[1]

	objs, err, warnings := dwg.LoadDWG(filename, []string{"Infostand"})
	if err != nil {
		log.Fatalln(err)
	}
	if len(warnings) > 0 {
		for _, w := range warnings {
			log.Println(w.Error())
		}
	}
	log.Println(objs)
}
