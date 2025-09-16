package util

import (
	"fmt"
	"math/big"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func StringToUint128(s string) (tbtypes.Uint128, error) {
	bi, ok := new(big.Int).SetString(s, 10) // parse decimal string
	if !ok {
		return tbtypes.Uint128{}, fmt.Errorf("invalid uint128 string: %s", s)
	}
	return tbtypes.BigIntToUint128(*bi), nil
}
