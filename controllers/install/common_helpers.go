package install

import (
	"fmt"
	"github.com/armadaproject/armada-operator/apis/common"
)

func ImageString(image common.Image) string {
	return fmt.Sprintf("%s/%s", image.Repository, image.Tag)
}
