package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-oa/app/core/service/internal/data/ent"
	"go-wind-oa/app/core/service/internal/data/ent/sealapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type SealApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewSealApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *SealApplicationRepo {
	return &SealApplicationRepo{
		log:       ctx.NewLoggerHelper("seal-application/repo/core-service"),
		entClient: entClient,
	}
}

func sealStatusToProto(s sealapplication.SealStatus) oaV1.SealApplication_SealStatus {
	switch s {
	case sealapplication.SealStatusApproved:
		return oaV1.SealApplication_APPROVED
	case sealapplication.SealStatusRejected:
		return oaV1.SealApplication_REJECTED
	case sealapplication.SealStatusWithdrawn:
		return oaV1.SealApplication_WITHDRAWN
	default:
		return oaV1.SealApplication_PENDING
	}
}

func sealStatusToEntity(s oaV1.SealApplication_SealStatus) sealapplication.SealStatus {
	switch s {
	case oaV1.SealApplication_APPROVED:
		return sealapplication.SealStatusApproved
	case oaV1.SealApplication_REJECTED:
		return sealapplication.SealStatusRejected
	case oaV1.SealApplication_WITHDRAWN:
		return sealapplication.SealStatusWithdrawn
	default:
		return sealapplication.SealStatusPending
	}
}

func sealTypeToProto(s sealapplication.SealType) oaV1.SealApplication_SealType {
	switch s {
	case sealapplication.SealTypeContractSeal:
		return oaV1.SealApplication_CONTRACT_SEAL
	case sealapplication.SealTypeFinanceSeal:
		return oaV1.SealApplication_FINANCE_SEAL
	case sealapplication.SealTypeLegalSeal:
		return oaV1.SealApplication_LEGAL_SEAL
	default:
		return oaV1.SealApplication_OFFICIAL_SEAL
	}
}

func sealTypeToEntity(s oaV1.SealApplication_SealType) sealapplication.SealType {
	switch s {
	case oaV1.SealApplication_CONTRACT_SEAL:
		return sealapplication.SealTypeContractSeal
	case oaV1.SealApplication_FINANCE_SEAL:
		return sealapplication.SealTypeFinanceSeal
	case oaV1.SealApplication_LEGAL_SEAL:
		return sealapplication.SealTypeLegalSeal
	default:
		return sealapplication.SealTypeOfficialSeal
	}
}

func sealApplicationToDTO(e *ent.SealApplication) *oaV1.SealApplication {
	if e == nil {
		return nil
	}
	status := oaV1.SealApplication_PENDING
	if e.SealStatus != nil {
		status = sealStatusToProto(*e.SealStatus)
	}
	sealType := oaV1.SealApplication_OFFICIAL_SEAL
	if e.SealType != nil {
		sealType = sealTypeToProto(*e.SealType)
	}
	dto := &oaV1.SealApplication{
		Id:          trans.Ptr(e.ID),
		Purpose:     trans.Ptr(e.Purpose),
		SealType:    sealType.Enum(),
		FileCount:   trans.Ptr(e.FileCount),
		Recipient:   trans.Ptr(e.Recipient),
		SealStatus:  status.Enum(),
		TenantId:    e.TenantID,
	}
	if e.InstanceID != nil {
		dto.InstanceId = trans.Ptr(*e.InstanceID)
	}
	if e.CreatedBy != nil {
		dto.CreatedBy = e.CreatedBy
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

func (r *SealApplicationRepo) Create(
	ctx context.Context,
	tid, uid uint32,
	purpose string,
	sealType oaV1.SealApplication_SealType,
	fileCount int32,
	recipient string,
) (uint32, error) {
	entity, err := r.entClient.Client().SealApplication.Create().
		SetPurpose(purpose).
		SetSealType(sealTypeToEntity(sealType)).
		SetFileCount(fileCount).
		SetRecipient(recipient).
		SetSealStatus(sealapplication.SealStatusPending).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert seal application failed: %s", err.Error())
		return 0, oaV1.ErrorInternalServerError("insert seal application failed")
	}
	return entity.ID, nil
}

func (r *SealApplicationRepo) Get(ctx context.Context, tid, id uint32) (*oaV1.SealApplication, error) {
	entity, err := r.entClient.Client().SealApplication.Query().
		Where(
			sealapplication.IDEQ(id),
			sealapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("seal application not found")
		}
		r.log.Errorf("query seal application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query seal application failed")
	}
	return sealApplicationToDTO(entity), nil
}

func (r *SealApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.SealApplication_SealStatus) error {
	if _, err := r.entClient.Client().SealApplication.Update().
		Where(
			sealapplication.IDEQ(id),
			sealapplication.TenantIDEQ(tid),
		).
		SetSealStatus(sealStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update seal application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update seal application status failed")
	}
	return nil
}

func (r *SealApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().SealApplication.Update().
		Where(
			sealapplication.IDEQ(id),
			sealapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link seal application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link seal application instance failed")
	}
	return nil
}

func (r *SealApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.SealApplication_SealStatus,
	page, pageSize int32,
) ([]*oaV1.SealApplication, int, error) {
	query := r.entClient.Client().SealApplication.Query().
		Where(sealapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(sealapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(sealapplication.SealStatusEQ(sealStatusToEntity(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count seal applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count seal applications failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	entities, err := query.Order(ent.Desc(sealapplication.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list seal applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list seal applications failed")
	}

	items := make([]*oaV1.SealApplication, 0, len(entities))
	for _, e := range entities {
		items = append(items, sealApplicationToDTO(e))
	}
	return items, total, nil
}

func (r *SealApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.SealApplication, error) {
	entity, err := r.entClient.Client().SealApplication.Query().
		Where(
			sealapplication.IDEQ(id),
			sealapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query seal application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query seal application failed")
	}
	return entity, nil
}
