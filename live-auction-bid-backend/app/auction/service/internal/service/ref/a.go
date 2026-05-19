//go:build ignore
// +build ignore

package service

import (
	"context"
	v2 "odin/api/loli/service/v2"
	"odin/app/loli/service/internal/biz"

	"github.com/pkg/errors"
)

func (s *UserService) GetAirplaneInfo(ctx context.Context, req *v2.GetAirplaneInfoReq) (*v2.GetAirplaneInfoReply, error) {
	rv, err := s.uc.GetAirplaneInfo(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "GetAirplaneInfo service error")
	}
	return &v2.GetAirplaneInfoReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}

func (s *UserService) CreateAirplaneOrder(ctx context.Context, req *v2.CreateAirplaneOrderReq) (*v2.CreateAirplaneOrderReply, error) {
	rv, err := s.uc.CreateAirplaneOrder(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "CreateAirplaneOrder service error")
	}
	return &v2.CreateAirplaneOrderReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}

func (s *UserService) AirplaneStartLoad(ctx context.Context, req *v2.AirplaneStartLoadReq) (*v2.AirplaneStartLoadReply, error) {
	rv, err := s.uc.AirplaneStartLoad(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "AirplaneStartLoad service error")
	}
	return &v2.AirplaneStartLoadReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}

func (s *UserService) FillAirplaneOrder(ctx context.Context, req *v2.FillAirplaneOrderReq) (*v2.FillAirplaneOrderReply, error) {
	rv, err := s.uc.FillAirplaneOrder(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
		Data:      req.Data,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "FillAirplaneOrder service error")
	}
	return &v2.FillAirplaneOrderReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}

func (s *UserService) CollectAirplane(ctx context.Context, req *v2.CollectAirplaneReq) (*v2.CollectAirplaneReply, error) {
	rv, err := s.uc.CollectAirplane(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
		Data:      req.Data,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "CollectAirplane service error")
	}
	return &v2.CollectAirplaneReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}

func (s *UserService) AccAirplaneOrder(ctx context.Context, req *v2.AccAirplaneOrderReq) (*v2.AccAirplaneOrderReply, error) {
	rv, err := s.uc.AccAirplaneOrder(ctx, &biz.User{
		Fid:       req.Head.Fid,
		Yid:       req.Head.Yid,
		Ts:        req.Head.Ts,
		Sts:       req.Head.Sts,
		Sign:      req.Head.Sign,
		Nonce:     req.Head.Nonce,
		AppVer:    req.Head.Version,
		Platform:  req.Head.Platform,
		Etc:       req.Head.Etc,
		BasicInfo: req.Head.Basicinfo,
		Data:      req.Data,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "AccAirplaneOrder service error")
	}
	return &v2.AccAirplaneOrderReply{
		Ts:   rv.Ts,
		Data: rv.Data,
	}, err
}
