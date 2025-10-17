package controller

import (
	"errors"
	"fmt"
	"math/rand"
	// "ms-gateway/common/util"
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

func GetRandDefIcon(gender int) string {
	icons := []string{}
	if gender == 1 { // male
		icons = []string{
			"https://i.ibb.co/fYrCXYn1/icon-male-01.webp",
			"https://i.ibb.co/YBThXZ4G/icon-male-02.webp",
			"https://i.ibb.co/99MhfMXt/icon-male-03.webp",
			"https://i.ibb.co/C5c51dYg/icon-male-04.webp",
		}
	} else { // female
		icons = []string{
			"https://i.ibb.co/WWwmtG46/icon-female-01.webp",
			"https://i.ibb.co/n2YhSJb/icon-female-02.webp",
			"https://i.ibb.co/WWBKbktK/icon-female-03.webp",
			"https://i.ibb.co/Psh1mcXL/icon-female-04.webp",
		}
	}

	return icons[rand.Intn(len(icons))]
}

func GetRandDefIntroImg(gender int) string {
	imgs := []string{}
	if gender == 1 { // male
		imgs = []string{
			"https://i.ibb.co/YThwgz5L/male-ai-01.webp",
			"https://i.ibb.co/QF37KRST/male-ai-02.webp",
			"https://i.ibb.co/57Sc1pt/male-ai-03.webp",
			"https://i.ibb.co/twcFThVW/male-ai-04.webp",
		}
	} else { // female
		imgs = []string{
			"https://i.ibb.co/NG6Rjcn/female-ai-01.webp",
			"https://i.ibb.co/Cs4g64YD/female-ai-02.webp",
			"https://i.ibb.co/k2NHsnSB/female-ai-03.webp",
			"https://i.ibb.co/cSH3snRv/female-ai-04.webp",
			"https://i.ibb.co/DHGx3BWx/female-ai-05.webp",
			"https://i.ibb.co/vxPLGg8b/female-ai-06.webp",
			"https://i.ibb.co/dwqPqwT0/female-ai-08.webp",
			"https://i.ibb.co/pNKtFnW/female-ai-07.webp",
			"https://i.ibb.co/WNQ2ntzr/female-ai-09.webp",
			"https://i.ibb.co/GQtMXWW6/female-ai-10.webp",
			"https://i.ibb.co/Q7CgP0J9/female-ai-12.webp",
			"https://i.ibb.co/fY2RfJzh/female-ai-11.webp",
		}
	}

	return imgs[rand.Intn(len(imgs))]
}
