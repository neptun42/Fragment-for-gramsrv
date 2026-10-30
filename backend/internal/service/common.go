package service

import (
	"context"
	"strings"

	"lunax/internal/domain"
	"lunax/internal/network"
)

// откуда сервисы берут адаптер нужной приватки
type Networks interface {
	Get(ctx context.Context, serverID int64) (network.Network, error)
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// красивое имя для истории и инвентаря
func lotTitle(l domain.Lot) string {
	switch l.Kind {
	case domain.LotUsername:
		return "@" + l.Name
	case domain.LotGift:
		if l.Gift != nil {
			return l.Gift.Collection
		}
	}
	return l.Name
}

func itemFromLot(l domain.Lot, userID int64, value domain.Cents, source string) domain.Item {
	it := domain.Item{UserID: userID, LotID: l.ID, Title: lotTitle(l), Value: value, Source: source}
	switch l.Kind {
	case domain.LotUsername:
		it.Kind, it.Name = domain.ItemUsername, l.Name
	case domain.LotNumber:
		it.Kind, it.Name = domain.ItemNumber, l.Name
	case domain.LotGift:
		it.Kind = domain.ItemGift
		if l.Gift != nil {
			it.Slug, it.Num, it.GiftID = l.Gift.Slug, l.Gift.Num, l.Gift.GiftID
		}
	}
	return it
}
