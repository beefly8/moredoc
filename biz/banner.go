package biz

import (
	"context"

	pb "moredoc/api/v1"
	"moredoc/middleware/auth"
	"moredoc/model"
	"moredoc/util"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BannerAPIService struct {
	pb.UnimplementedBannerAPIServer
	dbModel *model.DBModel
	logger  *zap.Logger
}

func NewBannerAPIService(dbModel *model.DBModel, logger *zap.Logger) (service *BannerAPIService) {
	return &BannerAPIService{dbModel: dbModel, logger: logger.Named("BannerAPIService")}
}

func (s *BannerAPIService) checkPermission(ctx context.Context) (*auth.UserClaims, error) {
	return checkGRPCPermission(s.dbModel, ctx)
}

// CreateBanner 创建轮播图
func (s *BannerAPIService) CreateBanner(ctx context.Context, req *pb.Banner) (*pb.Banner, error) {
	_, err := s.checkPermission(ctx)
	if err != nil {
		return nil, err
	}

	var banner model.Banner
	util.CopyStruct(req, &banner)
	err = s.dbModel.CreateBanner(&banner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	pbBanner := &pb.Banner{}
	util.CopyStruct(&banner, pbBanner)

	return pbBanner, nil
}

// UpdateBanner 更新轮播图
func (s *BannerAPIService) UpdateBanner(ctx context.Context, req *pb.Banner) (*emptypb.Empty, error) {
	_, err := s.checkPermission(ctx)
	if err != nil {
		return nil, err
	}

	var banner model.Banner
	util.CopyStruct(req, &banner)
	err = s.dbModel.UpdateBanner(&banner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *BannerAPIService) DeleteBanner(ctx context.Context, req *pb.DeleteBannerRequest) (*emptypb.Empty, error) {
	_, err := s.checkPermission(ctx)
	if err != nil {
		return nil, err
	}

	err = s.dbModel.DeleteBanner(req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *BannerAPIService) GetBanner(ctx context.Context, req *pb.GetBannerRequest) (*pb.Banner, error) {
	if _, errPermission := s.checkPermission(ctx); errPermission != nil {
		return nil, errPermission
	}

	banner, err := s.dbModel.GetBanner(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pbBanner := &pb.Banner{}
	util.CopyStruct(banner, pbBanner)
	return pbBanner, nil
}

// GetBanners 获取轮播图列表
func (s *BannerAPIService) ListBanner(ctx context.Context, req *pb.ListBannerRequest) (*pb.ListBannerReply, error) {
	var opt = &model.OptionGetBannerList{
		Page:         int(req.Page),
		Size:         int(req.Size_),
		WithCount:    true,
		SelectFields: []string{"id", "title", "path", "url"}, // 对于非权限用户，可查询的字段
		QueryIn:      make(map[string][]interface{}),
	}

	if len(req.Type) > 0 {
		opt.QueryIn["type"] = util.Slice2Interface(req.Type)
	}

	_, errPermission := s.checkPermission(ctx)
	if errPermission != nil {
		opt.QueryIn["enable"] = []interface{}{true} // 非权限用户，只能查询正常状态的轮播图
	} else {
		opt.SelectFields = req.Field // 权限用户，可查询指定字段
		if len(req.Enable) > 0 {
			opt.QueryIn["enable"] = util.Slice2Interface(req.Enable)
		}

		if req.Wd != "" {
			opt.QueryLike = map[string][]interface{}{"title": {req.Wd}, "description": {req.Wd}}
		}
	}

	banners, total, err := s.dbModel.GetBannerList(opt)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbBanner []*pb.Banner
	util.CopyStruct(banners, &pbBanner)

	return &pb.ListBannerReply{Total: total, Banner: pbBanner}, nil
}
