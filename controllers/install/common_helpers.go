package install

import (
	"fmt"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func ImageString(image installv1alpha1.Image) string {
	return fmt.Sprintf("%s/%s", image.Repository, image.Tag)
}
