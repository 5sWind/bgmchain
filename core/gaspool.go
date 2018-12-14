//
//
//
//
//
//
//
//
//
//
//
//
//
//
//

package core

import "math/big"

//
//
//
type GasPool big.Int

//
func (gp *GasPool) AddGas(amount *big.Int) *GasPool {
	i := (*big.Int)(gp)
	i.Add(i, amount)
	return gp
}

//
//
func (gp *GasPool) SubGas(amount *big.Int) error {
	i := (*big.Int)(gp)
	if i.Cmp(amount) < 0 {
		return ErrGasLimitReached
	}
	i.Sub(i, amount)
	return nil
}

func (gp *GasPool) String() string {
	return (*big.Int)(gp).String()
}
