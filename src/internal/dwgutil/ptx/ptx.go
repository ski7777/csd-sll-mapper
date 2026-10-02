package ptx

import (
	"errors"
	"os"
	"sync"

	"github.com/ski7777/csd-sll-mapper/internal/config"
	"github.com/ski7777/csd-sll-mapper/internal/ptxutil"
)

type PtxData map[string]EventData

type EventData []ItemData

type ItemData struct {
	OrderCode        string
	OrderPositionID  int
	ProductID        int
	ProductVariantID *int
	Answers          map[string]string
}

func LoadPTX(conf *config.Config) (data PtxData, err error) {
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
	ptxc := ptxutil.NewPretixClient(url, token)
	organizer, ok := os.LookupEnv("CSD_PTX_ORGANIZER")
	if !ok {
		err = errors.New("CSD_PTX_ORGANIZER environment variable is not set")
		return
	}
	data = make(PtxData)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for en, e := range conf.Events {
		wg.Add(1)
		go func(eventName string, eventConfig config.EventConfig) {
			defer wg.Done()
			mu.Lock()
			data[eventName] = []ItemData{}
			mu.Unlock()
			orders, err := ptxc.GetOrders(organizer, eventName)
			if err != nil {
				err = errors.Join(err, errors.New("failed to get Pretix orders for event "+eventName))
				return
			}
			for _, o := range orders {
				if !(o.Status == "p" || (o.Status == "n" && o.ValidIfPending)) {
					continue
				}
				for _, p := range o.Positions {
					_, ok := eventConfig.ProductMapper.GetPtxProductMapping(p.ItemId, p.ItemVariationId)
					if !ok {
						continue
					}
					itemData := ItemData{
						OrderCode:        o.Code,
						OrderPositionID:  p.Id,
						ProductID:        p.ItemId,
						ProductVariantID: p.ItemVariationId,
						Answers: func() map[string]string {
							answers := make(map[string]string)
							for _, a := range p.Answers {
								answers[a.QuestionIdentifier] = a.Answer
							}
							return answers
						}(),
					}
					mu.Lock()
					data[eventName] = append(data[eventName], itemData)
					mu.Unlock()
				}
			}
		}(en, e)
	}
	wg.Wait()
	return

}
