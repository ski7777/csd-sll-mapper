package dwg

import (
	"errors"
	"log"
	"sync"

	"github.com/ski7777/csd-sll-mapper/internal/config"
	"github.com/ski7777/csd-sll-mapper/internal/dwgutil"
)

func LoadAllDWGs(config *config.Config) (dwgs map[string][]dwgutil.Object, err error, warnings []error) {
	wg := sync.WaitGroup{}
	dwgs = make(map[string][]dwgutil.Object)
	warnings = []error{}
	errs := []error{}
	mu := sync.Mutex{}
	for en, e := range config.Events {
		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			objs, err, dwgwarnings := dwgutil.LoadDWG(filename, e.GetAllBlockNames())
			log.Println("Loaded DWG for event", en, "from file", filename, "with", len(objs), "objects")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			warnings = append(warnings, dwgwarnings...)
			dwgs[en] = objs
		}(e.DwgFilePath)
	}
	wg.Wait()
	err = errors.Join(errs...)
	return
}
