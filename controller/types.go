package controller

import (
	"fmt"
	"math/rand"
)

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

var maleIcons = []string{
	"https://i.ibb.co/fYrCXYn1/icon-male-01.webp",
	"https://i.ibb.co/YBThXZ4G/icon-male-02.webp",
	"https://i.ibb.co/99MhfMXt/icon-male-03.webp",
	"https://i.ibb.co/C5c51dYg/icon-male-04.webp",
}

var femaleIcons = []string{
	"https://i.ibb.co/WWwmtG46/icon-female-01.webp",
	"https://i.ibb.co/n2YhSJb/icon-female-02.webp",
	"https://i.ibb.co/WWBKbktK/icon-female-03.webp",
	"https://i.ibb.co/Psh1mcXL/icon-female-04.webp",
}

var maleIntroImgs = []string{
	"https://i.ibb.co/YThwgz5L/male-ai-01.webp",
	"https://i.ibb.co/QF37KRST/male-ai-02.webp",
	"https://i.ibb.co/57Sc1pt/male-ai-03.webp",
	"https://i.ibb.co/twcFThVW/male-ai-04.webp",
}

var femaleIntroImgs = []string{
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

func GetRandDefIcon(gender string) string {
	if gender == "1" { // male
		return maleIcons[rand.Intn(len(maleIcons))]
	}
	return femaleIcons[rand.Intn(len(femaleIcons))]
}

func GetRandDefIntroImg(gender string) string {
	if gender == "1" { // male
		return maleIntroImgs[rand.Intn(len(maleIntroImgs))]
	}
	return femaleIntroImgs[rand.Intn(len(femaleIntroImgs))]
}
