package pretix

import (
	"pretix-excel-analyze/internal/config"
)

func GetEventStats(ptx *PretixClient, c *config.Config, event string) (stats map[string]float64, err error) {
	items, err := ptx.GetItems(c.Pretix.Organizer, event)
	if err != nil {
		return
	}

	itemcat := map[ItemKey]string{}
	for _, item := range items {
		cat, ok := item.MetaData[c.Pretix.CategoryKey]
		if ok {
			itemcat[ItemKey{Id: item.Id}] = cat
		}
		for _, variation := range item.Variations {
			vcat, vok := variation.MetaData[c.Pretix.CategoryKey]
			if vok {
				itemcat[ItemKey{Id: item.Id, Variation: true, VariationId: variation.Id}] = vcat
			} else if ok {
				itemcat[ItemKey{Id: item.Id, Variation: true, VariationId: variation.Id}] = cat
			}
		}
	}

	orders, err := ptx.GetOrders("csd", "2026")
	if err != nil {
		return
	}
	stats = make(map[string]float64)
	for _, order := range orders {
		if order.Status == "c" /*cancelled*/ {
			continue
		}
		if order.RequireApproval {
			continue
		}
		positioncats := map[int]string{}
		for _, pos := range order.Positions {
			if pos.AddonTo != nil {
				continue
			}
			cat, ok := itemcat[NewItemKeyFromOrderPosition(pos)]
			if ok {
				positioncats[pos.Id] = cat
			}
		}
		for _, pos := range order.Positions {
			if pos.AddonTo == nil {
				continue
			}
			cat, ok := itemcat[NewItemKeyFromOrderPosition(pos)]
			if ok && cat != c.Pretix.Categories.InheritParent {
				positioncats[pos.Id] = cat
			} else if pcat, ok := positioncats[*pos.AddonTo]; cat == c.Pretix.Categories.InheritParent && ok {
				positioncats[pos.Id] = pcat
			}
		}
		var price float64
		for _, pos := range order.Positions {
			cat, ok := positioncats[pos.Id]
			if !ok {
				continue
			}
			price, err = pos.Price.Float64()
			if err != nil {
				return
			}
			cs, _ := stats[cat]
			stats[cat] = cs + price
		}
		for _, fee := range order.Fees {
			price, err = fee.Value.Float64()
			if err != nil {
				return
			}
			cs, _ := stats[c.Pretix.FeeCategory]
			stats[c.Pretix.FeeCategory] = cs + price
		}
	}
	return
}
