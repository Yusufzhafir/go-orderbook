package order

import (
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func toTigerBeetleUnitsCash(price model.Price, quantity model.Quantity) tbtypes.Uint128 {
	p := uint64(price)
	q := uint64(quantity)
	return tbtypes.ToUint128(p * q)
}

func toTigerBeetleUnitsAsset(quantity model.Quantity) tbtypes.Uint128 {
	return tbtypes.ToUint128(uint64(quantity))
}
