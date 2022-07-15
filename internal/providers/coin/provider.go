package coin_provider

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/aspiration-labs/pyggpot/internal/models"
	coin_service "github.com/aspiration-labs/pyggpot/rpc/go/coin"
	"github.com/twitchtv/twirp"
)

type coinServer struct {
	DB *sql.DB
}

func New(db *sql.DB) *coinServer {
	return &coinServer{
		DB: db,
	}
}

func (s *coinServer) AddCoins(ctx context.Context, request *coin_service.AddCoinsRequest) (*coin_service.CoinsListResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, twirp.InvalidArgumentError(err.Error(), "")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	for _, coin := range request.Coins {
		fmt.Println(coin)
		newCoin := models.Coin{
			PotID:        request.PotId,
			Denomination: int32(coin.Kind),
			CoinCount:    coin.Count,
		}
		err = newCoin.Save(tx)
		if err != nil {
			return nil, twirp.InvalidArgumentError(err.Error(), "")
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, twirp.NotFoundError(err.Error())
	}

	return &coin_service.CoinsListResponse{
		Coins: request.Coins,
	}, nil
}

func (s *coinServer) RemoveCoins(ctx context.Context, request *coin_service.RemoveCoinsRequest) (*coin_service.CoinsListResponse, error) {
	rand.Seed(time.Now().UnixNano())
	if err := request.Validate(); err != nil {
		return nil, twirp.InvalidArgumentError(err.Error(), "")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	coins := []*coin_service.Coins{}
	coinsInPot, err := models.CoinsInPotsByPot_id(tx, int(request.PotId))
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	rand.Shuffle(len(coinsInPot), func(i, j int) { coinsInPot[i], coinsInPot[j] = coinsInPot[j], coinsInPot[i] })
	total := request.Count
	for _, c := range coinsInPot {
		coin, err := models.CoinByID(tx, c.ID)
		if err != nil {
			return nil, twirp.InternalError(err.Error())
		}
		if total == 0 {
			break
		}
		count := total
		if total > c.CoinCount {
			coin.CoinCount = 0
			count = c.CoinCount
			total = total - c.CoinCount
		} else {
			coin.CoinCount = c.CoinCount - total
			total = 0
		}

		coins = append(coins, &coin_service.Coins{
			Kind:  coin_service.Coins_Kind(c.Denomination),
			Count: count,
		})
		coins = append(coins, &coin_service.Coins{
			Kind:  coin_service.Coins_Kind(c.Denomination),
			Count: count,
		})
		if coin.CoinCount == 0 {
			if err := coin.Delete(tx); err != nil {
				return nil, twirp.InternalError(err.Error())
			}
		} else {
			if err := coin.Update(tx); err != nil {
				return nil, twirp.InternalError(err.Error())
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return nil, twirp.NotFoundError(err.Error())
	}

	return &coin_service.CoinsListResponse{Coins: coins}, nil

}
