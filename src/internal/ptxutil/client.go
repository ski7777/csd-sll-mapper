package ptxutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type PretixClient struct {
	apikey      string
	instanceurl string
	client      *http.Client
}

func (p *PretixClient) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Token "+p.apikey)
	return http.DefaultTransport.RoundTrip(req)
}

func (p *PretixClient) do(req *http.Request) (res []byte, err error) {
	hres, err := p.client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = hres.Body.Close()
	}()
	if hres.StatusCode < 200 || hres.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("unexpected status code %d", hres.StatusCode))
		return
	}
	res, err = io.ReadAll(hres.Body)
	return
}

type PaginatedResponse[T any] struct {
	Results []T     `json:"results"`
	Next    *string `json:"next"`
}

func getPaginated[T any](p *PretixClient, url string) (res PaginatedResponse[T], err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	rres, err := p.do(req)
	if err != nil {
		return
	}
	res = PaginatedResponse[T]{}
	err = json.Unmarshal(rres, &res)
	if err != nil {
		return
	}
	if res.Next != nil {
		nextres, err := getPaginated[T](p, *res.Next)
		if err != nil {
			return res, err
		}
		res.Results = append(res.Results, nextres.Results...)
	}
	return
}

type Event struct {
	Slug string `json:"slug"`
}

func (p *PretixClient) GetEvents(organizer string) (res []Event, err error) {
	pres, err := getPaginated[Event](p, fmt.Sprintf("%s/api/v1/organizers/%s/events/", p.instanceurl, organizer))
	if err != nil {
		return
	}
	res = pres.Results
	return
}

type Answer struct {
	QuestionID         int      `json:"question"`
	Answer             string   `json:"answer"`
	QuestionIdentifier string   `json:"question_identifier"`
	Options            []int    `json:"options"`
	OptionIdentifiers  []string `json:"option_identifiers"`
}

type OrderPosition struct {
	Id              int         `json:"id"`
	ItemId          int         `json:"item"`
	ItemVariationId *int        `json:"variation"`
	Price           json.Number `json:"price"`
	AddonTo         *int        `json:"addon_to"`
	Answers         []Answer    `json:"answers"`
}

type OrderFee struct {
	Value json.Number `json:"value"`
}

type Order struct {
	Code            string          `json:"code"`
	Status          string          `json:"status"`
	ValidIfPending  bool            `json:"valid_if_pending"`
	RequireApproval bool            `json:"require_approval"`
	Positions       []OrderPosition `json:"positions"`
	Fees            []OrderFee      `json:"fees"`
}

func (p *PretixClient) GetOrders(organizer, event string) (res []Order, err error) {
	pres, err := getPaginated[Order](p, fmt.Sprintf("%s/api/v1/organizers/%s/events/%s/orders/", p.instanceurl, organizer, event))
	if err != nil {
		return
	}
	res = pres.Results
	return
}

func NewPretixClient(instanceurl, apikey string) (client *PretixClient) {
	client = &PretixClient{
		apikey:      apikey,
		instanceurl: instanceurl,
	}
	client.client = &http.Client{Transport: client}
	return
}
