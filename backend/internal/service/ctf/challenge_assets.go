package ctf

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// CreateContract 添加题目合约。
func (s *challengeService) CreateContract(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.CreateChallengeContractReq) (*dto.ChallengeContractResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return nil, err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, errcode.ErrChallengeContractNameRequired
	}
	if req.SourceCode == "" {
		return nil, errcode.ErrChallengeContractSourceEmpty
	}
	contract := buildChallengeContract(challengeID, req)
	if err := s.contractRepo.Create(ctx, contract); err != nil {
		return nil, err
	}
	return &dto.ChallengeContractResp{
		ID:          int64String(contract.ID),
		ChallengeID: int64String(contract.ChallengeID),
		Name:        contract.Name,
		DeployOrder: contract.DeployOrder,
	}, nil
}

// UpdateContract 编辑题目合约。
func (s *challengeService) UpdateContract(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeContractReq) error {
	contract, err := s.contractRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrChallengeContractNotFound
		}
		return err
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, contract.ChallengeID)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return err
	}
	fields := map[string]interface{}{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.SourceCode != nil {
		fields["source_code"] = *req.SourceCode
	}
	if req.ABI != nil {
		fields["abi"] = mustJSON(req.ABI)
	}
	if req.Bytecode != nil {
		fields["bytecode"] = *req.Bytecode
	}
	if req.ConstructorArgs != nil {
		fields["constructor_args"] = mustJSON(req.ConstructorArgs)
	}
	if req.DeployOrder != nil {
		fields["deploy_order"] = *req.DeployOrder
	}
	if len(fields) == 0 {
		return nil
	}
	return s.contractRepo.UpdateFields(ctx, id, fields)
}

// DeleteContract 删除题目合约。
func (s *challengeService) DeleteContract(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	contract, err := s.contractRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrChallengeContractNotFound
		}
		return err
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, contract.ChallengeID)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return err
	}
	return s.contractRepo.Delete(ctx, id)
}

// ListContracts 查询题目合约列表。
func (s *challengeService) ListContracts(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeContractListResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}
	contracts, err := s.contractRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ChallengeContractItem, 0, len(contracts))
	for _, contract := range contracts {
		items = append(items, buildChallengeContractItem(contract, true))
	}
	return &dto.ChallengeContractListResp{List: items}, nil
}

// CreateAssertion 添加题目断言。
func (s *challengeService) CreateAssertion(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.CreateChallengeAssertionReq) (*dto.ChallengeAssertionResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return nil, err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return nil, err
	}
	if !enum.IsValidAssertionType(req.AssertionType) {
		return nil, errcode.ErrChallengeAssertionTypeInvalid
	}
	if !enum.IsValidAssertionOperator(req.Operator) {
		return nil, errcode.ErrChallengeAssertionOpInvalid
	}
	assertion := buildChallengeAssertion(challengeID, req)
	if err := s.assertionRepo.Create(ctx, assertion); err != nil {
		return nil, err
	}
	return &dto.ChallengeAssertionResp{
		ID:            int64String(assertion.ID),
		ChallengeID:   int64String(assertion.ChallengeID),
		AssertionType: assertion.AssertionType,
		Target:        assertion.Target,
		Operator:      assertion.Operator,
		ExpectedValue: assertion.ExpectedValue,
		SortOrder:     assertion.SortOrder,
	}, nil
}

// UpdateAssertion 编辑题目断言。
func (s *challengeService) UpdateAssertion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeAssertionReq) error {
	assertion, err := s.assertionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrChallengeAssertionNotFound
		}
		return err
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, assertion.ChallengeID)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return err
	}
	fields := map[string]interface{}{}
	if req.AssertionType != nil {
		if !enum.IsValidAssertionType(*req.AssertionType) {
			return errcode.ErrChallengeAssertionTypeInvalid
		}
		fields["assertion_type"] = *req.AssertionType
	}
	if req.Target != nil {
		fields["target"] = *req.Target
	}
	if req.Operator != nil {
		if !enum.IsValidAssertionOperator(*req.Operator) {
			return errcode.ErrChallengeAssertionOpInvalid
		}
		fields["operator"] = *req.Operator
	}
	if req.ExpectedValue != nil {
		fields["expected_value"] = *req.ExpectedValue
	}
	if req.Description != nil {
		fields["description"] = req.Description
	}
	if req.ExtraParams != nil {
		fields["extra_params"] = mustJSON(req.ExtraParams)
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if len(fields) == 0 {
		return nil
	}
	return s.assertionRepo.UpdateFields(ctx, id, fields)
}

// DeleteAssertion 删除题目断言。
func (s *challengeService) DeleteAssertion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	assertion, err := s.assertionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrChallengeAssertionNotFound
		}
		return err
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, assertion.ChallengeID)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return err
	}
	return s.assertionRepo.Delete(ctx, id)
}

// ListAssertions 查询题目断言列表。
func (s *challengeService) ListAssertions(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeAssertionListResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}
	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ChallengeAssertionItem, 0, len(assertions))
	for _, assertion := range assertions {
		items = append(items, buildChallengeAssertionItem(assertion))
	}
	return &dto.ChallengeAssertionListResp{List: items}, nil
}

// SortAssertions 调整题目断言排序。
func (s *challengeService) SortAssertions(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.SortChallengeAssertionReq) error {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return err
	}
	if err := ensureChallengeDraft(challenge); err != nil {
		return err
	}
	items := make([]ctfrepo.AssertionSortItem, 0, len(req.Items))
	for _, item := range req.Items {
		id, err := snowflake.ParseString(item.ID)
		if err != nil {
			return errcode.ErrInvalidID
		}
		items = append(items, ctfrepo.AssertionSortItem{
			ID:        id,
			SortOrder: item.SortOrder,
		})
	}
	return s.assertionRepo.BatchUpdateSort(ctx, items)
}

// buildChallengeContract 构建题目合约实体。
func buildChallengeContract(challengeID int64, req *dto.CreateChallengeContractReq) *entity.ChallengeContract {
	contract := &entity.ChallengeContract{
		ID:              snowflake.Generate(),
		ChallengeID:     challengeID,
		Name:            req.Name,
		SourceCode:      req.SourceCode,
		ABI:             mustJSON(req.ABI),
		Bytecode:        req.Bytecode,
		ConstructorArgs: mustJSON(req.ConstructorArgs),
		DeployOrder:     req.DeployOrder,
	}
	if contract.DeployOrder == 0 {
		contract.DeployOrder = 1
	}
	return contract
}

// buildChallengeAssertion 构建题目断言实体。
func buildChallengeAssertion(challengeID int64, req *dto.CreateChallengeAssertionReq) *entity.ChallengeAssertion {
	return &entity.ChallengeAssertion{
		ID:            snowflake.Generate(),
		ChallengeID:   challengeID,
		AssertionType: req.AssertionType,
		Target:        req.Target,
		Operator:      req.Operator,
		ExpectedValue: req.ExpectedValue,
		Description:   req.Description,
		ExtraParams:   mustJSON(req.ExtraParams),
		SortOrder:     req.SortOrder,
	}
}

// ensureChallengeDraft 确认题目仍处于草稿态，草稿外不得修改合约和断言配置。
func ensureChallengeDraft(challenge *entity.Challenge) error {
	if challenge == nil || challenge.Status != enum.ChallengeStatusDraft {
		return errcode.ErrChallengeStatusInvalid
	}
	return nil
}
