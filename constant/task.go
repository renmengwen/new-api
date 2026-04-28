package constant

type TaskPlatform string

const (
	TaskPlatformSuno          TaskPlatform = "suno"
	TaskPlatformMidjourney                 = "mj"
	TaskPlatformGPTProtoImage              = "gptproto_image"
)

const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"

	TaskTypeImageGeneration = "image_generation"
	TaskTypeVideoGeneration = "video_generation"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
)

var SunoModel2Action = map[string]string{
	"suno_music":  SunoActionMusic,
	"suno_lyrics": SunoActionLyrics,
}
