package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/ski7777/csd-sll-mapper/internal/dwg"
	"github.com/ski7777/csd-sll-mapper/internal/pretix"
)

func loaddwg() (err error) {
	if len(os.Args) != 2 {
		err = errors.New("Usage: go run main.go <filename.dwg>")
		return
	}

	filename := os.Args[1]

	objs, err, warnings := dwg.LoadDWG(filename, []string{"Infostand", "Gastrostand"})
	if err != nil {
		err = fmt.Errorf("failed to load DWG file: %w", err)
		return
	}
	if len(warnings) > 0 {
		for _, w := range warnings {
			log.Println(w.Error())
		}
	}
	log.Println(objs)
	return
}

func loadptx() (err error) {
	url, ok := os.LookupEnv("CSD_PTX_URL")
	if !ok {
		err = errors.New("CSD_PTX_URL environment variable is not set")
		return
	}

	token, ok := os.LookupEnv("CSD_PTX_TOKEN")
	if !ok {
		err = errors.New("CSD_PTX_TOKEN environment variable is not set")
		return
	}
	ptx := pretix.NewPretixClient(url, token)
	organizer, ok := os.LookupEnv("CSD_PTX_ORGANIZER")
	if !ok {
		err = errors.New("CSD_PTX_ORGANIZER environment variable is not set")
		return
	}
	events, err := ptx.GetEvents(organizer)
	if err != nil {
		err = fmt.Errorf("failed to get Pretix events: %w", err)
		return
	}
	var orders []pretix.Order
	for _, e := range events {
		orders, err = ptx.GetOrders(organizer, e.Slug)
		if err != nil {
			err = fmt.Errorf("failed to get Pretix orders: %w", err)
			return
		}

		for _, o := range orders {
			if !(o.Status == "p" || (o.Status == "n" && o.ValidIfPending)) {
				continue
			}
			for _, p := range o.Positions {
				// just a test, not implementing 245,246 and variants of 37 yet
				if !slices.Contains([]int{37, 38, 39}, p.ItemId) {
					continue
				}
				for _, a := range p.Answers {
					if slices.Contains([]string{"standname", "standnummer"}, a.QuestionIdentifier) {
						log.Printf("Order %s, Position %d, Question %d / %s: Answer %s", o.Code, p.Id, a.QuestionID, a.QuestionIdentifier, a.Answer)
					}
				}
			}
		}
	}
	return
}

func main() {
	/* ToDo
	load a central JSON file with all events mapped to a dwg each
	in the json we should also map products to certain output definitions
	*/

	loaddwg()
	loadptx()

}
