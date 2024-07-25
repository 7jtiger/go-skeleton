package controller

import (
	"errors"
	"fmt"
	// "go-skeleton/common/util"
)

var defaultGasLimit = uint64(100000000)

// var allowanceToSet = util.ToWei(int64(1000000000)) //10억
// var allowanceMin = util.ToWei(int64(100000000))    //1억
var NotFoundChain = errors.New("not found chain")
var notFound = fmt.Errorf("not found")
var unknownUser = fmt.Errorf("unknown user")

func joinMsg(args ...interface{}) string {
	msg := ""
	for i, a := range args {
		if i == 0 {
			msg = fmt.Sprint(a)
		} else {
			msg = fmt.Sprintf("%v %v", msg, a)
		}
	}
	return msg
}
