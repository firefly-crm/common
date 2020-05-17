package exchanges

import (
	"errors"
	"github.com/firefly-crm/common/rabbit"
)

type Exchange int8

const (
	FireflyCRMTelegramUpdates = iota + 1
)

type ExchangeKV struct {
	ID   Exchange
	Opts rabbit.ExchangeOptions
}

var exchanges = []*ExchangeKV{
	{
		ID: FireflyCRMTelegramUpdates,
		Opts: rabbit.ExchangeOptions{
			Name:    "crm-telegram-updates",
			Type:    "topic",
			Durable: true,
		},
	},
}

var exchangesByID = make(map[Exchange]*ExchangeKV)

func init() {
	for _, v := range exchanges {
		exchangesByID[v.ID] = v
	}
}

func ExchangeByID(id Exchange) (*ExchangeKV, error) {
	v, ok := exchangesByID[id]
	if !ok {
		return nil, errors.New("not found exchange")
	}

	return v, nil
}

func MustExchangeByID(id Exchange) *ExchangeKV {
	v, err := ExchangeByID(id)
	if err != nil {
		panic(err)
	}

	return v
}
