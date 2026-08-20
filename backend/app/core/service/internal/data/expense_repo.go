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
	"go-wind-oa/app/core/service/internal/data/ent/expenseapplication"

	oaV1 "go-wind-oa/api/gen/go/oa/service/v1"
)

type ExpenseApplicationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewExpenseApplicationRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *ExpenseApplicationRepo {
	return &ExpenseApplicationRepo{
		log:       ctx.NewLoggerHelper("expense-application/repo/core-service"),
		entClient: entClient,
	}
}

func expenseStatusToProto(s expenseapplication.ExpenseStatus) oaV1.ExpenseApplication_ExpenseStatus {
	switch s {
	case expenseapplication.ExpenseStatusApproved:
		return oaV1.ExpenseApplication_APPROVED
	case expenseapplication.ExpenseStatusRejected:
		return oaV1.ExpenseApplication_REJECTED
	case expenseapplication.ExpenseStatusWithdrawn:
		return oaV1.ExpenseApplication_WITHDRAWN
	default:
		return oaV1.ExpenseApplication_PENDING
	}
}

func expenseStatusToEntity(s oaV1.ExpenseApplication_ExpenseStatus) expenseapplication.ExpenseStatus {
	switch s {
	case oaV1.ExpenseApplication_APPROVED:
		return expenseapplication.ExpenseStatusApproved
	case oaV1.ExpenseApplication_REJECTED:
		return expenseapplication.ExpenseStatusRejected
	case oaV1.ExpenseApplication_WITHDRAWN:
		return expenseapplication.ExpenseStatusWithdrawn
	default:
		return expenseapplication.ExpenseStatusPending
	}
}

func expenseItemToDTO(e *ent.ExpenseItem) *oaV1.ExpenseItem {
	if e == nil {
		return nil
	}
	dto := &oaV1.ExpenseItem{
		Id:          trans.Ptr(e.ID),
		Category:    trans.Ptr(e.Category),
		Amount:      trans.Ptr(e.Amount),
		Description: trans.Ptr(e.Description),
	}
	if e.ExpenseDate != nil {
		dto.ExpenseDate = timeutil.TimeToTimestamppb(e.ExpenseDate)
	}
	if e.InvoiceFileID != nil {
		dto.InvoiceFileId = trans.Ptr(*e.InvoiceFileID)
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	return dto
}

func expenseApplicationToDTO(e *ent.ExpenseApplication) *oaV1.ExpenseApplication {
	if e == nil {
		return nil
	}
	status := oaV1.ExpenseApplication_PENDING
	if e.ExpenseStatus != nil {
		status = expenseStatusToProto(*e.ExpenseStatus)
	}
	dto := &oaV1.ExpenseApplication{
		Id:            trans.Ptr(e.ID),
		Title:         trans.Ptr(e.Title),
		TotalAmount:   trans.Ptr(e.TotalAmount),
		ExpenseStatus: status.Enum(),
		TenantId:      e.TenantID,
	}
	if e.InstanceID != nil {
		dto.InstanceId = trans.Ptr(*e.InstanceID)
	}
	if e.CreatedBy != nil {
		dto.CreatedBy = e.CreatedBy
	}
	dto.CreatedAt = timeutil.TimeToTimestamppb(e.CreatedAt)
	for _, item := range e.Edges.Items {
		dto.Items = append(dto.Items, expenseItemToDTO(item))
	}
	return dto
}

// CreateWithItems 创建申请单后逐条存明细（无事务，与既有无事务多步写入的风险面一致）。
func (r *ExpenseApplicationRepo) CreateWithItems(
	ctx context.Context,
	tid, uid uint32,
	title string,
	items []*oaV1.ExpenseItem,
) (uint32, float64, error) {
	total := 0.0
	for _, item := range items {
		total += item.GetAmount()
	}
	entity, err := r.entClient.Client().ExpenseApplication.Create().
		SetTitle(title).
		SetExpenseStatus(expenseapplication.ExpenseStatusPending).
		SetTotalAmount(total).
		SetTenantID(tid).
		SetCreatedBy(uid).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert expense application failed: %s", err.Error())
		return 0, 0, oaV1.ErrorInternalServerError("insert expense application failed")
	}
	for _, item := range items {
		ib := r.entClient.Client().ExpenseItem.Create().
			SetApplicationID(entity.ID).
			SetCategory(item.GetCategory()).
			SetAmount(item.GetAmount()).
			SetDescription(item.GetDescription()).
			SetTenantID(tid).
			SetCreatedAt(time.Now())
		if item.GetExpenseDate() != nil {
			ib.SetExpenseDate(item.GetExpenseDate().AsTime())
		}
		if item.GetInvoiceFileId() != 0 {
			ib.SetInvoiceFileID(item.GetInvoiceFileId())
		}
		if _, err := ib.Save(ctx); err != nil {
			r.log.Errorf("insert expense item failed: %s", err.Error())
			return 0, 0, oaV1.ErrorInternalServerError("insert expense item failed")
		}
	}
	return entity.ID, total, nil
}

// Get 按主键读取（WithItems 边加载明细）。
func (r *ExpenseApplicationRepo) Get(ctx context.Context, tid, id uint32) (*oaV1.ExpenseApplication, error) {
	entity, err := r.entClient.Client().ExpenseApplication.Query().
		Where(
			expenseapplication.IDEQ(id),
			expenseapplication.TenantIDEQ(tid),
		).
		WithItems().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, oaV1.ErrorNotFound("expense application not found")
		}
		r.log.Errorf("query expense application failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query expense application failed")
	}
	return expenseApplicationToDTO(entity), nil
}

func (r *ExpenseApplicationRepo) UpdateStatus(ctx context.Context, tid, id uint32, status oaV1.ExpenseApplication_ExpenseStatus) error {
	if _, err := r.entClient.Client().ExpenseApplication.Update().
		Where(
			expenseapplication.IDEQ(id),
			expenseapplication.TenantIDEQ(tid),
		).
		SetExpenseStatus(expenseStatusToEntity(status)).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("update expense application status failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("update expense application status failed")
	}
	return nil
}

func (r *ExpenseApplicationRepo) SetInstanceID(ctx context.Context, tid, id, instanceID uint32) error {
	if _, err := r.entClient.Client().ExpenseApplication.Update().
		Where(
			expenseapplication.IDEQ(id),
			expenseapplication.TenantIDEQ(tid),
		).
		SetInstanceID(instanceID).
		SetUpdatedAt(time.Now()).
		Save(ctx); err != nil {
		r.log.Errorf("link expense application instance failed: %s", err.Error())
		return oaV1.ErrorInternalServerError("link expense application instance failed")
	}
	return nil
}

// List 查询申请单。userID==0 时查全部（admin）；status 非零值时按状态过滤。
// pageSize>0 时分页（page 从 1 起）并返回真实总数，否则全量（total=条数）。
func (r *ExpenseApplicationRepo) List(
	ctx context.Context,
	tid, userID uint32,
	status oaV1.ExpenseApplication_ExpenseStatus,
	page, pageSize int32,
) ([]*oaV1.ExpenseApplication, int, error) {
	query := r.entClient.Client().ExpenseApplication.Query().
		Where(expenseapplication.TenantIDEQ(tid))
	if userID != 0 {
		query = query.Where(expenseapplication.CreatedByEQ(userID))
	}
	if status != 0 {
		query = query.Where(expenseapplication.ExpenseStatusEQ(expenseStatusToEntity(status)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count expense applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("count expense applications failed")
	}
	if pageSize > 0 {
		if page < 1 {
			page = 1
		}
		query = query.Offset((int(page) - 1) * int(pageSize)).Limit(int(pageSize))
	}
	// WithItems 预载明细（列表 DTO 读取 Edges.Items）。
	entities, err := query.Order(ent.Desc(expenseapplication.FieldID)).WithItems().All(ctx)
	if err != nil {
		r.log.Errorf("list expense applications failed: %s", err.Error())
		return nil, 0, oaV1.ErrorInternalServerError("list expense applications failed")
	}
	items := make([]*oaV1.ExpenseApplication, 0, len(entities))
	for _, e := range entities {
		items = append(items, expenseApplicationToDTO(e))
	}
	return items, total, nil
}

// GetEntity 供工作流终结回调读取原始实体（校验关联与状态同步）。不存在返回 nil。
func (r *ExpenseApplicationRepo) GetEntity(ctx context.Context, tid, id uint32) (*ent.ExpenseApplication, error) {
	entity, err := r.entClient.Client().ExpenseApplication.Query().
		Where(
			expenseapplication.IDEQ(id),
			expenseapplication.TenantIDEQ(tid),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query expense application entity failed: %s", err.Error())
		return nil, oaV1.ErrorInternalServerError("query expense application failed")
	}
	return entity, nil
}
