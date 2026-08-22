package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-oa/app/core/service/internal/data"

	permissionV1 "go-wind-oa/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-utils/trans"

	"go-wind-oa/pkg/constants"
	"go-wind-oa/pkg/metadata"
	appViewer "go-wind-oa/pkg/entgo/viewer"
)

type MenuService struct {
	permissionV1.UnimplementedMenuServiceServer

	log *log.Helper

	menuRepo *data.MenuRepo
}

func NewMenuService(ctx *bootstrap.Context, menuRepo *data.MenuRepo) *MenuService {
	svc := &MenuService{
		log:      ctx.NewLoggerHelper("menu/service/core-service"),
		menuRepo: menuRepo,
	}

	svc.init()

	return svc
}

func (s *MenuService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.menuRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultMenus(ctx)
	}
}

func (s *MenuService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListMenuResponse, error) {
	ret, err := s.menuRepo.List(ctx, req, false)
	if err != nil {

		return nil, err
	}

	return ret, nil
}

func (s *MenuService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountMenuResponse, error) {
	count, err := s.menuRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &permissionV1.CountMenuResponse{
		Count: uint64(count),
	}, nil
}

func (s *MenuService) Get(ctx context.Context, req *permissionV1.GetMenuRequest) (*permissionV1.Menu, error) {
	ret, err := s.menuRepo.Get(ctx, req)
	if err != nil {

		return nil, err
	}

	return ret, nil
}

func (s *MenuService) Create(ctx context.Context, req *permissionV1.CreateMenuRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.menuRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MenuService) Update(ctx context.Context, req *permissionV1.UpdateMenuRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.menuRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MenuService) Delete(ctx context.Context, req *permissionV1.DeleteMenuRequest) (*emptypb.Empty, error) {
	if err := s.menuRepo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *MenuService) createDefaultMenus(ctx context.Context) error {
	for _, m := range constants.DefaultMenus {
		if err := s.menuRepo.Create(ctx, &permissionV1.CreateMenuRequest{Data: m}); err != nil {
			s.log.Errorf("create default menu err: %v", err)
			return err
		}
	}
	return nil
}

// SyncMenus 同步菜单（将前端传入的树形菜单递归插入数据库，先清空后重建）
func (s *MenuService) SyncMenus(ctx context.Context, req *permissionV1.SyncMenusRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := metadata.FromServerContext(ctx)
	if err != nil {
		return nil, permissionV1.ErrorUnauthorized("operator context missing")
	}

	// 清空现有菜单数据
	if err = s.menuRepo.Truncate(ctx); err != nil {
		return nil, err
	}

	// 递归插入树形菜单
	count, err := s.syncMenuTree(ctx, req.Items, nil, uint32(operator.GetUserId()))
	if err != nil {
		return nil, err
	}

	s.log.Infof("sync menus success, total: %d", count)

	return &emptypb.Empty{}, nil
}

// syncMenuTree 递归插入菜单树：先插入父节点拿到 ID，再设置子节点的 parent_id
func (s *MenuService) syncMenuTree(ctx context.Context, menus []*permissionV1.Menu, parentId *uint32, operatorId uint32) (int, error) {
	count := 0
	for _, m := range menus {
		if m == nil {
			continue
		}

		// 保存子节点引用后清除，避免写入
		children := m.Children
		m.Children = nil

		// 清除前端可能传入的 ID，由数据库自增生成
		m.Id = nil
		m.ParentId = parentId
		m.Module = trans.Ptr(constants.ComponentToModule(m.GetComponent()))
		m.CreatedBy = trans.Ptr(operatorId)
		m.UpdatedBy = nil

		// 插入当前节点，获取数据库生成的 ID
		created, err := s.menuRepo.CreateReturn(ctx, &permissionV1.CreateMenuRequest{Data: m})
		if err != nil {
			s.log.Errorf("sync menu failed, name: %s, err: %v", m.GetName(), err)
			return count, err
		}
		count++

		// 递归插入子节点
		if len(children) > 0 {
			n, err := s.syncMenuTree(ctx, children, created.Id, operatorId)
			if err != nil {
				return count, err
			}
			count += n
		}
	}

	return count, nil
}
