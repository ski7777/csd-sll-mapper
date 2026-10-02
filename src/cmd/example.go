package main

import (
	"errors"
	"log"
	"os"
	"sync"

	"github.com/ski7777/csd-sll-mapper/internal/config"
	"github.com/ski7777/csd-sll-mapper/internal/dwg"
	"github.com/ski7777/csd-sll-mapper/internal/dwgutil"
	"github.com/ski7777/csd-sll-mapper/internal/dwgutil/ptx"
)

func loadConfig(filePath string) (cfg *config.Config, err error) {
	cfgfn, ok := os.LookupEnv("CSD_CONFIG")
	if !ok {
		err = errors.New("CSD_PTX_URL environment variable is not set")
		return
	}
	cfg, err = config.NewConfigFromFile(cfgfn)
	return
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	var dwgdata map[string][]dwgutil.Object
	var ptxdata ptx.PtxData
	errs := []error{}
	warnings := []error{}
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var ierr error
		var iwarnings []error
		dwgdata, ierr, iwarnings = dwg.LoadAllDWGs(cfg)
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, ierr)
		warnings = append(warnings, iwarnings...)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var ierr error
		mu.Lock()
		defer mu.Unlock()
		ptxdata, ierr = ptx.LoadPTX(cfg)
		errs = append(errs, ierr)
	}()
	wg.Wait()
	if len(errs) > 0 {
		log.Println("Errors occured")
		for _, e := range errs {
			log.Println("e")
			log.Println(e)
		}
		return
	}
	if len(warnings) > 0 {
		log.Println("Warnings occurred")
		for _, w := range warnings {
			log.Println("w")
			log.Println(w)
		}
	}
	log.Println(dwgdata)
	log.Println(ptxdata)
}
