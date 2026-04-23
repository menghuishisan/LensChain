package ctf

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// ListSWCRegistry 查询 SWC Registry 列表。
func (s *challengeService) ListSWCRegistry(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SWCRegistryListReq) ([]dto.SWCRegistryItem, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	items := buildSWCRegistryItems()
	if req.Keyword == "" {
		return items, nil
	}
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	result := make([]dto.SWCRegistryItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.SwcID), keyword) || strings.Contains(strings.ToLower(item.Title), keyword) {
			result = append(result, item)
		}
	}
	return result, nil
}

// ImportSWC 从 SWC 样例导入题目。
func (s *challengeService) ImportSWC(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportSWCChallengeReq) (*dto.ChallengeStatusResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	if !isBaseScoreWithinDifficultyRange(req.Difficulty, req.BaseScore) {
		return nil, errcode.ErrChallengeScoreInvalid.WithMessage("基础分值超出难度范围")
	}
	entry, ok := findSWCRegistryItem(req.SwcID)
	if !ok {
		return nil, errcode.ErrSWCEntryInvalid
	}
	if !entry.HasExample {
		return nil, errcode.ErrSWCExampleUnavailable
	}
	challenge, contracts, assertions := buildChallengeFromSWC(req, sc)
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txChallengeRepo := ctfrepo.NewChallengeRepository(tx)
		txContractRepo := ctfrepo.NewChallengeContractRepository(tx)
		txAssertionRepo := ctfrepo.NewChallengeAssertionRepository(tx)
		if err := txChallengeRepo.Create(ctx, challenge); err != nil {
			return err
		}
		if err := txContractRepo.BatchCreate(ctx, contracts); err != nil {
			return err
		}
		return txAssertionRepo.BatchCreate(ctx, assertions)
	})
	if err != nil {
		return nil, err
	}
	resp := buildChallengeStatusResp(challenge)
	resp.ContractsGenerated = intPtr(len(contracts))
	resp.AssertionsGenerated = intPtr(len(assertions))
	return resp, nil
}

// ListTemplates 查询参数化模板列表。
func (s *challengeService) ListTemplates(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeTemplateListReq) (*dto.ChallengeTemplateListResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	templates, _, err := s.templateRepo.List(ctx, &ctfrepo.ChallengeTemplateListParams{
		VulnerabilityType: req.VulnerabilityType,
		Keyword:           req.Keyword,
		Page:              1,
		PageSize:          1000,
	})
	if err != nil {
		return nil, err
	}
	items := make([]dto.ChallengeTemplateListItem, 0, len(templates))
	for _, template := range templates {
		item, err := buildChallengeTemplateListItem(template)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &dto.ChallengeTemplateListResp{List: items}, nil
}

// GetTemplate 获取模板详情。
func (s *challengeService) GetTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeTemplateDetailResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrChallengeTemplateNotFound
		}
		return nil, err
	}
	return buildChallengeTemplateDetail(template)
}

// GenerateFromTemplate 从模板生成题目。
func (s *challengeService) GenerateFromTemplate(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GenerateChallengeFromTemplateReq) (*dto.ChallengeStatusResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	if !isBaseScoreWithinDifficultyRange(req.Difficulty, req.BaseScore) {
		return nil, errcode.ErrChallengeScoreInvalid.WithMessage("基础分值超出难度范围")
	}
	templateID, err := snowflake.ParseString(req.TemplateID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrChallengeTemplateNotFound
		}
		return nil, err
	}
	if err := validateTemplateDifficulty(template, req.Difficulty); err != nil {
		return nil, err
	}
	challenge, contracts, assertions, err := buildChallengeFromTemplate(template, req, sc)
	if err != nil {
		return nil, err
	}
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txChallengeRepo := ctfrepo.NewChallengeRepository(tx)
		txContractRepo := ctfrepo.NewChallengeContractRepository(tx)
		txAssertionRepo := ctfrepo.NewChallengeAssertionRepository(tx)
		txTemplateRepo := ctfrepo.NewChallengeTemplateRepository(tx)
		if err := txChallengeRepo.Create(ctx, challenge); err != nil {
			return err
		}
		if err := txContractRepo.BatchCreate(ctx, contracts); err != nil {
			return err
		}
		if err := txAssertionRepo.BatchCreate(ctx, assertions); err != nil {
			return err
		}
		return txTemplateRepo.IncrementUsage(ctx, template.ID, 1)
	})
	if err != nil {
		return nil, err
	}
	resp := buildChallengeStatusResp(challenge)
	resp.ContractsGenerated = intPtr(len(contracts))
	resp.AssertionsGenerated = intPtr(len(assertions))
	return resp, nil
}

// ImportExternalVulnerability 从外部真实漏洞源导入题目草稿。
// A级源可生成链上验证题，B级源生成待补全链上草稿，C级源按文档降级为 blockchain/misc 静态素材。
func (s *challengeService) ImportExternalVulnerability(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportExternalVulnerabilityReq) (*dto.ChallengeStatusResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	if !isBaseScoreWithinDifficultyRange(req.Difficulty, req.BaseScore) {
		return nil, errcode.ErrChallengeScoreInvalid.WithMessage("基础分值超出难度范围")
	}
	challenge, contracts, assertions := buildChallengeFromExternalSource(req, sc)
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txChallengeRepo := ctfrepo.NewChallengeRepository(tx)
		txContractRepo := ctfrepo.NewChallengeContractRepository(tx)
		txAssertionRepo := ctfrepo.NewChallengeAssertionRepository(tx)
		if err := txChallengeRepo.Create(ctx, challenge); err != nil {
			return err
		}
		if len(contracts) > 0 {
			if err := txContractRepo.BatchCreate(ctx, contracts); err != nil {
				return err
			}
		}
		if len(assertions) > 0 {
			return txAssertionRepo.BatchCreate(ctx, assertions)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp := buildChallengeStatusResp(challenge)
	resp.ContractsGenerated = intPtr(len(contracts))
	resp.AssertionsGenerated = intPtr(len(assertions))
	return resp, nil
}

// buildSWCRegistryItems 构建内置 SWC 样例列表。
func buildSWCRegistryItems() []dto.SWCRegistryItem {
	return []dto.SWCRegistryItem{
		{SwcID: "SWC-107", Title: "Reentrancy", Description: "外部调用重入导致资金被重复提取。", Severity: "High", HasExample: true, SuggestedDifficulty: enum.CtfDifficultyEasy},
		{SwcID: "SWC-101", Title: "Integer Overflow and Underflow", Description: "整数溢出/下溢引发余额或权限逻辑错误。", Severity: "High", HasExample: true, SuggestedDifficulty: enum.CtfDifficultyEasy},
		{SwcID: "SWC-105", Title: "Unprotected Ether Withdrawal", Description: "缺少访问控制导致任意地址提走资金。", Severity: "Medium", HasExample: true, SuggestedDifficulty: enum.CtfDifficultyMedium},
	}
}

// findSWCRegistryItem 按 SWC 编号查找内置样例。
func findSWCRegistryItem(swcID string) (dto.SWCRegistryItem, bool) {
	for _, item := range buildSWCRegistryItems() {
		if item.SwcID == swcID {
			return item, true
		}
	}
	return dto.SWCRegistryItem{}, false
}

// buildChallengeFromSWC 根据 SWC 条目生成题目、合约和断言。
func buildChallengeFromSWC(req *dto.ImportSWCChallengeReq, sc *svcctx.ServiceContext) (*entity.Challenge, []*entity.ChallengeContract, []*entity.ChallengeAssertion) {
	challengeID := snowflake.Generate()
	sourcePath := int16(enum.ChallengeSourceSWC)
	challenge := &entity.Challenge{
		ID:          challengeID,
		Title:       req.Title,
		Description: "由 SWC Registry 示例自动生成，教师可继续补充说明和素材。",
		Category:    enum.ChallengeCategoryContract,
		Difficulty:  req.Difficulty,
		BaseScore:   req.BaseScore,
		FlagType:    enum.FlagTypeOnChain,
		RuntimeMode: enum.RuntimeModeIsolated,
		SourcePath:  &sourcePath,
		SwcID:       stringPtr(req.SwcID),
		AuthorID:    sc.UserID,
		SchoolID:    sc.SchoolID,
		Status:      enum.ChallengeStatusDraft,
	}
	contracts := []*entity.ChallengeContract{
		{
			ID:          snowflake.Generate(),
			ChallengeID: challengeID,
			Name:        "Challenge",
			SourceCode:  buildSWCContractSource(req.SwcID),
			ABI:         datatypes.JSON([]byte("[]")),
			Bytecode:    "0x00",
			DeployOrder: 1,
		},
	}
	assertions := []*entity.ChallengeAssertion{
		{
			ID:            snowflake.Generate(),
			ChallengeID:   challengeID,
			AssertionType: enum.AssertionTypeStorage,
			Target:        "solved",
			Operator:      "eq",
			ExpectedValue: "true",
			Description:   stringPtr("攻击成功后目标状态应被修改为 solved=true"),
			SortOrder:     1,
		},
	}
	return challenge, contracts, assertions
}

// buildChallengeFromExternalSource 根据外部真实漏洞源等级生成题目草稿。
func buildChallengeFromExternalSource(req *dto.ImportExternalVulnerabilityReq, sc *svcctx.ServiceContext) (*entity.Challenge, []*entity.ChallengeContract, []*entity.ChallengeAssertion) {
	challengeID := snowflake.Generate()
	sourcePath := int16(enum.ChallengeSourceCustom)
	category := req.Category
	flagType := int16(enum.FlagTypeOnChain)
	runtimeMode := int16(enum.RuntimeModeIsolated)
	if req.SourceGrade == "C" {
		if category == enum.ChallengeCategoryContract {
			category = enum.ChallengeCategoryBlockchain
		}
		flagType = int16(enum.FlagTypeDynamic)
	}
	if req.SourceGrade == "A" && req.ChainConfig != nil && req.ChainConfig.Fork != nil && req.ChainConfig.Fork.BlockNumber > 0 {
		runtimeMode = int16(enum.RuntimeModeForked)
	}
	metadata := map[string]interface{}{
		"external_source": map[string]interface{}{
			"source_grade":          req.SourceGrade,
			"vulnerability_name":    req.VulnerabilityName,
			"source_url":            req.SourceURL,
			"confidence_score":      req.ConfidenceScore,
			"reproducibility_score": req.ReproducibilityScore,
			"reference_event":       req.ReferenceEvent,
		},
	}
	challenge := &entity.Challenge{
		ID:                challengeID,
		Title:             req.Title,
		Description:       "由外部真实漏洞源导入生成的题目草稿，等级：" + req.SourceGrade + "，来源：" + req.SourceURL,
		Category:          category,
		Difficulty:        req.Difficulty,
		BaseScore:         req.BaseScore,
		FlagType:          flagType,
		RuntimeMode:       runtimeMode,
		ChainConfig:       mustJSON(req.ChainConfig),
		SetupTransactions: mustJSON(req.SetupTransactions),
		SourcePath:        &sourcePath,
		TemplateParams:    mustJSON(metadata),
		AuthorID:          sc.UserID,
		SchoolID:          sc.SchoolID,
		Status:            enum.ChallengeStatusDraft,
	}
	if req.SourceGrade == "C" {
		return challenge, nil, nil
	}
	sourceCode := derefString(req.SourceCode)
	if strings.TrimSpace(sourceCode) == "" {
		return challenge, nil, nil
	}
	contracts := []*entity.ChallengeContract{{
		ID:          snowflake.Generate(),
		ChallengeID: challengeID,
		Name:        detectTemplateContractName(sourceCode),
		SourceCode:  sourceCode,
		ABI:         datatypes.JSON([]byte("[]")),
		Bytecode:    "0x00",
		DeployOrder: 1,
	}}
	assertions := []*entity.ChallengeAssertion(nil)
	if req.SourceGrade == "A" {
		assertions = []*entity.ChallengeAssertion{{
			ID:            snowflake.Generate(),
			ChallengeID:   challengeID,
			AssertionType: enum.AssertionTypeCustomScript,
			Target:        "external_poc",
			Operator:      "eq",
			ExpectedValue: "true",
			Description:   stringPtr("A级外部源默认断言，教师需在预验证前补充精确断言。"),
			ExtraParams:   mustJSON(map[string]interface{}{"poc_content": derefString(req.PocContent)}),
			SortOrder:     1,
		}}
	}
	return challenge, contracts, assertions
}

// buildSWCContractSource 根据 SWC 编号生成最小示例源码。
func buildSWCContractSource(swcID string) string {
	switch swcID {
	case "SWC-101":
		return "pragma solidity ^0.8.20; contract Challenge { uint8 public counter = 250; function increase(uint8 n) external { unchecked { counter += n; } } }"
	case "SWC-105":
		return "pragma solidity ^0.8.20; contract Challenge { address public owner; constructor() payable { owner = msg.sender; } function withdraw() external { payable(msg.sender).transfer(address(this).balance); } }"
	default:
		return "pragma solidity ^0.8.20; contract Challenge { bool public solved; function attack() external { solved = true; } }"
	}
}

// validateTemplateDifficulty 校验题目难度是否落在模板适用范围。
func validateTemplateDifficulty(template *entity.ChallengeTemplate, difficulty int16) error {
	var difficultyRange dto.DifficultyRange
	if err := decodeJSON(template.DifficultyRange, &difficultyRange); err != nil {
		return err
	}
	if difficulty < difficultyRange.Min || difficulty > difficultyRange.Max {
		return errcode.ErrChallengeTemplateDifficultyErr
	}
	return nil
}

// buildChallengeFromTemplate 根据模板生成题目、合约和断言。
func buildChallengeFromTemplate(template *entity.ChallengeTemplate, req *dto.GenerateChallengeFromTemplateReq, sc *svcctx.ServiceContext) (*entity.Challenge, []*entity.ChallengeContract, []*entity.ChallengeAssertion, error) {
	var baseAssertions []dto.ChallengeAssertionItem
	if err := decodeJSON(template.BaseAssertions, &baseAssertions); err != nil {
		return nil, nil, nil, err
	}
	if err := validateTemplateParams(template, req.TemplateParams); err != nil {
		return nil, nil, nil, err
	}
	challengeID := snowflake.Generate()
	sourcePath := int16(enum.ChallengeSourceTemplate)
	challenge := &entity.Challenge{
		ID:             challengeID,
		Title:          req.Title,
		Description:    derefString(template.Description),
		Category:       enum.ChallengeCategoryContract,
		Difficulty:     req.Difficulty,
		BaseScore:      req.BaseScore,
		FlagType:       enum.FlagTypeOnChain,
		RuntimeMode:    enum.RuntimeModeIsolated,
		SourcePath:     &sourcePath,
		TemplateID:     &template.ID,
		TemplateParams: mustJSON(req.TemplateParams),
		AuthorID:       sc.UserID,
		SchoolID:       sc.SchoolID,
		Status:         enum.ChallengeStatusDraft,
	}
	sourceCode := substituteTemplateString(template.BaseSourceCode, req.TemplateParams)
	contracts := []*entity.ChallengeContract{
		{
			ID:          snowflake.Generate(),
			ChallengeID: challengeID,
			Name:        detectTemplateContractName(sourceCode),
			SourceCode:  sourceCode,
			ABI:         datatypes.JSON([]byte("[]")),
			Bytecode:    "0x00",
			DeployOrder: 1,
		},
	}
	if len(template.BaseSetupTransactions) > 0 {
		var baseSetupTransactions []dto.ChallengeSetupTransaction
		if err := decodeJSON(template.BaseSetupTransactions, &baseSetupTransactions); err != nil {
			return nil, nil, nil, err
		}
		challenge.SetupTransactions = mustJSON(substituteTemplateSetupTransactions(baseSetupTransactions, req.TemplateParams))
	}
	assertions := make([]*entity.ChallengeAssertion, 0, len(baseAssertions))
	for idx, item := range baseAssertions {
		extraParams := substituteTemplateMap(item.ExtraParams, req.TemplateParams)
		assertions = append(assertions, &entity.ChallengeAssertion{
			ID:            snowflake.Generate(),
			ChallengeID:   challengeID,
			AssertionType: item.AssertionType,
			Target:        substituteTemplateString(item.Target, req.TemplateParams),
			Operator:      item.Operator,
			ExpectedValue: substituteTemplateString(item.ExpectedValue, req.TemplateParams),
			Description:   substituteOptionalTemplateString(item.Description, req.TemplateParams),
			ExtraParams:   mustJSON(extraParams),
			SortOrder:     idx + 1,
		})
	}
	return challenge, contracts, assertions, nil
}

// substituteTemplateSetupTransactions 对模板中的初始化交易执行占位符替换。
func substituteTemplateSetupTransactions(items []dto.ChallengeSetupTransaction, params map[string]interface{}) []dto.ChallengeSetupTransaction {
	result := make([]dto.ChallengeSetupTransaction, 0, len(items))
	for _, item := range items {
		result = append(result, dto.ChallengeSetupTransaction{
			From:     substituteTemplateString(item.From, params),
			To:       substituteTemplateString(item.To, params),
			Function: substituteTemplateString(item.Function, params),
			Args:     normalizeTemplateArgs(substituteTemplateValue(item.Args, params)),
			Value:    substituteTemplateString(item.Value, params),
		})
	}
	return result
}

// normalizeTemplateArgs 把模板替换后的任意值收敛为参数数组。
func normalizeTemplateArgs(value interface{}) []interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return []interface{}{}
	}
	return items
}

// validateTemplateParams 校验模板参数是否完整。
func validateTemplateParams(template *entity.ChallengeTemplate, params map[string]interface{}) error {
	var parameters dto.ChallengeTemplateParameters
	if err := decodeJSON(template.Parameters, &parameters); err != nil {
		return err
	}
	for _, item := range parameters.Params {
		value, ok := params[item.Key]
		if !ok || value == nil {
			return errcode.ErrChallengeTemplateParamMissing
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			return errcode.ErrChallengeTemplateParamMissing
		}
	}
	return nil
}

var templateContractNamePattern = regexp.MustCompile(`\bcontract\s+([A-Za-z_][A-Za-z0-9_]*)`)

// detectTemplateContractName 从模板源码中推断主合约名。
func detectTemplateContractName(source string) string {
	match := templateContractNamePattern.FindStringSubmatch(source)
	if len(match) >= 2 {
		return match[1]
	}
	return "Challenge"
}

// substituteOptionalTemplateString 按模板参数替换可选文本。
func substituteOptionalTemplateString(value *string, params map[string]interface{}) *string {
	if value == nil {
		return nil
	}
	replaced := substituteTemplateString(*value, params)
	return &replaced
}

// substituteTemplateMap 对模板中的 map 值递归执行字符串替换。
func substituteTemplateMap(value map[string]interface{}, params map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = substituteTemplateValue(item, params)
	}
	return result
}

// substituteTemplateValue 对模板中的任意值递归执行字符串替换。
func substituteTemplateValue(value interface{}, params map[string]interface{}) interface{} {
	switch typed := value.(type) {
	case string:
		return substituteTemplateString(typed, params)
	case []interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, substituteTemplateValue(item, params))
		}
		return items
	case map[string]interface{}:
		return substituteTemplateMap(typed, params)
	default:
		return value
	}
}

// substituteTemplateString 按 {{key}} 占位符替换模板字符串。
func substituteTemplateString(raw string, params map[string]interface{}) string {
	result := raw
	for key, value := range params {
		placeholder := "{{" + key + "}}"
		placeholderWithSpace := "{{ " + key + " }}"
		rendered := fmt.Sprint(value)
		result = strings.ReplaceAll(result, placeholder, rendered)
		result = strings.ReplaceAll(result, placeholderWithSpace, rendered)
	}
	return result
}

// buildChallengeTemplateListItem 构建模板列表项。
func buildChallengeTemplateListItem(template *entity.ChallengeTemplate) (dto.ChallengeTemplateListItem, error) {
	var difficultyRange dto.DifficultyRange
	var variants []dto.ChallengeTemplateVariant
	if err := decodeJSON(template.DifficultyRange, &difficultyRange); err != nil {
		return dto.ChallengeTemplateListItem{}, err
	}
	if err := decodeJSON(template.Variants, &variants); err != nil {
		return dto.ChallengeTemplateListItem{}, err
	}
	return dto.ChallengeTemplateListItem{
		ID:                int64String(template.ID),
		Name:              template.Name,
		Code:              template.Code,
		VulnerabilityType: template.VulnerabilityType,
		Description:       derefString(template.Description),
		DifficultyRange:   difficultyRange,
		VariantCount:      len(variants),
		UsageCount:        template.UsageCount,
	}, nil
}

// buildChallengeTemplateDetail 构建模板详情响应。
func buildChallengeTemplateDetail(template *entity.ChallengeTemplate) (*dto.ChallengeTemplateDetailResp, error) {
	var difficultyRange dto.DifficultyRange
	var parameters dto.ChallengeTemplateParameters
	var variants []dto.ChallengeTemplateVariant
	var referenceEvents []dto.ChallengeReferenceEvent
	if err := decodeJSON(template.DifficultyRange, &difficultyRange); err != nil {
		return nil, err
	}
	if err := decodeJSON(template.Parameters, &parameters); err != nil {
		return nil, err
	}
	if err := decodeJSON(template.Variants, &variants); err != nil {
		return nil, err
	}
	if err := decodeJSON(template.ReferenceEvents, &referenceEvents); err != nil {
		return nil, err
	}
	return &dto.ChallengeTemplateDetailResp{
		ID:                int64String(template.ID),
		Name:              template.Name,
		Code:              template.Code,
		Description:       derefString(template.Description),
		VulnerabilityType: template.VulnerabilityType,
		DifficultyRange:   difficultyRange,
		Parameters:        parameters,
		Variants:          variants,
		ReferenceEvents:   referenceEvents,
		UsageCount:        template.UsageCount,
	}, nil
}
