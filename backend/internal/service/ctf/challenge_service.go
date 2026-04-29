// challenge_service.go
// 模块05 — CTF竞赛：题目基础管理业务逻辑。
// 负责题目基础 CRUD、权限边界和详情组装，合约、模板、预验证与审核逻辑拆到对应功能域文件。

package ctf

import (
	"context"
	"strings"

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

// challengeService 题目管理服务实现。
type challengeService struct {
	db               *gorm.DB
	challengeRepo    ctfrepo.ChallengeRepository
	templateRepo     ctfrepo.ChallengeTemplateRepository
	contractRepo     ctfrepo.ChallengeContractRepository
	assertionRepo    ctfrepo.ChallengeAssertionRepository
	reviewRepo       ctfrepo.ChallengeReviewRepository
	verificationRepo ctfrepo.ChallengeVerificationRepository
	userQuerier      UserSummaryQuerier
	schoolQuerier    SchoolNameQuerier
	provisioner      NamespaceProvisioner
	envProvisioner   ChallengeEnvironmentProvisioner
	submissionExec   ChallengeSubmissionExecutor
}

// NewChallengeService 创建题目管理服务实例。
func NewChallengeService(
	db *gorm.DB,
	challengeRepo ctfrepo.ChallengeRepository,
	templateRepo ctfrepo.ChallengeTemplateRepository,
	contractRepo ctfrepo.ChallengeContractRepository,
	assertionRepo ctfrepo.ChallengeAssertionRepository,
	reviewRepo ctfrepo.ChallengeReviewRepository,
	verificationRepo ctfrepo.ChallengeVerificationRepository,
	userQuerier UserSummaryQuerier,
	schoolQuerier SchoolNameQuerier,
	provisioner NamespaceProvisioner,
	envProvisioner ChallengeEnvironmentProvisioner,
	submissionExec ChallengeSubmissionExecutor,
) ChallengeService {
	return &challengeService{
		db:               db,
		challengeRepo:    challengeRepo,
		templateRepo:     templateRepo,
		contractRepo:     contractRepo,
		assertionRepo:    assertionRepo,
		reviewRepo:       reviewRepo,
		verificationRepo: verificationRepo,
		userQuerier:      userQuerier,
		schoolQuerier:    schoolQuerier,
		provisioner:      provisioner,
		envProvisioner:   envProvisioner,
		submissionExec:   submissionExec,
	}
}

// Create 创建题目。
func (s *challengeService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateChallengeReq) (*dto.ChallengeStatusResp, error) {
	if err := s.ensureChallengeEditor(sc); err != nil {
		return nil, err
	}
	runtimeMode := resolveChallengeRuntimeMode(req.RuntimeMode)
	if err := validateChallengeUpsertReq(req.Category, req.Difficulty, req.BaseScore, req.FlagType, req.StaticFlag, req.DynamicFlagSecret, req.SourcePath, runtimeMode, req.ChainConfig); err != nil {
		return nil, err
	}

	templateID, err := parseOptionalSnowflake(req.TemplateID)
	if err != nil {
		return nil, err
	}
	challenge := &entity.Challenge{
		ID:          snowflake.Generate(),
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Difficulty:  req.Difficulty,
		BaseScore:   req.BaseScore,
		FlagType:    req.FlagType,
		RuntimeMode: runtimeMode,
		AuthorID:    sc.UserID,
		SchoolID:    sc.SchoolID,
		Status:      enum.ChallengeStatusDraft,
	}
	if challenge.StaticFlag, err = encryptSensitiveText(req.StaticFlag); err != nil {
		return nil, err
	}
	if challenge.DynamicFlagSecret, err = encryptSensitiveText(req.DynamicFlagSecret); err != nil {
		return nil, err
	}
	if req.ChainConfig != nil {
		challenge.ChainConfig = mustJSON(req.ChainConfig)
	}
	if req.SetupTransactions != nil {
		challenge.SetupTransactions = mustJSON(req.SetupTransactions)
	}
	sourcePath := int16(enum.ChallengeSourceCustom)
	challenge.SourcePath = &sourcePath
	if req.SourcePath != nil {
		challenge.SourcePath = req.SourcePath
	}
	if req.SwcID != nil {
		challenge.SwcID = req.SwcID
	}
	if templateID != nil {
		challenge.TemplateID = templateID
	}
	if len(req.TemplateParams) > 0 {
		challenge.TemplateParams = mustJSON(req.TemplateParams)
	}
	if req.EnvironmentConfig != nil {
		challenge.EnvironmentConfig = mustJSON(req.EnvironmentConfig)
	}
	if len(req.AttachmentURLs) > 0 {
		challenge.AttachmentURLs = mustJSON(req.AttachmentURLs)
	}
	if err := s.challengeRepo.Create(ctx, challenge); err != nil {
		return nil, err
	}
	return buildChallengeStatusResp(challenge), nil
}

// List 查询题目列表。
func (s *challengeService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeListReq) (*dto.ChallengeListResp, error) {
	if err := s.ensureChallengeViewer(sc); err != nil {
		return nil, err
	}
	params := &ctfrepo.ChallengeListParams{
		Category:   req.Category,
		Difficulty: req.Difficulty,
		FlagType:   req.FlagType,
		Status:     req.Status,
		IsPublic:   req.IsPublic,
		Keyword:    req.Keyword,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	if req.AuthorID != "" {
		authorID, err := snowflake.ParseString(req.AuthorID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.AuthorID = authorID
	}

	var (
		challenges []*entity.Challenge
		total      int64
		err        error
	)
	switch {
	case sc.IsSuperAdmin():
		challenges, total, err = s.challengeRepo.List(ctx, params)
	case sc.IsSchoolAdmin():
		params.SchoolID = sc.SchoolID
		challenges, total, err = s.challengeRepo.List(ctx, params)
	default:
		params.SchoolID = sc.SchoolID
		challenges, total, err = s.challengeRepo.ListVisibleToTeacher(ctx, sc.UserID, sc.SchoolID, params)
	}
	if err != nil {
		return nil, err
	}

	items := make([]dto.ChallengeListItem, 0, len(challenges))
	for _, challenge := range challenges {
		items = append(items, s.buildChallengeListItem(ctx, challenge))
	}
	return &dto.ChallengeListResp{
		List:       items,
		Pagination: paginationResp(req.Page, req.PageSize, total),
	}, nil
}

// Get 获取题目详情。
func (s *challengeService) Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeDetailResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}

	contracts, _ := s.contractRepo.ListByChallengeID(ctx, challenge.ID)
	assertions, _ := s.assertionRepo.ListByChallengeID(ctx, challenge.ID)
	verification, _ := s.verificationRepo.GetLatestByChallengeID(ctx, challenge.ID)
	return s.buildChallengeDetail(ctx, challenge, contracts, assertions, verification)
}

// Update 编辑题目。
func (s *challengeService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeReq) error {
	challenge, err := getChallenge(ctx, s.challengeRepo, id)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if challenge.Status != enum.ChallengeStatusDraft && challenge.Status != enum.ChallengeStatusRejected {
		return errcode.ErrChallengeStatusInvalid
	}
	effectiveDifficulty := challenge.Difficulty
	if req.Difficulty != nil {
		effectiveDifficulty = *req.Difficulty
	}
	effectiveBaseScore := challenge.BaseScore
	if req.BaseScore != nil {
		effectiveBaseScore = *req.BaseScore
	}
	effectiveCategory := challenge.Category
	if req.Category != nil {
		effectiveCategory = *req.Category
	}
	effectiveFlagType := challenge.FlagType
	if req.FlagType != nil {
		effectiveFlagType = *req.FlagType
	}
	effectiveStaticFlag := challenge.StaticFlag
	if req.StaticFlag != nil {
		effectiveStaticFlag = req.StaticFlag
	}
	effectiveDynamicFlagSecret := challenge.DynamicFlagSecret
	if req.DynamicFlagSecret != nil {
		effectiveDynamicFlagSecret = req.DynamicFlagSecret
	}
	effectiveRuntimeMode := challenge.RuntimeMode
	if req.RuntimeMode != nil {
		effectiveRuntimeMode = *req.RuntimeMode
	}
	effectiveChainConfig := loadChallengeChainConfig(challenge)
	if req.ChainConfig != nil {
		effectiveChainConfig = req.ChainConfig
	}
	if !isBaseScoreWithinDifficultyRange(effectiveDifficulty, effectiveBaseScore) {
		return errcode.ErrChallengeScoreInvalid.WithMessage("基础分值超出难度范围")
	}
	if err := validateChallengeUpsertReq(effectiveCategory, effectiveDifficulty, effectiveBaseScore, effectiveFlagType, effectiveStaticFlag, effectiveDynamicFlagSecret, challenge.SourcePath, effectiveRuntimeMode, effectiveChainConfig); err != nil {
		return err
	}

	fields := map[string]interface{}{}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Category != nil {
		if !enum.IsValidChallengeCategory(*req.Category) {
			return errcode.ErrChallengeCategoryInvalid
		}
		fields["category"] = *req.Category
	}
	if req.Difficulty != nil {
		if !enum.IsValidCtfDifficulty(*req.Difficulty) {
			return errcode.ErrChallengeDifficultyInvalid
		}
		fields["difficulty"] = *req.Difficulty
	}
	if req.BaseScore != nil {
		if *req.BaseScore <= 0 {
			return errcode.ErrChallengeScoreInvalid
		}
		fields["base_score"] = *req.BaseScore
	}
	if req.FlagType != nil {
		if !enum.IsValidFlagType(*req.FlagType) {
			return errcode.ErrChallengeFlagTypeInvalid
		}
		fields["flag_type"] = *req.FlagType
	}
	if req.RuntimeMode != nil {
		fields["runtime_mode"] = *req.RuntimeMode
	}
	if req.StaticFlag != nil {
		encryptedFlag, encryptErr := encryptSensitiveText(req.StaticFlag)
		if encryptErr != nil {
			return encryptErr
		}
		fields["static_flag"] = encryptedFlag
	}
	if req.DynamicFlagSecret != nil {
		encryptedSecret, encryptErr := encryptSensitiveText(req.DynamicFlagSecret)
		if encryptErr != nil {
			return encryptErr
		}
		fields["dynamic_flag_secret"] = encryptedSecret
	}
	if req.ChainConfig != nil {
		fields["chain_config"] = mustJSON(req.ChainConfig)
	}
	if req.SetupTransactions != nil {
		fields["setup_transactions"] = mustJSON(req.SetupTransactions)
	}
	if req.EnvironmentConfig != nil {
		fields["environment_config"] = mustJSON(req.EnvironmentConfig)
	}
	if req.AttachmentURLs != nil {
		fields["attachment_urls"] = mustJSON(req.AttachmentURLs)
	}
	if req.IsPublic != nil {
		fields["is_public"] = *req.IsPublic
	}
	if len(fields) == 0 {
		return nil
	}
	return s.challengeRepo.UpdateFields(ctx, challenge.ID, fields)
}

// Delete 删除题目。
func (s *challengeService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	challenge, err := getChallenge(ctx, s.challengeRepo, id)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if challenge.UsageCount > 0 {
		return errcode.ErrChallengeStatusInvalid.WithMessage("已被竞赛使用的题目不可删除")
	}
	return database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txContractRepo := ctfrepo.NewChallengeContractRepository(tx)
		txAssertionRepo := ctfrepo.NewChallengeAssertionRepository(tx)
		txChallengeRepo := ctfrepo.NewChallengeRepository(tx)
		if err := txContractRepo.DeleteByChallengeID(ctx, challenge.ID); err != nil {
			return err
		}
		if err := txAssertionRepo.DeleteByChallengeID(ctx, challenge.ID); err != nil {
			return err
		}
		return txChallengeRepo.SoftDelete(ctx, challenge.ID)
	})
}

// ensureChallengeViewer 校验当前上下文是否具备题目查看权限。
func (s *challengeService) ensureChallengeViewer(sc *svcctx.ServiceContext) error {
	if sc == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() || sc.IsTeacher() {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureChallengeEditor 校验当前上下文是否具备题目编辑入口权限。
func (s *challengeService) ensureChallengeEditor(sc *svcctx.ServiceContext) error {
	return s.ensureChallengeViewer(sc)
}

// ensureChallengeReadable 校验当前上下文是否可读取指定题目。
// 题目详情、合约、断言、预验证和审核记录都属于敏感信息面，
// 文档约定只允许超级管理员和题目作者访问，不能因为题目已公开就放开源码和断言详情。
func (s *challengeService) ensureChallengeReadable(sc *svcctx.ServiceContext, challenge *entity.Challenge) error {
	if err := s.ensureChallengeViewer(sc); err != nil {
		return err
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if challenge.AuthorID == sc.UserID {
		return nil
	}
	return errcode.ErrChallengeNotFound
}

// ensureChallengeManageable 校验当前上下文是否可管理指定题目。
// 题目维护入口遵循“题目作者/超级管理员”权限边界，不扩展为同校管理员。
func (s *challengeService) ensureChallengeManageable(sc *svcctx.ServiceContext, challenge *entity.Challenge) error {
	if sc == nil || challenge == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if challenge.AuthorID == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// buildChallengeListItem 构建题目列表项。
func (s *challengeService) buildChallengeListItem(ctx context.Context, challenge *entity.Challenge) dto.ChallengeListItem {
	var sourcePathText *string
	if challenge.SourcePath != nil {
		sourcePathText = stringPtr(enum.GetChallengeSourcePathText(*challenge.SourcePath))
	}
	return dto.ChallengeListItem{
		ID:              int64String(challenge.ID),
		Title:           challenge.Title,
		Category:        challenge.Category,
		CategoryText:    enum.GetChallengeCategoryText(challenge.Category),
		Difficulty:      challenge.Difficulty,
		DifficultyText:  enum.GetCtfDifficultyText(challenge.Difficulty),
		BaseScore:       challenge.BaseScore,
		FlagType:        challenge.FlagType,
		FlagTypeText:    enum.GetFlagTypeText(challenge.FlagType),
		RuntimeMode:     int16Ptr(challenge.RuntimeMode),
		RuntimeModeText: stringPtr(enum.GetRuntimeModeText(challenge.RuntimeMode)),
		SourcePath:      challenge.SourcePath,
		SourcePathText:  sourcePathText,
		Status:          challenge.Status,
		StatusText:      enum.GetChallengeStatusText(challenge.Status),
		IsPublic:        challenge.IsPublic,
		UsageCount:      challenge.UsageCount,
		Author:          buildChallengeAuthorBrief(ctx, s.userQuerier, challenge.AuthorID),
		CreatedAt:       timeString(challenge.CreatedAt),
	}
}

// buildChallengeDetail 构建题目详情响应。
func (s *challengeService) buildChallengeDetail(ctx context.Context, challenge *entity.Challenge, contracts []*entity.ChallengeContract, assertions []*entity.ChallengeAssertion, verification *entity.ChallengeVerification) (*dto.ChallengeDetailResp, error) {
	chainConfig := loadChallengeChainConfig(challenge)
	var environmentConfig *dto.ChallengeEnvironmentConfig
	if err := decodeJSON(challenge.EnvironmentConfig, &environmentConfig); err != nil {
		return nil, err
	}
	setupTransactions := []dto.ChallengeSetupTransaction{}
	if err := decodeJSON(challenge.SetupTransactions, &setupTransactions); err != nil {
		return nil, err
	}
	attachmentURLs := []string{}
	if err := decodeJSON(challenge.AttachmentURLs, &attachmentURLs); err != nil {
		return nil, err
	}
	resp := &dto.ChallengeDetailResp{
		ID:                int64String(challenge.ID),
		Title:             challenge.Title,
		Description:       challenge.Description,
		Category:          challenge.Category,
		CategoryText:      enum.GetChallengeCategoryText(challenge.Category),
		Difficulty:        challenge.Difficulty,
		DifficultyText:    enum.GetCtfDifficultyText(challenge.Difficulty),
		BaseScore:         challenge.BaseScore,
		FlagType:          challenge.FlagType,
		FlagTypeText:      enum.GetFlagTypeText(challenge.FlagType),
		RuntimeMode:       int16Ptr(challenge.RuntimeMode),
		RuntimeModeText:   stringPtr(enum.GetRuntimeModeText(challenge.RuntimeMode)),
		ChainConfig:       chainConfig,
		SetupTransactions: setupTransactions,
		SourcePath:        challenge.SourcePath,
		SwcID:             challenge.SwcID,
		EnvironmentConfig: environmentConfig,
		AttachmentURLs:    attachmentURLs,
		Status:            challenge.Status,
		StatusText:        enum.GetChallengeStatusText(challenge.Status),
		IsPublic:          challenge.IsPublic,
		UsageCount:        challenge.UsageCount,
		Author:            buildChallengeAuthorBrief(ctx, s.userQuerier, challenge.AuthorID),
		CreatedAt:         timeString(challenge.CreatedAt),
		UpdatedAt:         timeString(challenge.UpdatedAt),
	}
	if challenge.SourcePath != nil {
		resp.SourcePathText = stringPtr(enum.GetChallengeSourcePathText(*challenge.SourcePath))
	}
	if challenge.TemplateID != nil {
		value := int64String(*challenge.TemplateID)
		resp.TemplateID = &value
	}
	for _, contract := range contracts {
		resp.Contracts = append(resp.Contracts, buildChallengeContractItem(contract, true))
	}
	for _, assertion := range assertions {
		resp.Assertions = append(resp.Assertions, buildChallengeAssertionItem(assertion))
	}
	if verification != nil {
		resp.LatestVerification = &dto.VerificationSummary{
			ID:          int64String(verification.ID),
			Status:      verification.Status,
			StatusText:  enum.GetVerificationStatusText(verification.Status),
			CompletedAt: optionalTimeString(verification.CompletedAt),
		}
	}
	return resp, nil
}

// buildChallengeStatusResp 构建题目状态响应。
func buildChallengeStatusResp(challenge *entity.Challenge) *dto.ChallengeStatusResp {
	resp := &dto.ChallengeStatusResp{
		ID:          int64String(challenge.ID),
		Title:       challenge.Title,
		Category:    challenge.Category,
		Difficulty:  challenge.Difficulty,
		BaseScore:   challenge.BaseScore,
		FlagType:    challenge.FlagType,
		RuntimeMode: int16Ptr(challenge.RuntimeMode),
		Status:      challenge.Status,
		StatusText:  enum.GetChallengeStatusText(challenge.Status),
		CreatedAt:   optionalTimeString(&challenge.CreatedAt),
	}
	if challenge.Category != "" {
		resp.CategoryText = stringPtr(enum.GetChallengeCategoryText(challenge.Category))
	}
	if challenge.Difficulty > 0 {
		resp.DifficultyText = stringPtr(enum.GetCtfDifficultyText(challenge.Difficulty))
	}
	if challenge.FlagType > 0 {
		resp.FlagTypeText = stringPtr(enum.GetFlagTypeText(challenge.FlagType))
	}
	if challenge.RuntimeMode > 0 {
		resp.RuntimeModeText = stringPtr(enum.GetRuntimeModeText(challenge.RuntimeMode))
	}
	if challenge.SourcePath != nil {
		resp.SourcePath = challenge.SourcePath
		resp.SourcePathText = stringPtr(enum.GetChallengeSourcePathText(*challenge.SourcePath))
	}
	if challenge.SwcID != nil {
		resp.SwcID = challenge.SwcID
	}
	if challenge.TemplateID != nil {
		value := int64String(*challenge.TemplateID)
		resp.TemplateID = &value
	}
	return resp
}

// buildChallengeContractItem 构建题目合约列表项。
func buildChallengeContractItem(contract *entity.ChallengeContract, withDetail bool) dto.ChallengeContractItem {
	item := dto.ChallengeContractItem{
		ID:          int64String(contract.ID),
		ChallengeID: stringPtr(int64String(contract.ChallengeID)),
		Name:        contract.Name,
		DeployOrder: contract.DeployOrder,
	}
	if !withDetail {
		return item
	}
	item.SourceCode = &contract.SourceCode
	item.Bytecode = &contract.Bytecode
	_ = decodeJSON(contract.ABI, &item.ABI)
	_ = decodeJSON(contract.ConstructorArgs, &item.ConstructorArgs)
	return item
}

// buildChallengeAssertionItem 构建题目断言列表项。
func buildChallengeAssertionItem(assertion *entity.ChallengeAssertion) dto.ChallengeAssertionItem {
	item := dto.ChallengeAssertionItem{
		ID:            int64String(assertion.ID),
		ChallengeID:   stringPtr(int64String(assertion.ChallengeID)),
		AssertionType: assertion.AssertionType,
		Target:        assertion.Target,
		Operator:      assertion.Operator,
		ExpectedValue: assertion.ExpectedValue,
		Description:   assertion.Description,
		SortOrder:     assertion.SortOrder,
	}
	_ = decodeJSON(assertion.ExtraParams, &item.ExtraParams)
	return item
}

// buildChallengeAuthorBrief 构建题目作者摘要信息。
func buildChallengeAuthorBrief(ctx context.Context, userQuerier UserSummaryQuerier, authorID int64) *dto.ChallengeAuthorBrief {
	if userQuerier == nil {
		return &dto.ChallengeAuthorBrief{
			ID:   int64String(authorID),
			Name: "",
		}
	}
	return &dto.ChallengeAuthorBrief{
		ID:   int64String(authorID),
		Name: userQuerier.GetUserName(ctx, authorID),
	}
}

// validateChallengeUpsertReq 校验题目创建时的核心字段约束。
func validateChallengeUpsertReq(category string, difficulty int16, baseScore int, flagType int16, staticFlag, dynamicFlagSecret *string, sourcePath *int16, runtimeMode int16, chainConfig *dto.ChallengeChainConfig) error {
	if !enum.IsValidChallengeCategory(category) {
		return errcode.ErrChallengeCategoryInvalid
	}
	if !enum.IsValidCtfDifficulty(difficulty) {
		return errcode.ErrChallengeDifficultyInvalid
	}
	if baseScore <= 0 {
		return errcode.ErrChallengeScoreInvalid
	}
	if !isBaseScoreWithinDifficultyRange(difficulty, baseScore) {
		return errcode.ErrChallengeScoreInvalid.WithMessage("基础分值超出难度范围")
	}
	if !enum.IsValidFlagType(flagType) {
		return errcode.ErrChallengeFlagTypeInvalid
	}
	if !enum.IsValidRuntimeMode(runtimeMode) {
		return errcode.ErrChallengeRuntimeModeInvalid
	}
	if flagType == enum.FlagTypeStatic && (staticFlag == nil || strings.TrimSpace(*staticFlag) == "") {
		return errcode.ErrChallengeStaticFlagRequired
	}
	if flagType == enum.FlagTypeDynamic && (dynamicFlagSecret == nil || strings.TrimSpace(*dynamicFlagSecret) == "") {
		return errcode.ErrInvalidParams.WithMessage("动态 Flag 题目必须填写动态密钥")
	}
	if sourcePath != nil && !enum.IsValidChallengeSourcePath(*sourcePath) {
		return errcode.ErrInvalidParams.WithMessage("题目来源路径无效")
	}
	if err := validateChallengeRuntimeConfig(category, flagType, runtimeMode, chainConfig); err != nil {
		return err
	}
	return nil
}

// resolveChallengeRuntimeMode 解析题目运行时模式，未显式传入时回落到独立链模式。
func resolveChallengeRuntimeMode(runtimeMode *int16) int16 {
	if runtimeMode == nil {
		return enum.RuntimeModeIsolated
	}
	return *runtimeMode
}

// validateChallengeRuntimeConfig 校验双运行时模式下的链题配置约束。
func validateChallengeRuntimeConfig(category string, flagType, runtimeMode int16, chainConfig *dto.ChallengeChainConfig) error {
	if runtimeMode == enum.RuntimeModeForked {
		if category != enum.ChallengeCategoryContract || flagType != enum.FlagTypeOnChain {
			return errcode.ErrChallengeRuntimeConfigInvalid.WithMessage("仅链上验证的智能合约题支持 Fork 模式")
		}
		if chainConfig == nil || chainConfig.Fork == nil {
			return errcode.ErrChallengeRuntimeConfigInvalid.WithMessage("Fork 模式缺少链配置")
		}
		if strings.TrimSpace(chainConfig.Fork.RPCURL) == "" || chainConfig.Fork.BlockNumber <= 0 {
			return errcode.ErrChallengeRuntimeConfigInvalid.WithMessage("Fork 模式必须配置 RPC 地址和固定历史区块")
		}
	}
	return nil
}

// loadChallengeChainConfig 读取题目链配置，解码失败时按空配置处理。
func loadChallengeChainConfig(challenge *entity.Challenge) *dto.ChallengeChainConfig {
	if challenge == nil || len(challenge.ChainConfig) == 0 {
		return nil
	}
	var chainConfig dto.ChallengeChainConfig
	if err := decodeJSON(challenge.ChainConfig, &chainConfig); err != nil {
		return nil
	}
	return &chainConfig
}

// isBaseScoreWithinDifficultyRange 校验基础分值是否落在难度对应的文档区间内。
// 范围定义来自模块05功能需求文档中的“五级难度体系”表。
func isBaseScoreWithinDifficultyRange(difficulty int16, baseScore int) bool {
	minScore, maxScore := challengeBaseScoreRange(difficulty)
	if minScore == 0 && maxScore == 0 {
		return false
	}
	return baseScore >= minScore && baseScore <= maxScore
}

// challengeBaseScoreRange 返回指定难度对应的基础分值区间。
func challengeBaseScoreRange(difficulty int16) (int, int) {
	switch difficulty {
	case enum.CtfDifficultyWarmup:
		return 100, 200
	case enum.CtfDifficultyEasy:
		return 200, 400
	case enum.CtfDifficultyMedium:
		return 400, 600
	case enum.CtfDifficultyHard:
		return 600, 800
	case enum.CtfDifficultyInsane:
		return 800, 1000
	default:
		return 0, 0
	}
}
