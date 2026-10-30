package anthropic

import "github.com/bassner/claudodex/internal/modelconfig"

const (
	ModelOpus   = modelconfig.DefaultOpusModel
	ModelSonnet = modelconfig.DefaultSonnetModel
	ModelHaiku  = modelconfig.DefaultHaikuModel
	ModelFable  = modelconfig.DefaultFableModel
)

func MapModel(model string) string {
	return modelconfig.Default().MapModel(model)
}
