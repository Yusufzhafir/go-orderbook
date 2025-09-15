package order

import (
	"fmt"
	"math/big"

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

func stringToUint128(s string) (tbtypes.Uint128, error) {
	bi, ok := new(big.Int).SetString(s, 10) // parse decimal string
	if !ok {
		return tbtypes.Uint128{}, fmt.Errorf("invalid uint128 string: %s", s)
	}
	return tbtypes.BigIntToUint128(*bi), nil
}
