package service

import (
	"time"

	"task203-radarqc/internal/calibration"
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// StationService 雷达站业务服务。
type StationService struct {
	store *store.Store
	cal   *calibration.Manager
}

// CreateStationInput 创建站点入参。
type CreateStationInput struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Band      string  `json:"band"`
}

// Create 创建站点（初始状态 active）。
func (s *StationService) Create(in CreateStationInput) (*model.RadarStation, error) {
	if in.Name == "" {
		return nil, model.NewInvalidInput("站点名称不能为空")
	}
	if in.Latitude < -90 || in.Latitude > 90 {
		return nil, model.NewInvalidInput("latitude 必须在 [-90,90] 范围，得到 %.4f", in.Latitude)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return nil, model.NewInvalidInput("longitude 必须在 [-180,180] 范围，得到 %.4f", in.Longitude)
	}
	if in.Altitude < -500 || in.Altitude > 10000 {
		return nil, model.NewInvalidInput("altitude 必须在 [-500,10000] 米范围，得到 %.1f", in.Altitude)
	}
	if in.Band == "" {
		in.Band = "C"
	}
	now := time.Now().UTC()
	st := &model.RadarStation{
		ID:        "st-" + newSuffix(),
		Name:      in.Name,
		Latitude:  in.Latitude,
		Longitude: in.Longitude,
		Altitude:  in.Altitude,
		Band:      in.Band,
		Status:    model.StationActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Stations.Create(st); err != nil {
		return nil, err
	}
	return st, nil
}

// Get 查询站点。
func (s *StationService) Get(id string) (*model.RadarStation, error) {
	return s.store.Stations.Get(id)
}

// List 列出全部站点。
func (s *StationService) List() ([]*model.RadarStation, error) {
	return s.store.Stations.List()
}

// Transition 站点状态机流转：active↔calibrating、disabled→calibrating 等。
func (s *StationService) Transition(id string, to model.StationStatus) (*model.RadarStation, error) {
	st, err := s.store.Stations.Get(id)
	if err != nil {
		return nil, err
	}
	if st.Status == to {
		return st, nil
	}
	allowed := model.ValidStationTransitions(st.Status)
	for _, a := range allowed {
		if a == to {
			if err := s.store.Stations.UpdateStatus(id, to); err != nil {
				return nil, err
			}
			st.Status = to
			st.UpdatedAt = time.Now().UTC()
			return st, nil
		}
	}
	return nil, model.NewConflict("站点 %s 状态 %s 不能流转到 %s", id, st.Status, to)
}

// CreateCalibration 为站点创建校准草稿。
func (s *StationService) CreateCalibration(stationID string, zdrBias, zhOffset, rhohvBias float64, note string) (*model.Calibration, error) {
	return s.cal.CreateDraft(stationID, zdrBias, zhOffset, rhohvBias, note)
}

// ListCalibrations 列出站点校准版本。
func (s *StationService) ListCalibrations(stationID string) ([]*model.Calibration, error) {
	return s.store.Calibrations.ListByStation(stationID)
}

// PublishCalibration 发布校准版本。
func (s *StationService) PublishCalibration(id string) (*model.Calibration, error) {
	return s.cal.Publish(id)
}
