package main

import (
	"fmt"
	"strconv"
)

// MartialBE/one-hub (Source) Channel Types
const (
	SourceChannelTypeUnknown         = 0
	SourceChannelTypeOpenAI          = 1
	SourceChannelTypeAzure           = 3
	SourceChannelTypeCustom          = 8
	SourceChannelTypePaLM            = 11
	SourceChannelTypeAnthropic       = 14
	SourceChannelTypeBaidu           = 15
	SourceChannelTypeZhipu           = 16
	SourceChannelTypeAli             = 17
	SourceChannelTypeXunfei          = 18
	SourceChannelType360             = 19
	SourceChannelTypeOpenRouter      = 20
	SourceChannelTypeTencent         = 23
	SourceChannelTypeAzureSpeech     = 24
	SourceChannelTypeGemini          = 25
	SourceChannelTypeBaichuan        = 26
	SourceChannelTypeMiniMax         = 27
	SourceChannelTypeDeepseek        = 28
	SourceChannelTypeMoonshot        = 29
	SourceChannelTypeMistral         = 30
	SourceChannelTypeGroq            = 31
	SourceChannelTypeBedrock         = 32
	SourceChannelTypeLingyi          = 33
	SourceChannelTypeMidjourney      = 34
	SourceChannelTypeCloudflareAI    = 35
	SourceChannelTypeCohere          = 36
	SourceChannelTypeStabilityAI     = 37
	SourceChannelTypeCoze            = 38
	SourceChannelTypeOllama          = 39
	SourceChannelTypeHunyuan         = 40
	SourceChannelTypeSuno            = 41
	SourceChannelTypeVertexAI        = 42
	SourceChannelTypeLLAMA           = 43
	SourceChannelTypeIdeogram        = 44
	SourceChannelTypeSiliconflow     = 45
	SourceChannelTypeFlux            = 46
	SourceChannelTypeJina            = 47
	SourceChannelTypeRerank          = 48
	SourceChannelTypeGithub          = 49
	SourceChannelTypeRecraft         = 51
	SourceChannelTypeReplicate       = 52
	SourceChannelTypeKling           = 53
	SourceChannelTypeAzureDatabricks = 54
	SourceChannelTypeAzureV1         = 55
	SourceChannelTypeXAI             = 56
)

// songquanpeng/one-api (Target) Channel Types
const (
	TargetChannelTypeUnknown               = 0
	TargetChannelTypeOpenAI                = 1
	TargetChannelTypeAPI2D                 = 2
	TargetChannelTypeAzure                 = 3
	TargetChannelTypeCloseAI               = 4
	TargetChannelTypeOpenAISB              = 5
	TargetChannelTypeOpenAIMax             = 6
	TargetChannelTypeOhMyGPT               = 7
	TargetChannelTypeCustom                = 8
	TargetChannelTypeAils                  = 9
	TargetChannelTypeAIProxy               = 10
	TargetChannelTypePaLM                  = 11
	TargetChannelTypeAPI2GPT               = 12
	TargetChannelTypeAIGC2D                = 13
	TargetChannelTypeAnthropic             = 14
	TargetChannelTypeBaidu                 = 15
	TargetChannelTypeZhipu                 = 16
	TargetChannelTypeAli                   = 17
	TargetChannelTypeXunfei                = 18
	TargetChannelTypeAI360                 = 19
	TargetChannelTypeOpenRouter            = 20
	TargetChannelTypeAIProxyLibrary        = 21
	TargetChannelTypeFastGPT               = 22
	TargetChannelTypeTencent               = 23
	TargetChannelTypeGemini                = 24
	TargetChannelTypeMoonshot              = 25
	TargetChannelTypeBaichuan              = 26
	TargetChannelTypeMinimax               = 27
	TargetChannelTypeMistral               = 28
	TargetChannelTypeGroq                  = 29
	TargetChannelTypeOllama                = 30
	TargetChannelTypeLingYiWanWu           = 31
	TargetChannelTypeStepFun               = 32
	TargetChannelTypeAwsClaude             = 33
	TargetChannelTypeCoze                  = 34
	TargetChannelTypeCohere                = 35
	TargetChannelTypeDeepSeek              = 36
	TargetChannelTypeCloudflare            = 37
	TargetChannelTypeDeepL                 = 38
	TargetChannelTypeTogetherAI            = 39
	TargetChannelTypeDoubao                = 40
	TargetChannelTypeNovita                = 41
	TargetChannelTypeVertextAI             = 42
	TargetChannelTypeProxy                 = 43
	TargetChannelTypeSiliconFlow           = 44
	TargetChannelTypeXAI                   = 45
	TargetChannelTypeReplicate             = 46
	TargetChannelTypeBaiduV2               = 47
	TargetChannelTypeXunfeiV2              = 48
	TargetChannelTypeAliBailian            = 49
	TargetChannelTypeOpenAICompatible      = 50
	TargetChannelTypeGeminiOpenAICompatible = 51
	TargetChannelTypeDummy                 = 52
)

// Mapping from MartialBE/one-hub (Source) to songquanpeng/one-api (Target)
var channelMap = map[int]int{
	SourceChannelTypeOpenAI:       TargetChannelTypeOpenAI,
	SourceChannelTypeAzure:        TargetChannelTypeAzure,
	SourceChannelTypeCustom:       TargetChannelTypeCustom,
	SourceChannelTypePaLM:         TargetChannelTypePaLM,
	SourceChannelTypeAnthropic:    TargetChannelTypeAnthropic,
	SourceChannelTypeBaidu:        TargetChannelTypeBaidu,
	SourceChannelTypeZhipu:        TargetChannelTypeZhipu,
	SourceChannelTypeAli:          TargetChannelTypeAli,
	SourceChannelTypeXunfei:       TargetChannelTypeXunfei,
	SourceChannelType360:          TargetChannelTypeAI360,
	SourceChannelTypeOpenRouter:   TargetChannelTypeOpenRouter,
	SourceChannelTypeTencent:      TargetChannelTypeTencent,
	SourceChannelTypeGemini:       TargetChannelTypeGemini,
	SourceChannelTypeBaichuan:     TargetChannelTypeBaichuan,
	SourceChannelTypeMiniMax:      TargetChannelTypeMinimax,
	SourceChannelTypeDeepseek:     TargetChannelTypeDeepSeek,
	SourceChannelTypeMoonshot:     TargetChannelTypeMoonshot,
	SourceChannelTypeMistral:      TargetChannelTypeMistral,
	SourceChannelTypeGroq:         TargetChannelTypeGroq,
	SourceChannelTypeOllama:       TargetChannelTypeOllama,
	SourceChannelTypeLingyi:       TargetChannelTypeLingYiWanWu,
	SourceChannelTypeCoze:         TargetChannelTypeCoze,
	SourceChannelTypeCohere:       TargetChannelTypeCohere,
	SourceChannelTypeCloudflareAI: TargetChannelTypeCloudflare,
	SourceChannelTypeVertexAI:     TargetChannelTypeVertextAI,
	SourceChannelTypeSiliconflow:  TargetChannelTypeSiliconFlow,
	SourceChannelTypeXAI:          TargetChannelTypeXAI,
	SourceChannelTypeReplicate:    TargetChannelTypeReplicate,
}

// upgradeChannelType converts the channel type from Source (MartialBE) to Target (songquanpeng)
func upgradeChannelType(oldValue interface{}) interface{} {
	var oldVal int
	switch v := oldValue.(type) {
	case int:
		oldVal = v
	case []uint8:
		valStr := string(v)
		valInt, err := strconv.Atoi(valStr)
		if err != nil {
			fmt.Printf("渠道Type旧值: %s (解析错误), 新值: %d (未知类型)\n", valStr, TargetChannelTypeUnknown)
			return TargetChannelTypeUnknown
		}
		oldVal = valInt
	default:
		fmt.Printf("渠道Type旧值: 非整数或字节数组, 新值: %d (未知类型)\n", TargetChannelTypeUnknown)
		return TargetChannelTypeUnknown
	}

	if newVal, found := channelMap[oldVal]; found {
		fmt.Printf("渠道Type旧值: %d, 新值: %d\n", oldVal, newVal)
		return newVal
	}
	fmt.Printf("渠道Type旧值: %d, 新值未找到, 返回默认值: %d (未知类型)\n", oldVal, TargetChannelTypeUnknown)
	return TargetChannelTypeUnknown
}
